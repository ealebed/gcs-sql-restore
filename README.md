# gcs-sql-restore

PoC: upload a WordPress MySQL dump (`.sql` / `.sql.gz`) to GCS → Pub/Sub →
Cloud Run Function (Go) → Cloud SQL Admin **Import** into MySQL 8.4 database
`wordpress` (wipe & recreate).

The function never downloads the dump. That is what makes multi-GB restores
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
                                               ├── delete DB if exists
                                               ├── create `wordpress`
                                               ├── instances.import(gs://…)
                                               └── move object → imported/<ts>_file
                                                         │
                                                         ▼
                                              Cloud SQL MySQL 8.4
                                         (reads object via instance SA)
```

Decisions: [docs/decisions/](docs/decisions/) · Idea brief: [docs/ideas/gcs-sql-restore.md](docs/ideas/gcs-sql-restore.md)

## Defaults

| Setting | Value |
|---------|-------|
| Project | `ylebi-rnd` |
| Region | `europe-west3` |
| Database | `wordpress` |
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
  --database=wordpress \
  --project=ylebi-rnd
```

(Create `wordpress` first if doing a fully manual test: `gcloud sql databases create wordpress --instance=...`.)

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

Verify tables (Cloud SQL Studio, or `mysql` against the public IP with the root password output).

### Dump expectations

- File ends with `.sql` or `.sql.gz`
- Prefer dumps that target tables for database `wordpress` (avoid conflicting `CREATE DATABASE` for other names when possible)
- Each upload **destructively** replaces `wordpress`
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
| Function SA | Custom role `gcsSqlRestoreOrchestrator` + `roles/storage.objectUser` on the dumps bucket (archive move) |
| Cloud SQL instance SA | `roles/storage.objectViewer` on the dumps bucket |

## Not in this PoC

CI/CD, Direct VPC `mysql` client path, soft merges, multi-tenant routing, alerting UI.

## Teardown

```bash
cd terraform
terraform destroy
```
