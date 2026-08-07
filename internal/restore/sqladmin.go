package restore

import (
	"context"
	"fmt"
	"strings"

	sqladmin "google.golang.org/api/sqladmin/v1"
)

// APIClient implements SQLAdmin using the Cloud SQL Admin API.
type APIClient struct {
	svc *sqladmin.Service
}

// NewAPIClient creates an APIClient with Application Default Credentials.
func NewAPIClient(ctx context.Context) (*APIClient, error) {
	svc, err := sqladmin.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("create sqladmin client: %w", err)
	}
	return &APIClient{svc: svc}, nil
}

// ImportSQL starts a SQL import from a GCS URI.
// database may be empty when the dump itself contains CREATE DATABASE / USE
// (Cloud SQL Import overrides/ignores the API database field in that case).
func (c *APIClient) ImportSQL(ctx context.Context, projectID, instanceID, database, gcsURI string) (*Operation, error) {
	ctxImport := &sqladmin.ImportContext{
		FileType: "SQL",
		Uri:      gcsURI,
	}
	if database != "" {
		ctxImport.Database = database
	}
	op, err := c.svc.Instances.Import(projectID, instanceID, &sqladmin.InstancesImportRequest{
		ImportContext: ctxImport,
	}).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return mapOperation(op), nil
}

// GetOperation fetches a Cloud SQL operation by name.
func (c *APIClient) GetOperation(ctx context.Context, projectID, operationName string) (*Operation, error) {
	name := operationName
	if i := strings.LastIndex(operationName, "/"); i >= 0 {
		name = operationName[i+1:]
	}
	op, err := c.svc.Operations.Get(projectID, name).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return mapOperation(op), nil
}

func mapOperation(op *sqladmin.Operation) *Operation {
	out := &Operation{
		Name: op.Name,
		Done: op.Status == "DONE",
	}
	if op.Error != nil && len(op.Error.Errors) > 0 {
		parts := make([]string, 0, len(op.Error.Errors))
		for _, e := range op.Error.Errors {
			parts = append(parts, fmt.Sprintf("%s: %s", e.Code, e.Message))
		}
		out.Err = fmt.Errorf("cloud sql operation error: %s", strings.Join(parts, "; "))
	}
	return out
}
