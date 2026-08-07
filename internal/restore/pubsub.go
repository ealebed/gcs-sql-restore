package restore

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GCSObjectNotification is the JSON_API_V1 payload published by GCS notifications.
type GCSObjectNotification struct {
	Bucket string `json:"bucket"`
	Name   string `json:"name"`
}

// ParseGCSNotification decodes a GCS object notification from Pub/Sub message data.
func ParseGCSNotification(data []byte) (ObjectRef, error) {
	if len(data) == 0 {
		return ObjectRef{}, fmt.Errorf("empty pub/sub message data")
	}
	var n GCSObjectNotification
	if err := json.Unmarshal(data, &n); err != nil {
		return ObjectRef{}, fmt.Errorf("decode gcs notification: %w", err)
	}
	if strings.TrimSpace(n.Bucket) == "" || strings.TrimSpace(n.Name) == "" {
		return ObjectRef{}, fmt.Errorf("gcs notification missing bucket or name")
	}
	return ObjectRef(n), nil
}
