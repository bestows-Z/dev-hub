package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Store struct {
	client *minio.Client
	bucket string
}

func New(cfg config.StorageConfig) (*Store, error) {
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("MinIO access key and secret are required")
	}
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize MinIO client: %w", err)
	}
	return &Store{client: client, bucket: cfg.Bucket}, nil
}

func (s *Store) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Put(ctx context.Context, key string, body []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(body), int64(len(body)), minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, int64, string, error) {
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, "", err
	}
	info, err := object.Stat()
	if err != nil {
		_ = object.Close()
		return nil, 0, "", err
	}
	return object, info.Size, info.ContentType, nil
}

func (s *Store) RemovePrefix(ctx context.Context, prefix string) error {
	for item := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: prefix + "/", Recursive: true}) {
		if item.Err != nil {
			return item.Err
		}
		if err := s.client.RemoveObject(ctx, s.bucket, item.Key, minio.RemoveObjectOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func IsNotFound(err error) bool {
	code := minio.ToErrorResponse(err).Code
	return strings.EqualFold(code, "NoSuchKey") || strings.EqualFold(code, "NoSuchBucket") || strings.EqualFold(code, "NotFound")
}
