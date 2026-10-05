package project

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

const MaxBundleBytes = 20 << 20

type PreviewFile struct {
	Path        string
	Body        []byte
	ContentType string
}

var contentTypes = map[string]string{
	".html": "text/html; charset=utf-8", ".htm": "text/html; charset=utf-8",
	".css": "text/css; charset=utf-8", ".js": "application/javascript; charset=utf-8",
	".mjs": "application/javascript; charset=utf-8", ".json": "application/json; charset=utf-8",
	".svg": "image/svg+xml", ".png": "image/png", ".jpg": "image/jpeg",
	".jpeg": "image/jpeg", ".webp": "image/webp", ".gif": "image/gif",
	".ico": "image/x-icon", ".woff": "font/woff", ".woff2": "font/woff2",
	".ttf": "font/ttf", ".wasm": "application/wasm", ".txt": "text/plain; charset=utf-8",
	".xml": "application/xml", ".webmanifest": "application/manifest+json",
	".mp4": "video/mp4", ".webm": "video/webm",
}

func ParseStaticBundle(blob []byte) ([]PreviewFile, error) {
	if len(blob) == 0 || len(blob) > MaxBundleBytes {
		return nil, errors.New("ZIP must be between 1 byte and 20 MiB")
	}
	reader, err := zip.NewReader(bytes.NewReader(blob), int64(len(blob)))
	if err != nil {
		return nil, fmt.Errorf("invalid ZIP: %w", err)
	}
	if len(reader.File) > 500 {
		return nil, errors.New("ZIP contains more than 500 entries")
	}
	indexPath := ""
	for _, file := range reader.File {
		if file.Name == "index.html" || strings.HasSuffix(file.Name, "/index.html") {
			if indexPath != "" {
				return nil, errors.New("ZIP must contain exactly one index.html")
			}
			indexPath = file.Name
		}
	}
	if indexPath == "" {
		return nil, errors.New("ZIP needs an index.html")
	}
	prefix := strings.TrimSuffix(indexPath, "index.html")
	files := make([]PreviewFile, 0, len(reader.File))
	seen := make(map[string]bool)
	var total uint64
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := file.Name
		if strings.Contains(name, `\`) || strings.HasPrefix(name, "/") || path.Clean(name) != name || strings.HasPrefix(name, "../") || !strings.HasPrefix(name, prefix) {
			return nil, fmt.Errorf("unsafe or unrelated ZIP path: %q", name)
		}
		if file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlinks are not allowed: %q", name)
		}
		name = strings.TrimPrefix(name, prefix)
		if name == "" || seen[name] {
			return nil, fmt.Errorf("duplicate or empty path: %q", name)
		}
		seen[name] = true
		for _, segment := range strings.Split(name, "/") {
			if strings.HasPrefix(segment, ".") {
				return nil, fmt.Errorf("hidden files are not allowed: %q", name)
			}
		}
		contentType, ok := contentTypes[strings.ToLower(path.Ext(name))]
		if !ok {
			return nil, fmt.Errorf("unsupported preview file type: %q", name)
		}
		if file.UncompressedSize64 > 10<<20 || total+file.UncompressedSize64 > 100<<20 {
			return nil, errors.New("ZIP expands beyond the 10 MiB per-file or 100 MiB total limit")
		}
		handle, err := file.Open()
		if err != nil {
			return nil, err
		}
		body, readErr := io.ReadAll(io.LimitReader(handle, 10<<20+1))
		closeErr := handle.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(body) > 10<<20 {
			return nil, errors.New("preview file exceeds 10 MiB")
		}
		total += uint64(len(body))
		if total > 100<<20 {
			return nil, errors.New("ZIP expands beyond 100 MiB")
		}
		files = append(files, PreviewFile{Path: name, Body: body, ContentType: contentType})
	}
	return files, nil
}
