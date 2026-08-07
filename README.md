# gcs-sql-restore

PoC: upload a WordPress/phpMyAdmin MySQL dump (`.sql` / `.sql.gz`) to GCS →
Pub/Sub → Cloud Run Function (Go) → Cloud SQL Admin **Import** into MySQL 8.4.

The dump owns the database name via `CREATE DATABASE IF NOT EXISTS` + `USE`
(e.g. `camarotest2-wordpress`). The function does **not** wipe or pre-create a
fixed DB. It never downloads the dump — that is what makes multi-GB restores
viable.

## Architecture

See the Mermaid diagrams in [docs/architecture.md](docs/architecture.md).

```
GCS OBJECT_FINALIZE
        │
        ▼
   Pub/Sub topic  ──(Eventarc retry)──► Cloud Run Function (Go)
        │                                      │
        └── DLQ topic (ops / future)           ├── skip imported/ and non-SQL
                                               ├── instances.import(gs://…)  [no database=]
                                               └── move object → imported/<ts>_file
                                                         │
                                                         ▼
                                              Cloud SQL MySQL 8.4
                                         (CREATE DATABASE / USE from dump)
```

Decisions: [docs/decisions/](docs/decisions/) · Idea brief: [docs/ideas/gcs-sql-restore.md](docs/ideas/gcs-sql-restore.md)

## Defaults

| Setting | Value |
|---------|-------|
| Project | `ylebi-rnd` |
| Region | `europe-west3` |
| Database name | From dump (`CREATE DATABASE` / `USE`) |
| Public IP | enabled (PoC) |
| Pub/Sub path | `enable_pubsub = true` |
| Archive prefix | `imported/` (after successful import) |

## Quick start (manual apply)

### Prerequisites

- Terraform `>= 1.14`
- `gcloud` authenticated to `ylebi-rnd` with permission to create SQL, Functions, Pub/Sub, GCS, IAM
- Optional: a sample WordPress dump (start small, then multi-GB)

### Apply

```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars   # edit if needed
terraform init
terraform plan
terraform apply
```

Note outputs: `dumps_bucket_name`, `cloudsql_instance_name`, `function_name`.

```bash
terraform output dumps_bucket_name
terraform output -raw cloudsql_root_password   # PoC only
```

### Pre-flight (recommended)

Before relying on the function, confirm Import works once:

```bash
BUCKET=$(terraform output -raw dumps_bucket_name)
INSTANCE=$(terraform output -raw cloudsql_instance_name)

gsutil cp ./wordpress.sql.gz "gs://${BUCKET}/wordpress.sql.gz"
# If testing import manually without the function path, pause notifications or use a separate object name,
# then:
gcloud sql import sql "${INSTANCE}" "gs://${BUCKET}/manual-test.sql.gz" \
  --project=ylebi-rnd
```

(Omit `--database` when the dump includes `CREATE DATABASE` / `USE`.)

### End-to-end via the function

```bash
BUCKET=$(terraform output -raw dumps_bucket_name)
gsutil cp ./wordpress.sql.gz "gs://${BUCKET}/wordpress.sql.gz"
```

Watch logs:

```bash
gcloud functions logs read gcs-sql-restore-fn \
  --gen2 --region=europe-west3 --project=ylebi-rnd --limit=50
```

Verify tables (Cloud SQL Studio, or `mysql` against the public IP with the root password output). Database name comes from the dump (e.g. `camarotest2-wordpress`).

### Dump expectations

- File ends with `.sql` or `.sql.gz`
- Dump includes `CREATE DATABASE IF NOT EXISTS …` and `USE …` (phpMyAdmin-style)
- No fixed target DB name in the function — site-specific names are supported
- Samples often lack `DROP TABLE`; re-importing into an existing DB may fail on "table already exists"
- After a successful import the object is moved to `imported/<timestamp>_<basename>` (server-side); events under `imported/` are skipped

## Development

```bash
make test-race
make fmt
make lint
```

## Least privilege

| Identity | Access |
|----------|--------|
| Function SA | Custom role `gcsSqlRestoreOrchestrator` (`import` + `operations.get`) + `roles/storage.objectUser` on the dumps bucket |
| Cloud SQL instance SA | `roles/storage.objectViewer` on the dumps bucket |

## Not in this PoC

CI/CD, Direct VPC `mysql` client path, soft merges, multi-tenant routing, alerting UI.

## Teardown

```bash
cd terraform
terraform destroy
```
