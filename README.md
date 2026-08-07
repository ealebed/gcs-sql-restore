# gcs-sql-restore

PoC: upload a WordPress/phpMyAdmin MySQL dump (`.sql` / `.sql.gz`) to GCS →
Pub/Sub → Cloud Run Function (Go) → Cloud SQL Admin **Import** into MySQL 8.4.

The dump owns the database name via `CREATE DATABASE IF NOT EXISTS` + `USE`
(e.g. `test-wordpress`). The function does **not** wipe or pre-create a
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
| Cloud SQL IP | Private IP only (org policy blocks public IP) |
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
terraform output -raw cloudsql_studio_user
terraform output -raw cloudsql_studio_password   # PoC only
```

Terraform only redeploys the Cloud Run Function when the **function source zip hash** changes (Go code / `go.mod` / etc.). Docs, README, and Terraform-only edits should not rebuild the function. To apply DB/user changes without touching the function:

```bash
terraform apply -target=google_sql_user.studio
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

Verify tables in **Cloud SQL Studio** (Console → Cloud SQL → instance → Cloud SQL Studio):

- User: `terraform output -raw cloudsql_studio_user` → `sqladmin`
- Password: `terraform output -raw cloudsql_studio_password`
- Pick a database created by your dump (or `mysql` / `information_schema` to start)

Note: Studio does **not** support MySQL `root@%`. The PoC creates `sqladmin` with `cloudsqlsuperuser` instead.

To connect from a VM/laptop you need VPC reachability (same VPC, VPN, or Cloud SQL Auth Proxy on a resource in the VPC) — not needed for this PoC’s Import path.

### Dump expectations

- File ends with `.sql` or `.sql.gz`
- Dump includes `CREATE DATABASE IF NOT EXISTS …` and `USE …` (phpMyAdmin-style)
- No fixed target DB name in the function — site-specific names are supported
- Samples often lack `DROP TABLE`; re-importing into an existing DB may fail on "table already exists"
- After a successful import the object is moved to `imported/<timestamp>_<basename>` (server-side); events under `imported/` are skipped
- Event-triggered function timeout is max **540s**; larger dumps may still finish in Cloud SQL after the function acks — check operations/Studio; object stays unarchived until a completed run archives it

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

## TODO (later)

- [ ] Long-running import completion watcher: when Import outlives the 540s event-trigger window, poll the Cloud SQL operation asynchronously and archive to `imported/` only on success (see `tasks/todo.md`).

## Teardown

```bash
cd terraform
terraform destroy
```
