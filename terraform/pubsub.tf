resource "google_pubsub_topic" "dumps" {
  count = var.enable_pubsub ? 1 : 0

  name    = "${var.name_prefix}-dumps"
  project = var.project_id

  labels = local.labels

  depends_on = [google_project_service.services]
}

resource "google_pubsub_topic" "dumps_dlq" {
  count = var.enable_pubsub ? 1 : 0

  name    = "${var.name_prefix}-dumps-dlq"
  project = var.project_id

  labels = local.labels

  depends_on = [google_project_service.services]
}

resource "google_storage_notification" "dumps_finalize" {
  count = var.enable_pubsub ? 1 : 0

  bucket             = google_storage_bucket.dumps.name
  topic              = google_pubsub_topic.dumps[0].id
  payload_format     = "JSON_API_V1"
  event_types        = ["OBJECT_FINALIZE"]
  object_name_prefix = var.dump_object_prefix != "" ? var.dump_object_prefix : null

  depends_on = [google_pubsub_topic_iam_member.gcs_publisher]
}
