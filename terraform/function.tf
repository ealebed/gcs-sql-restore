resource "google_cloudfunctions2_function" "restore" {
  name     = "${var.name_prefix}-fn"
  project  = var.project_id
  location = var.region

  description = "Orchestrates Cloud SQL import from GCS SQL dumps"

  build_config {
    runtime     = "go126"
    entry_point = "RestoreSQLDump"

    source {
      storage_source {
        bucket = google_storage_bucket.function_source.name
        object = google_storage_bucket_object.function_source.name
      }
    }
  }

  service_config {
    available_memory                 = var.function_memory
    timeout_seconds                  = var.function_timeout_seconds
    max_instance_count               = 1
    max_instance_request_concurrency = 1
    service_account_email            = google_service_account.function.email
    ingress_settings                 = "ALLOW_INTERNAL_ONLY"

    environment_variables = local.function_env
  }

  dynamic "event_trigger" {
    for_each = var.enable_pubsub ? [1] : []
    content {
      trigger_region = var.region
      event_type     = "google.cloud.pubsub.topic.v1.messagePublished"
      pubsub_topic   = google_pubsub_topic.dumps[0].id
      retry_policy   = "RETRY_POLICY_RETRY"

      service_account_email = google_service_account.function.email
    }
  }

  labels = local.labels

  depends_on = [
    google_project_service.services,
    google_project_iam_member.function_restore_role,
  ]

  lifecycle {
    precondition {
      condition     = var.enable_pubsub
      error_message = "enable_pubsub=false is reserved for a future direct Eventarc path and is not implemented in this PoC."
    }
  }
}

# Dead-letter subscription for failed deliveries (ops inspection).
resource "google_pubsub_subscription" "dumps_dlq" {
  count = var.enable_pubsub ? 1 : 0

  name    = "${var.name_prefix}-dumps-dlq-pull"
  project = var.project_id
  topic   = google_pubsub_topic.dumps_dlq[0].name

  expiration_policy {
    ttl = ""
  }

  labels = local.labels

  depends_on = [google_project_service.services]
}
