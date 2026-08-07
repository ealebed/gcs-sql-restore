package restore

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// ErrSkip indicates the event should be ignored (not a failure).
var ErrSkip = errors.New("skip event")

// Config holds restore orchestrator settings.
type Config struct {
	ProjectID      string
	InstanceID     string
	DatabaseName   string
	ImportedPrefix string
	PollInterval   time.Duration
	PollTimeout    time.Duration
	// Now is optional; defaults to time.Now. Override in tests.
	Now func() time.Time
}

// Operation is a Cloud SQL long-running operation identifier.
type Operation struct {
	Name string
	Done bool
	Err  error
}

// SQLAdmin is the Cloud SQL Admin surface needed for wipe + import.
type SQLAdmin interface {
	DatabaseExists(ctx context.Context, projectID, instanceID, database string) (bool, error)
	DeleteDatabase(ctx context.Context, projectID, instanceID, database string) (*Operation, error)
	CreateDatabase(ctx context.Context, projectID, instanceID, database string) (*Operation, error)
	ImportSQL(ctx context.Context, projectID, instanceID, database, gcsURI string) (*Operation, error)
	GetOperation(ctx context.Context, projectID, operationName string) (*Operation, error)
}

// Service orchestrates destructive restore via Cloud SQL Import.
type Service struct {
	cfg    Config
	admin  SQLAdmin
	store  ObjectStore
	logger *slog.Logger
}

// NewService constructs a restore Service.
func NewService(cfg *Config, admin SQLAdmin, store ObjectStore, logger *slog.Logger) (*Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	c := *cfg
	if c.ProjectID == "" {
		return nil, fmt.Errorf("project id is required")
	}
	if c.InstanceID == "" {
		return nil, fmt.Errorf("instance id is required")
	}
	if c.DatabaseName == "" {
		return nil, fmt.Errorf("database name is required")
	}
	if admin == nil {
		return nil, fmt.Errorf("sql admin client is required")
	}
	if store == nil {
		return nil, fmt.Errorf("object store is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	if c.ImportedPrefix == "" {
		c.ImportedPrefix = DefaultImportedPrefix
	}
	if c.PollInterval <= 0 {
		c.PollInterval = 5 * time.Second
	}
	if c.PollTimeout <= 0 {
		c.PollTimeout = 55 * time.Minute
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return &Service{cfg: c, admin: admin, store: store, logger: logger}, nil
}

// RestoreObject wipes/creates the target database, imports the dump, then archives the object.
func (s *Service) RestoreObject(ctx context.Context, obj ObjectRef) error {
	if IsUnderPrefix(obj.Name, s.cfg.ImportedPrefix) {
		return fmt.Errorf("%w: object %q is under archive prefix %q", ErrSkip, obj.Name, s.cfg.ImportedPrefix)
	}
	if err := ValidateObject(obj); err != nil {
		return fmt.Errorf("%w: %v", ErrSkip, err)
	}

	s.logger.Info("starting destructive restore",
		"bucket", obj.Bucket,
		"object", obj.Name,
		"uri", obj.GCSURI(),
		"database", s.cfg.DatabaseName,
		"instance", s.cfg.InstanceID,
	)

	if err := s.wipeAndCreate(ctx); err != nil {
		return err
	}

	op, err := s.admin.ImportSQL(ctx, s.cfg.ProjectID, s.cfg.InstanceID, s.cfg.DatabaseName, obj.GCSURI())
	if err != nil {
		return fmt.Errorf("start import: %w", err)
	}
	s.logger.Info("import started", "operation", op.Name, "uri", obj.GCSURI())

	if err := s.waitOperation(ctx, op.Name); err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	dst := ArchiveObjectName(obj.Name, s.cfg.ImportedPrefix, s.cfg.Now())
	s.logger.Info("archiving dump after successful import",
		"from", obj.Name,
		"to", dst,
	)
	if err := s.store.MoveObject(ctx, obj.Bucket, obj.Name, dst); err != nil {
		return fmt.Errorf("archive dump after import: %w", err)
	}

	s.logger.Info("restore completed",
		"database", s.cfg.DatabaseName,
		"uri", obj.GCSURI(),
		"archived_to", fmt.Sprintf("gs://%s/%s", obj.Bucket, dst),
		"operation", op.Name,
	)
	return nil
}

func (s *Service) wipeAndCreate(ctx context.Context) error {
	exists, err := s.admin.DatabaseExists(ctx, s.cfg.ProjectID, s.cfg.InstanceID, s.cfg.DatabaseName)
	if err != nil {
		return fmt.Errorf("check database: %w", err)
	}
	if exists {
		s.logger.Info("deleting existing database", "database", s.cfg.DatabaseName)
		deleteOp, deleteErr := s.admin.DeleteDatabase(ctx, s.cfg.ProjectID, s.cfg.InstanceID, s.cfg.DatabaseName)
		if deleteErr != nil {
			return fmt.Errorf("delete database: %w", deleteErr)
		}
		if waitErr := s.waitOperation(ctx, deleteOp.Name); waitErr != nil {
			return fmt.Errorf("delete database operation: %w", waitErr)
		}
	}

	s.logger.Info("creating database", "database", s.cfg.DatabaseName)
	createOp, createErr := s.admin.CreateDatabase(ctx, s.cfg.ProjectID, s.cfg.InstanceID, s.cfg.DatabaseName)
	if createErr != nil {
		return fmt.Errorf("create database: %w", createErr)
	}
	if waitErr := s.waitOperation(ctx, createOp.Name); waitErr != nil {
		return fmt.Errorf("create database operation: %w", waitErr)
	}
	return nil
}

func (s *Service) waitOperation(ctx context.Context, operationName string) error {
	deadline := time.Now().Add(s.cfg.PollTimeout)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for operation %s after %s", operationName, s.cfg.PollTimeout)
		}

		op, err := s.admin.GetOperation(ctx, s.cfg.ProjectID, operationName)
		if err != nil {
			return fmt.Errorf("get operation %s: %w", operationName, err)
		}
		if op.Done {
			if op.Err != nil {
				return op.Err
			}
			return nil
		}

		timer := time.NewTimer(s.cfg.PollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
