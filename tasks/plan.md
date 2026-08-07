# Implementation Plan: gcs-sql-restore

## Overview
PoC that restores a WordPress MySQL dump from GCS into Cloud SQL (MySQL 8.4)
when an object is finalized. A Go Cloud Run Function (Gen2) orchestrates
Cloud SQL Admin API import (no dump streaming). GCS notifications publish to
Pub/Sub (DLQ included); the function is triggered from that topic. Target
project `ylebi-rnd`, region `europe-west3`, fixed DB name `wordpress`, public IP OK.

## Architecture Decisions
- **Import via Cloud SQL Admin API** — only path that scales to multi-GB without
  loading the dump into the function.
- **Destructive wipe** — delete DB if present, create `wordpress`, then import.
- **Pub/Sub on by default** (`enable_pubsub = true`) with Terraform toggle for
  a future direct Eventarc path.
- **Dedicated function SA + custom IAM role** — least privilege for demo story.
- **Cloud SQL instance SA → `roles/storage.objectViewer` on dump bucket** —
  required for Import to read `gs://`.
- **Go + Functions Framework (CloudEvents)** — fits workspace; Admin API client
  is mature. Runtime pinned to whatever Gen2 currently supports (see Task 1);
  local `go.mod` follows workspace `go 1.26.0` if runtime allows, otherwise
  the highest supported Functions runtime.
- **No CI/CD** — manual `terraform apply` only.

## Task List

### Phase 1: Foundation
- [ ] Task 1: Scaffold project layout, Go module, Makefile, lint/gitignore templates
- [ ] Task 2: Pure restore domain logic + unit tests (validate object, wipe/create/import orchestration behind interfaces)

### Checkpoint: Foundation
- [ ] `go test ./... -race` passes
- [ ] `gofmt -s` clean

### Phase 2: Function + Infra
- [ ] Task 3: CloudEvents/Pub/Sub handler wiring + main entrypoint
- [ ] Task 4: Terraform — providers, APIs, GCS bucket, Cloud SQL MySQL 8.4, IAM
- [ ] Task 5: Terraform — Pub/Sub + DLQ + GCS notification + Cloud Function Gen2
- [ ] Task 6: README + ADR(s) + example tfvars

### Checkpoint: Complete
- [ ] `terraform fmt` / validate (when credentials available)
- [ ] Unit tests pass; docs describe manual apply + verify steps
- [ ] Code review checklist completed

## Risks and Mitigations
| Risk | Impact | Mitigation |
|------|--------|------------|
| Multi-GB import exceeds function timeout while polling | Med | Long timeout (e.g. 60m); log operation name; document follow-up via `gcloud sql operations` |
| Dump contains `CREATE DATABASE` conflicting with wipe flow | Med | Document dump expectations; import targets `wordpress` |
| Overlapping OBJECT_FINALIZE events | Low | PoC accepts last-writer-wins; document single-flight |
| Functions Go runtime < workspace go 1.26 | Low | Pin deployable runtime; note in ADR |

## Open Questions
- None blocking — defaults confirmed by stakeholder.
