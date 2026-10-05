package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
)

func TestMinIOUploadAndRead(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_MINIO") != "1" {
		t.Skip("set DEVHUB_TEST_MINIO=1 for the local MinIO integration test")
	}
	store, err := New(config.StorageConfig{
		Endpoint:  os.Getenv("MINIO_ENDPOINT"),
		AccessKey: os.Getenv("MINIO_ROOT_USER"),
		SecretKey: os.Getenv("MINIO_ROOT_PASSWORD"),
		Bucket:    "devhub-projects-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := store.EnsureBucket(ctx); err != nil {
		t.Fatal(err)
	}
	prefix := fmt.Sprintf("tests/preview-%d", time.Now().UnixNano())
	defer func() {
		if err := store.RemovePrefix(ctx, prefix); err != nil {
			t.Errorf("cleanup MinIO test object: %v", err)
		}
	}()
	if err := store.Put(ctx, prefix+"/index.html", []byte("<h1>preview</h1>"), "text/html"); err != nil {
		t.Fatal(err)
	}
	object, size, contentType, err := store.Get(ctx, prefix+"/index.html")
	if err != nil {
		t.Fatal(err)
	}
	defer object.Close()
	body, err := io.ReadAll(object)
	if err != nil || size != int64(len(body)) || contentType != "text/html" || string(body) != "<h1>preview</h1>" {
		t.Fatalf("unexpected object: size=%d type=%s body=%q err=%v", size, contentType, body, err)
	}
}
