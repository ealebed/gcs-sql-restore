# ADR-002: GCS finalize via Pub/Sub to Cloud Run Function

## Status
Accepted

## Date
2026-08-07

## Context
We need a retry-friendly path from GCS `OBJECT_FINALIZE` to the restore
orchestrator. Options included a direct Eventarc GCS trigger, or
GCS → Pub/Sub → function, with an `enable_pubsub` Terraform toggle.

## Decision
Default to Pub/Sub (`enable_pubsub = true`):

1. `google_storage_notification` publishes JSON_API_V1 payloads to a topic
2. Cloud Functions Gen2 `event_trigger` subscribes via Eventarc
   (`messagePublished`) with retry enabled
3. A DLQ topic (+ pull subscription) is provisioned for ops inspection and
   future binding; Eventarc retries are the primary reliability mechanism in
   this PoC

The Go function filters for `.sql` / `.sql.gz`, skips objects already under the
`imported/` archive prefix, and acknowledges those without error (no retry
storm). A successful import finishes with a server-side move into
`imported/<timestamp>_<basename>`, which itself emits `OBJECT_FINALIZE` and is
ignored by the prefix filter.

## Alternatives Considered

### Direct Eventarc on GCS object finalized
- Pros: Fewer resources
- Cons: Weaker story for buffer/retry/DLQ evolution the team asked to see
- Kept available by setting `enable_pubsub = false` later (requires adding a
  direct trigger block — not implemented in v1 beyond the flag)

### Pub/Sub push subscription with native dead-letter policy
- Pros: First-class DLQ on the subscription
- Cons: Duplicates Eventarc delivery semantics; more moving parts for PoC
- Deferred; DLQ topic is created so the team can migrate to this pattern

## Consequences
- Terraform creates topic, DLQ topic, GCS notification, and Eventarc-triggered
  function
- GCS project service account must be `pubsub.publisher` on the topic
- Function SA uses a custom role limited to import/wipe operations, plus
  `roles/storage.objectUser` on the dumps bucket to archive objects after import
- Archive moves must skip the `imported/` prefix to avoid recursive triggers
