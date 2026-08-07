package restore

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

// ObjectStore moves GCS objects (server-side copy + delete).
type ObjectStore interface {
	MoveObject(ctx context.Context, bucket, srcObject, dstObject string) error
}

// GCSStore implements ObjectStore with the GCS client.
type GCSStore struct {
	client *storage.Client
}

// NewGCSStore creates a GCSStore with Application Default Credentials.
func NewGCSStore(ctx context.Context) (*GCSStore, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}
	return &GCSStore{client: client}, nil
}

// Close releases the underlying storage client.
func (s *GCSStore) Close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}

// MoveObject copies src to dst within the same bucket, then deletes src.
// The copy is server-side and does not stream bytes through this process.
func (s *GCSStore) MoveObject(ctx context.Context, bucket, srcObject, dstObject string) error {
	if bucket == "" || srcObject == "" || dstObject == "" {
		return fmt.Errorf("bucket, src, and dst are required")
	}
	if srcObject == dstObject {
		return fmt.Errorf("src and dst object names are identical: %s", srcObject)
	}

	src := s.client.Bucket(bucket).Object(srcObject)
	dst := s.client.Bucket(bucket).Object(dstObject)
	if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
		return fmt.Errorf("copy gs://%s/%s → gs://%s/%s: %w", bucket, srcObject, bucket, dstObject, err)
	}
	if err := src.Delete(ctx); err != nil {
		return fmt.Errorf("delete gs://%s/%s after copy: %w", bucket, srcObject, err)
	}
	return nil
}
