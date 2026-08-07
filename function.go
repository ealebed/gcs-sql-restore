// Package function is the Cloud Run Function entrypoint for GCS dump restores.
package function

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	cloudevent "github.com/cloudevents/sdk-go/v2/event"

	"github.com/ylebi/gcs-sql-restore/internal/restore"
)

func init() {
	functions.CloudEvent("RestoreSQLDump", restoreSQLDump)
}

// MessagePublishedData is the CloudEvent data for Pub/Sub messagePublished.
type MessagePublishedData struct {
	Message PubSubMessage `json:"message"`
}

// PubSubMessage is the Pub/Sub message envelope inside a CloudEvent.
type PubSubMessage struct {
	Data []byte `json:"data"`
}

func restoreSQLDump(ctx context.Context, event cloudevent.Event) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := configFromEnv()
	if err != nil {
		return err
	}

	var msg MessagePublishedData
	if dataErr := event.DataAs(&msg); dataErr != nil {
		return fmt.Errorf("decode cloud event: %w", dataErr)
	}

	obj, err := restore.ParseGCSNotification(msg.Message.Data)
	if err != nil {
		return fmt.Errorf("parse gcs notification: %w", err)
	}

	if restore.IsUnderPrefix(obj.Name, cfg.ImportedPrefix) {
		logger.Info("skipping archived object", "bucket", obj.Bucket, "object", obj.Name)
		return nil
	}
	if !restore.IsSQLDump(obj.Name) {
		logger.Info("skipping non-sql object", "bucket", obj.Bucket, "object", obj.Name)
		return nil
	}

	admin, err := restore.NewAPIClient(ctx)
	if err != nil {
		return err
	}
	store, err := restore.NewGCSStore(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = store.Close()
	}()

	svc, err := restore.NewService(&cfg, admin, store, logger)
	if err != nil {
		return err
	}

	if err := svc.RestoreObject(ctx, obj); err != nil {
		if errors.Is(err, restore.ErrSkip) {
			logger.Info("skipping object", "error", err.Error())
			return nil
		}
		return err
	}
	return nil
}

func configFromEnv() (restore.Config, error) {
	cfg := restore.Config{
		ProjectID:      os.Getenv("GCP_PROJECT"),
		InstanceID:     os.Getenv("CLOUDSQL_INSTANCE"),
		ImportedPrefix: os.Getenv("IMPORTED_PREFIX"),
		PollInterval:   5 * time.Second,
		PollTimeout:    55 * time.Minute,
	}
	if cfg.ProjectID == "" {
		cfg.ProjectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if cfg.ImportedPrefix == "" {
		cfg.ImportedPrefix = restore.DefaultImportedPrefix
	}
	if cfg.ProjectID == "" || cfg.InstanceID == "" {
		return restore.Config{}, fmt.Errorf("GCP_PROJECT (or GOOGLE_CLOUD_PROJECT) and CLOUDSQL_INSTANCE are required")
	}
	return cfg, nil
}
