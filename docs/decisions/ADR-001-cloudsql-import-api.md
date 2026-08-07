# ADR-001: Use Cloud SQL Import API instead of mysql client

## Status
Accepted

## Date
2026-08-07

## Context
The PoC must restore WordPress MySQL dumps (including multi-GB `.sql` / `.sql.gz`
files) from GCS into Cloud SQL MySQL 8.4 when an object is finalized. Two
approaches were considered:

1. Cloud SQL Admin API `instances.import` reading directly from `gs://`
2. A function/job that downloads the dump and loads it with a `mysql` client
   over public IP or Direct VPC

Sample phpMyAdmin dumps include site-specific names, e.g.:

```sql
CREATE DATABASE IF NOT EXISTS `test-wordpress` ...;
USE `test-wordpress`;
```

They do not use a fixed `wordpress` database name and do not include
`DROP TABLE` / wipe semantics.

## Decision
Use the Cloud SQL Import API. The Cloud Run Function only orchestrates
Admin API calls (start import with **no** `database` field, poll operation,
archive object). It never streams dump bytes.

The dump owns database creation and selection via `CREATE DATABASE IF NOT EXISTS`
and `USE`. The function does not delete or pre-create databases.

## Alternatives Considered

### Direct VPC + mysql client
- Pros: Full control over mysql flags; works if Import API rejects a dump shape
- Cons: Function must handle multi-GB I/O, VPC wiring, credentials, and longer
  failure modes; worse fit for a reliability demo
- Rejected for PoC; reserved as fallback if Import API proves insufficient

### Fixed DB wipe/create (`wordpress`) before import
- Pros: Predictable target name; clean slate each run
- Cons: Conflicts with real dumps that embed site-specific DB names; unnecessary
  when dumps already use `CREATE DATABASE IF NOT EXISTS`
- Rejected after inspecting sample dumps

### Manual `gcloud sql import sql` only
- Pros: Zero runtime code
- Cons: Does not demonstrate automated least-privilege trigger path for the team
- Rejected as the primary deliverable (still useful as a pre-flight check)

## Consequences
- Multi-GB dumps are feasible without sizing the function for data throughput
- IAM must grant the Cloud SQL instance service account `objectViewer` on the
  dump bucket
- Dumps must include `CREATE DATABASE` / `USE` (or import will have no target DB)
- Re-importing the same dump may fail if tables already exist (samples have
  `CREATE TABLE` without `DROP TABLE`) — operators should use a fresh DB name,
  drop tables manually, or export dumps with drops when re-loading
- Function timeout must cover import duration when polling (PoC uses up to 3600s)
