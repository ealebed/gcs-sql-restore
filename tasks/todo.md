# gcs-sql-restore — Task Checklist

## Phase 1: Foundation
- [x] Task 1: Scaffold project (module, Makefile, lint, gitignore, pre-commit)
- [x] Task 2: Restore domain logic + unit tests

## Checkpoint: Foundation
- [x] Tests pass with race detector
- [x] Code formatted

## Phase 2: Function + Infra
- [x] Task 3: CloudEvents handler + function entrypoint
- [x] Task 4: Terraform core (APIs, bucket, Cloud SQL, IAM)
- [x] Task 5: Terraform Pub/Sub + Function
- [x] Task 6: README + ADRs + tfvars example

## Checkpoint: Complete
- [x] Docs + review done
- [x] Ready for manual `terraform apply` in `ylebi-rnd`

## Future improvements
- [ ] **Long-running import completion + archive:** Event-triggered functions cap at 540s. Today the function starts Import, polls ~8m, then acks without archiving if still running (Cloud SQL import continues). Add a follow-up path (e.g. Cloud Scheduler + Function/Job, or Workflow) that watches the Cloud SQL operation and moves the dump to `imported/` only after `DONE` success — so multi-GB restores get reliable auto-archive without Pub/Sub duplicate imports.
