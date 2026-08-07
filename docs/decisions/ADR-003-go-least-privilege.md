# ADR-003: Go Cloud Run Function with dedicated least-privilege SA

## Status
Accepted

## Date
2026-08-07

## Context
The runtime can be Go or Python. The workspace standardizes on Go 1.26, and
Cloud Run functions support the `go126` runtime. Identity should be a dedicated
service account with minimal roles for the internal team review.

## Decision
- Implement the function in Go with Functions Framework (CloudEvents)
- Deploy with runtime `go126`
- Use dedicated SA `gcs-sql-restore-fn` bound to custom role
  `gcsSqlRestoreOrchestrator` with only:
  - `cloudsql.instances.get` / `import`
  - `cloudsql.operations.get`
- Grant that SA `roles/storage.objectUser` on the dumps bucket so it can
  server-side move dumps into `imported/` after success (no dump bytes through
  the function)

Database create/delete permissions are intentionally omitted: phpMyAdmin-style
dumps include `CREATE DATABASE IF NOT EXISTS` / `USE` and own the DB name.

## Alternatives Considered

### Python + google-cloud-sql
- Pros: Fine Admin API support
- Cons: No advantage for this orchestration-only workload; mismatches workspace

### roles/cloudsql.admin on the function SA
- Pros: Faster to wire
- Cons: Far broader than needed for the least-privilege demo
- Rejected

## Consequences
- Custom role must be kept in sync if Admin API surface grows
- Cloud SQL instance SA still needs bucket read access (separate from function SA)
