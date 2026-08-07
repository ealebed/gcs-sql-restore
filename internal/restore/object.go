package restore

import (
	"fmt"
	"path"
	"strings"
	"time"
)

// ObjectRef identifies a GCS object that may contain a SQL dump.
type ObjectRef struct {
	Bucket string
	Name   string
}

// GCSURI returns the gs:// URI for the object.
func (o ObjectRef) GCSURI() string {
	return fmt.Sprintf("gs://%s/%s", o.Bucket, o.Name)
}

// IsSQLDump reports whether the object name looks like a supported SQL dump.
// Supported suffixes: .sql and .sql.gz (case-insensitive).
func IsSQLDump(objectName string) bool {
	base := strings.ToLower(path.Base(objectName))
	if base == "" || base == "." || base == "/" {
		return false
	}
	return strings.HasSuffix(base, ".sql.gz") || strings.HasSuffix(base, ".sql")
}

// DefaultImportedPrefix is the GCS prefix used after a successful import.
const DefaultImportedPrefix = "imported"

// IsUnderPrefix reports whether objectName is already under prefix (e.g. imported/).
func IsUnderPrefix(objectName, prefix string) bool {
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return false
	}
	return objectName == prefix || strings.HasPrefix(objectName, prefix+"/")
}

// ArchiveObjectName builds the destination object key under prefix.
// Example: wordpress.sql.gz → imported/20060102T150405Z_wordpress.sql.gz
func ArchiveObjectName(objectName, prefix string, at time.Time) string {
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		prefix = DefaultImportedPrefix
	}
	stamp := at.UTC().Format("20060102T150405Z")
	return path.Join(prefix, stamp+"_"+path.Base(objectName))
}

// ValidateObject ensures the object reference is usable for import.
func ValidateObject(obj ObjectRef) error {
	if strings.TrimSpace(obj.Bucket) == "" {
		return fmt.Errorf("bucket is required")
	}
	if strings.TrimSpace(obj.Name) == "" {
		return fmt.Errorf("object name is required")
	}
	if strings.HasSuffix(obj.Name, "/") {
		return fmt.Errorf("object name %q looks like a directory prefix", obj.Name)
	}
	if !IsSQLDump(obj.Name) {
		return fmt.Errorf("object %q is not a .sql or .sql.gz dump", obj.Name)
	}
	return nil
}
