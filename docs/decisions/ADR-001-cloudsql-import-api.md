# ADR-001: Use Cloud SQL Import API instead of mysql client

## Status
Accepted

## Date
2026-08-07

## Context
The PoC must restore WordPress MySQL dumps (including multi-GB `.sql` / `.sql.gz`
files) from GCS into Cloud SQL MySQL 8.4 when an object is finalized.

Sample phpMyAdmin dumps include site-specific names, e.g.:

```sql
CREATE DATABASE IF NOT EXISTS `camarotest2-wordpress` ...;
USE `camarotest2-wordpress`;
```

They do not include `DROP TABLE`. Re-importing into an existing DB fails with
`ERROR 1050 Table already exists` unless the target database is removed first.

## Decision
Use the Cloud SQL Import API. The Cloud Run Function:

1. Peeks a small GCS prefix (gunzip if needed) and parses `CREATE DATABASE` / `USE`
2. Drops that database via Admin API if it already exists (destructive fresh restore)
3. Starts `instances.import` with **no** `database` field (dump recreates DB + tables)
4. Polls the operation, then archives the object under `imported/`

It never streams full dump bytes through the function process.

## Alternatives Considered

### Direct VPC + mysql client
- Pros: Full control over mysql flags
- Cons: Multi-GB I/O, VPC wiring, credentials
- Rejected for PoC; reserved if Import API proves insufficient

### Drop tables instead of DROP DATABASE
- Pros: Keeps empty DB shell
- Cons: More surface area; incomplete if dump adds tables
- Rejected

### Import without pre-drop
- Pros: Simpler first version
- Cons: Second upload fails on existing tables (observed in PoC)
- Rejected after validation

## Consequences
- Multi-GB dumps stay feasible (peek + Admin API only)
- Function SA needs `databases.get` / `databases.delete` plus import/operations
- Cloud SQL instance SA needs `objectViewer` on the dumps bucket
- Dumps must include `CREATE DATABASE` / `USE` near the start of the file
- Every successful re-upload **wipes** the dump's target database (by design)
- Event-triggered functions cap at 540s; long imports may finish after ack without
  auto-archive (see future TODO)
