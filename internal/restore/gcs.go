package restore

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
)

// ObjectStore reads prefixes and moves GCS objects (server-side copy + delete).
type ObjectStore interface {
	ReadObjectPrefix(ctx context.Context, bucket, object string, maxBytes int64) ([]byte, error)
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

// ReadObjectPrefix returns up to maxBytes from the start of the object.
// For .gz / .sql.gz objects the bytes are gunzipped first (still only a small prefix).
func (s *GCSStore) ReadObjectPrefix(ctx context.Context, bucket, object string, maxBytes int64) ([]byte, error) {
	if bucket == "" || object == "" {
		return nil, fmt.Errorf("bucket and object are required")
	}
	if maxBytes <= 0 {
		maxBytes = DefaultPeekBytes
	}

	r, err := s.client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("open gs://%s/%s: %w", bucket, object, err)
	}
	defer func() { _ = r.Close() }()

	var reader io.Reader = r
	lower := strings.ToLower(object)
	if strings.HasSuffix(lower, ".gz") {
		gz, gzErr := gzip.NewReader(r)
		if gzErr != nil {
			return nil, fmt.Errorf("gzip open gs://%s/%s: %w", bucket, object, gzErr)
		}
		defer func() { _ = gz.Close() }()
		reader = gz
	}

	data, err := io.ReadAll(io.LimitReader(reader, maxBytes))
	if err != nil {
		return nil, fmt.Errorf("read prefix gs://%s/%s: %w", bucket, object, err)
	}
	return data, nil
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
