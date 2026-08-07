package restore_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/ylebi/gcs-sql-restore/internal/restore"
)

func TestIsSQLDump(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "sql", in: "wordpress.sql", want: true},
		{name: "sql gz", in: "path/wordpress.sql.gz", want: true},
		{name: "case", in: "Dump.SQL.GZ", want: true},
		{name: "csv", in: "data.csv", want: false},
		{name: "partial", in: "file.sql.bak", want: false},
		{name: "empty", in: "", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := restore.IsSQLDump(tc.in); got != tc.want {
				t.Fatalf("IsSQLDump(%q)=%v want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsUnderPrefix(t *testing.T) {
	t.Parallel()
	if !restore.IsUnderPrefix("imported/x.sql.gz", "imported") {
		t.Fatal("expected under prefix")
	}
	if restore.IsUnderPrefix("wordpress.sql.gz", "imported") {
		t.Fatal("did not expect under prefix")
	}
}

func TestArchiveObjectName(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 8, 7, 12, 30, 0, 0, time.UTC)
	got := restore.ArchiveObjectName("path/wordpress.sql.gz", "imported", at)
	want := "imported/20260807T123000Z_wordpress.sql.gz"
	if got != want {
		t.Fatalf("ArchiveObjectName=%q want %q", got, want)
	}
}

func TestValidateObject(t *testing.T) {
	t.Parallel()
	err := restore.ValidateObject(restore.ObjectRef{Bucket: "b", Name: "x.txt"})
	if err == nil {
		t.Fatal("expected error for non-sql object")
	}
	err = restore.ValidateObject(restore.ObjectRef{Bucket: "b", Name: "ok.sql.gz"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseGCSNotification(t *testing.T) {
	t.Parallel()
	obj, err := restore.ParseGCSNotification([]byte(`{"bucket":"dumps","name":"wordpress.sql.gz"}`))
	if err != nil {
		t.Fatalf("ParseGCSNotification: %v", err)
	}
	if obj.Bucket != "dumps" || obj.Name != "wordpress.sql.gz" {
		t.Fatalf("unexpected object: %+v", obj)
	}
	if obj.GCSURI() != "gs://dumps/wordpress.sql.gz" {
		t.Fatalf("unexpected uri: %s", obj.GCSURI())
	}
}

func TestRestoreObjectHappyPath(t *testing.T) {
	t.Parallel()
	admin := &fakeSQLAdmin{
		ops: map[string]*restore.Operation{
			"op-import": {Name: "op-import", Done: true},
		},
	}
	store := &fakeObjectStore{}
	fixed := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	svc, err := restore.NewService(&restore.Config{
		ProjectID:      "proj",
		InstanceID:     "inst",
		ImportedPrefix: "imported",
		PollInterval:   time.Millisecond,
		PollTimeout:    time.Second,
		Now:            func() time.Time { return fixed },
	}, admin, store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	err = svc.RestoreObject(context.Background(), restore.ObjectRef{Bucket: "b", Name: "wp.sql.gz"})
	if err != nil {
		t.Fatalf("RestoreObject: %v", err)
	}
	if !admin.imported {
		t.Fatal("expected import")
	}
	if admin.importDB != "" {
		t.Fatalf("expected empty import database (dump-owned), got %q", admin.importDB)
	}
	if store.src != "wp.sql.gz" || store.dst != "imported/20260807T120000Z_wp.sql.gz" {
		t.Fatalf("unexpected archive move: %s → %s", store.src, store.dst)
	}
}

func TestRestoreObjectSkipsNonSQL(t *testing.T) {
	t.Parallel()
	admin := &fakeSQLAdmin{}
	svc, err := restore.NewService(&restore.Config{
		ProjectID:  "proj",
		InstanceID: "inst",
	}, admin, &fakeObjectStore{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	err = svc.RestoreObject(context.Background(), restore.ObjectRef{Bucket: "b", Name: "readme.md"})
	if !errors.Is(err, restore.ErrSkip) {
		t.Fatalf("expected ErrSkip, got %v", err)
	}
}

func TestRestoreObjectSkipsArchivedPrefix(t *testing.T) {
	t.Parallel()
	admin := &fakeSQLAdmin{}
	svc, err := restore.NewService(&restore.Config{
		ProjectID:      "proj",
		InstanceID:     "inst",
		ImportedPrefix: "imported",
	}, admin, &fakeObjectStore{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	err = svc.RestoreObject(context.Background(), restore.ObjectRef{
		Bucket: "b",
		Name:   "imported/20260807T120000Z_wp.sql.gz",
	})
	if !errors.Is(err, restore.ErrSkip) {
		t.Fatalf("expected ErrSkip, got %v", err)
	}
	if admin.imported {
		t.Fatal("must not import archived objects")
	}
}

type fakeSQLAdmin struct {
	mu       sync.Mutex
	imported bool
	importDB string
	ops      map[string]*restore.Operation
}

func (f *fakeSQLAdmin) ImportSQL(_ context.Context, _, _, database, uri string) (*restore.Operation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.imported = true
	f.importDB = database
	if uri == "" {
		return nil, errors.New("empty uri")
	}
	return &restore.Operation{Name: "op-import"}, nil
}

func (f *fakeSQLAdmin) GetOperation(_ context.Context, _, name string) (*restore.Operation, error) {
	op, ok := f.ops[name]
	if !ok {
		return &restore.Operation{Name: name, Done: true}, nil
	}
	return op, nil
}

type fakeObjectStore struct {
	mu  sync.Mutex
	src string
	dst string
}

func (f *fakeObjectStore) MoveObject(_ context.Context, _, srcObject, dstObject string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.src = srcObject
	f.dst = dstObject
	return nil
}
