package project

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testRuntimeZIP(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func validRuntimeFiles() map[string]string {
	return map[string]string{
		"docker-compose.yml":  "services:\n  frontend:\n    build: ./frontend\n  backend:\n    build: ./backend\n",
		"frontend/Dockerfile": "FROM node:20-alpine\n",
		"backend/Dockerfile":  "FROM node:20-alpine\n",
		"frontend/index.html": "<h1>hello</h1>",
	}
}

func TestRuntimeBundleExtractsRequiredStructure(t *testing.T) {
	blob := testRuntimeZIP(t, validRuntimeFiles())
	if err := ValidateRuntimeBundle(blob); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := ExtractRuntimeBundle(blob, directory); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(directory, "frontend", "index.html"))
	if err != nil || string(content) != "<h1>hello</h1>" {
		t.Fatalf("content=%q err=%v", content, err)
	}
}

func TestRuntimeBundleRejectsUnsafePathsAndCompose(t *testing.T) {
	for _, test := range []struct{ name, path, content, want string }{
		{"traversal", "frontend/../../secrets", "x", "unsafe"},
		{"outside", "secrets.txt", "x", "outside"},
		{"extra service", "docker-compose.yml", "services:\n  frontend:\n    build: ./frontend\n  backend:\n    build: ./backend\n  database:\n    image: postgres\n", "compose"},
	} {
		t.Run(test.name, func(t *testing.T) {
			files := validRuntimeFiles()
			files[test.path] = test.content
			err := ValidateRuntimeBundle(testRuntimeZIP(t, files))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("err=%v, want %q", err, test.want)
			}
		})
	}
}
