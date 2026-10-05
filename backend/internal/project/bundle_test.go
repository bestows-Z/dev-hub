package project

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func makeBundle(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, body := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestParseStaticBundleAcceptsDistFolder(t *testing.T) {
	files, err := ParseStaticBundle(makeBundle(t, map[string]string{
		"dist/index.html":    "<script src=\"./assets/app.js\"></script>",
		"dist/assets/app.js": "console.log('hello')",
	}))
	if err != nil || len(files) != 2 {
		t.Fatalf("files=%v err=%v", files, err)
	}
	for _, file := range files {
		if strings.HasPrefix(file.Path, "dist/") {
			t.Fatalf("root folder was not stripped: %q", file.Path)
		}
	}
}

func TestParseStaticBundleRejectsTraversalAndSecrets(t *testing.T) {
	for _, name := range []string{"../secret.txt", ".env", "assets/config.yml"} {
		_, err := ParseStaticBundle(makeBundle(t, map[string]string{"index.html": "ok", name: "secret"}))
		if err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}
