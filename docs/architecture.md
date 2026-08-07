# Architecture

PoC flow: upload a WordPress MySQL dump to GCS, restore it into Cloud SQL via
the Admin Import API, then archive the object under `imported/`.

## Diagram

```mermaid
flowchart LR
  subgraph Upload
    U[Operator / CI uploads<br/>.sql or .sql.gz]
  end

  subgraph GCS["GCS dumps bucket"]
    O[Object at bucket root<br/>e.g. wordpress.sql.gz]
    A["imported/timestamp_basename<br/>after success"]
  end

  subgraph Messaging
    T[Pub/Sub topic]
    DLQ[Pub/Sub DLQ topic]
  end

  subgraph Compute["Cloud Run Function Gen2 (Go)"]
    F[RestoreSQLDump]
    F --> V{SQL dump and<br/>not under imported/?}
    V -->|no| S[Ack / skip]
    V -->|yes| W[Delete DB if exists<br/>Create wordpress]
    W --> I[Cloud SQL instances.import<br/>gs://…]
    I --> P[Poll Operation]
    P --> M[Server-side move<br/>to imported/]
  end

  subgraph Data["Cloud SQL MySQL 8.4"]
    DB[(wordpress)]
  end

  U -->|OBJECT_FINALIZE| O
  O -->|GCS notification| T
  T -->|Eventarc retry| F
  T -.-> DLQ
  I -->|Import reads object<br/>via Cloud SQL instance SA| O
  I --> DB
  M --> A
  A -.->|OBJECT_FINALIZE ignored<br/>by prefix filter| F
```

## Sequence

```mermaid
sequenceDiagram
  actor Op as Operator
  participant GCS as GCS bucket
  participant PS as Pub/Sub
  participant Fn as Cloud Run Function
  participant SQL as Cloud SQL Admin API
  participant Inst as Cloud SQL MySQL 8.4

  Op->>GCS: upload wordpress.sql.gz
  GCS->>PS: OBJECT_FINALIZE (JSON_API_V1)
  PS->>Fn: messagePublished (Eventarc)
  Fn->>Fn: validate .sql/.sql.gz, skip imported/
  Fn->>SQL: delete wordpress (if exists)
  Fn->>SQL: create wordpress
  Fn->>SQL: instances.import(gs://bucket/wordpress.sql.gz)
  SQL->>Inst: load dump (instance SA reads GCS)
  SQL-->>Fn: Operation DONE
  Fn->>GCS: copy to imported/timestamp_wordpress.sql.gz
  Fn->>GCS: delete original object
  Note over GCS,Fn: finalize on imported/ is skipped
```

## Identities

| Principal | Role |
|-----------|------|
| Function SA | Custom Cloud SQL orchestrator role + `roles/storage.objectUser` on dumps bucket |
| Cloud SQL instance SA | `roles/storage.objectViewer` on dumps bucket (Import read) |
| GCS project SA | `roles/pubsub.publisher` on the dumps topic |

See also: [ADR-001](decisions/ADR-001-cloudsql-import-api.md), [ADR-002](decisions/ADR-002-pubsub-trigger.md), [ADR-003](decisions/ADR-003-go-least-privilege.md).
