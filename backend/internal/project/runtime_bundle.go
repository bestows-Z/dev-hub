package project

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

const MaxRuntimeBundleBytes = 50 << 20
const maxRuntimeFileBytes = 50 << 20
const maxRuntimeExpandedBytes = 250 << 20

func runtimeBundleReader(blob []byte) (*zip.Reader, error) {
	if len(blob) == 0 || len(blob) > MaxRuntimeBundleBytes {
		return nil, errors.New("runtime ZIP must be between 1 byte and 50 MiB")
	}
	reader, err := zip.NewReader(bytes.NewReader(blob), int64(len(blob)))
	if err != nil {
		return nil, fmt.Errorf("invalid runtime ZIP: %w", err)
	}
	if len(reader.File) > 5000 {
		return nil, errors.New("runtime ZIP has more than 5000 entries")
	}
	return reader, nil
}

func runtimePath(file *zip.File) (string, error) {
	name := strings.TrimSuffix(file.Name, "/")
	if name == "" || strings.ContainsAny(name, "\\\x00") || strings.HasPrefix(name, "/") || path.Clean(name) != name || strings.HasPrefix(name, "../") {
		return "", fmt.Errorf("unsafe runtime ZIP path: %q", file.Name)
	}
	if mode := file.FileInfo().Mode(); mode&os.ModeType != 0 && !mode.IsDir() {
		return "", fmt.Errorf("special file is not allowed: %q", name)
	}
	if name != "docker-compose.yml" && name != "README.md" && !strings.HasPrefix(name, "frontend/") && !strings.HasPrefix(name, "backend/") && name != "frontend" && name != "backend" {
		return "", fmt.Errorf("file is outside required directories: %q", name)
	}
	if (name == "frontend" || name == "backend") && !file.FileInfo().IsDir() {
		return "", fmt.Errorf("%q must be a directory", name)
	}
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") && part != ".dockerignore" {
			return "", fmt.Errorf("hidden file is not allowed: %q", name)
		}
	}
	if file.UncompressedSize64 > maxRuntimeFileBytes {
		return "", fmt.Errorf("runtime file too large: %q", name)
	}
	return name, nil
}

func ValidateRuntimeBundle(blob []byte) error {
	reader, err := runtimeBundleReader(blob)
	if err != nil {
		return err
	}
	seen := make(map[string]bool)
	var total uint64
	var compose []byte
	for _, file := range reader.File {
		name, err := runtimePath(file)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			continue
		}
		folded := strings.ToLower(name)
		if seen[folded] {
			return fmt.Errorf("duplicate runtime ZIP path: %q", name)
		}
		seen[folded] = true
		if name == "docker-compose.yml" && file.UncompressedSize64 > 64<<10 {
			return errors.New("docker-compose.yml is too large")
		}
		handle, err := file.Open()
		if err != nil {
			return err
		}
		var count int64
		if name == "docker-compose.yml" {
			compose, err = io.ReadAll(io.LimitReader(handle, 64<<10+1))
			count = int64(len(compose))
		} else {
			count, err = io.Copy(io.Discard, io.LimitReader(handle, maxRuntimeFileBytes+1))
		}
		closeErr := handle.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if count > maxRuntimeFileBytes || (name == "docker-compose.yml" && count > 64<<10) {
			return fmt.Errorf("runtime file too large: %q", name)
		}
		total += uint64(count)
		if total > maxRuntimeExpandedBytes {
			return errors.New("runtime ZIP expands beyond 250 MiB")
		}
	}
	for _, required := range []string{"docker-compose.yml", "frontend/Dockerfile", "backend/Dockerfile"} {
		if !seen[strings.ToLower(required)] {
			return fmt.Errorf("runtime ZIP needs %s", required)
		}
	}
	var settings struct {
		Services map[string]struct {
			Build string `yaml:"build"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(compose, &settings); err != nil {
		return fmt.Errorf("invalid docker-compose.yml: %w", err)
	}
	if len(settings.Services) != 2 || settings.Services["frontend"].Build != "./frontend" || settings.Services["backend"].Build != "./backend" {
		return errors.New("compose must contain only frontend and backend, built from ./frontend and ./backend")
	}
	return nil
}

func ExtractRuntimeBundle(blob []byte, directory string) error {
	if err := ValidateRuntimeBundle(blob); err != nil {
		return err
	}
	reader, _ := runtimeBundleReader(blob)
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name, _ := runtimePath(file)
		target := filepath.Join(directory, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		input, err := file.Open()
		if err != nil {
			_ = output.Close()
			return err
		}
		_, copyErr := io.Copy(output, io.LimitReader(input, maxRuntimeFileBytes+1))
		inputErr := input.Close()
		outputErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputErr != nil {
			return inputErr
		}
		if outputErr != nil {
			return outputErr
		}
	}
	return nil
}
