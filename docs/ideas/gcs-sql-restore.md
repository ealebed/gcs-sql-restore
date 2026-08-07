# gcs-sql-restore

## Problem Statement
How might we let an internal platform team prove that uploading a WordPress
MySQL dump to GCS automatically restores a Cloud SQL 8.4 database, with
least-privilege identity and a retry-friendly trigger path?

## Recommended Direction
Build a small Go Cloud Run Function that never downloads the dump. On
object finalize for `.sql` / `.sql.gz`, Pub/Sub delivers the event to the
function. The function:

1. Validates object name/type
2. Calls Cloud SQL Admin API `instances.import` with `gs://…` (no fixed
   `database` — dump includes `CREATE DATABASE IF NOT EXISTS` / `USE`)
3. Polls the long-running Operation until success/failure
4. Moves the object under `imported/` and emits structured logs

Infra is Terraform: GCS bucket, Pub/Sub (+ DLQ), Eventarc/GCS notification,
Cloud SQL MySQL 8.4, dedicated function service account with minimal roles,
and Cloud SQL instance SA granted objectViewer on the dump bucket.

`enable_pubsub` stays in the module (default `true`) so the team can
compare the direct-trigger path later without a redesign.

Language: Go (Admin API orchestration; fits this workspace).

### Confirmed PoC defaults
- Target database name: `wordpress`
- Cloud SQL public IP: allowed for PoC
- Region: `europe-west3`
- GCP project: `ylebi-rnd`

## Key Assumptions to Validate
- [ ] Sample WordPress `.sql.gz` imports cleanly via Cloud SQL Import API
      — test once with `gcloud sql import sql` before wiring the function
- [ ] Multi-GB import completes within the function’s configured timeout
      when polling — measure with a large dump; if not, switch poll to
      “start + log operation id” for follow-up
- [ ] Delete + create database is accepted wipe semantics for the use case
      — confirm with the team in the PoC review

## MVP Scope
**In**
- Terraform: bucket, MySQL 8.4 instance, DB name variable, Pub/Sub + DLQ,
  Cloud Run Function (Gen2), dedicated SA + IAM bindings, Eventarc/GCS→Pub/Sub
- Go function: filter, wipe/create DB, import, poll, structured logs
- README: apply steps, upload test, how to verify tables

**Out**
- CI/CD, multi-env, Slack/email alerts, parallel restores, Direct VPC mysql
  client path, WordPress app tier, dump validation beyond extension/name

## Not Doing (and Why)
- Streaming dump through the function — breaks on multi-GB; Import API is the product
- Direct VPC + mysql client — reserved only if Import API fails later
- Full status UI / job store — one-shot logs are enough for PoC approval
- Soft/merge restores — destructive wipe is the explicit contract
- CI/CD pipelines — manual `terraform apply` per request

## Open Questions
- ~~Fixed target DB name vs derive from object path?~~ → fixed `wordpress`
- ~~GCP region / project id?~~ → `europe-west3` / `ylebi-rnd`
- ~~Public IP for PoC?~~ → yes
