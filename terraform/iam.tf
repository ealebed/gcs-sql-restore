data "google_project" "current" {
  project_id = var.project_id
}

resource "google_service_account" "function" {
  account_id   = "${var.name_prefix}-fn"
  display_name = "GCS SQL restore function"
  project      = var.project_id
}

resource "google_project_iam_custom_role" "restore_orchestrator" {
  role_id     = "gcsSqlRestoreOrchestrator"
  title       = "GCS SQL Restore Orchestrator"
  description = "Minimal Cloud SQL Admin permissions to drop target DB and import SQL dumps from GCS"
  project     = var.project_id

  permissions = [
    "cloudsql.instances.get",
    "cloudsql.instances.import",
    "cloudsql.databases.get",
    "cloudsql.databases.delete",
    "cloudsql.operations.get",
  ]
}

resource "google_project_iam_member" "function_restore_role" {
  project = var.project_id
  role    = google_project_iam_custom_role.restore_orchestrator.id
  member  = "serviceAccount:${google_service_account.function.email}"
}

# Cloud SQL service agent must read the dump object during Import.
resource "google_storage_bucket_iam_member" "cloudsql_dumps_viewer" {
  bucket = google_storage_bucket.dumps.name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_sql_database_instance.main.service_account_email_address}"
}

# Function archives dumps (server-side copy + delete) after successful import.
resource "google_storage_bucket_iam_member" "function_dumps_object_user" {
  bucket = google_storage_bucket.dumps.name
  role   = "roles/storage.objectUser"
  member = "serviceAccount:${google_service_account.function.email}"
}

# Eventarc trigger identity invokes the underlying Cloud Run service.
resource "google_cloud_run_service_iam_member" "function_invoker" {
  count = var.enable_pubsub ? 1 : 0

  project  = var.project_id
  location = var.region
  service  = google_cloudfunctions2_function.restore.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.function.email}"
}

resource "google_cloud_run_service_iam_member" "eventarc_agent_invoker" {
  count = var.enable_pubsub ? 1 : 0

  project  = var.project_id
  location = var.region
  service  = google_cloudfunctions2_function.restore.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:service-${data.google_project.current.number}@gcp-sa-eventarc.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "function_eventarc_receiver" {
  count = var.enable_pubsub ? 1 : 0

  project = var.project_id
  role    = "roles/eventarc.eventReceiver"
  member  = "serviceAccount:${google_service_account.function.email}"
}

resource "google_service_account_iam_member" "eventarc_act_as_function" {
  count = var.enable_pubsub ? 1 : 0

  service_account_id = google_service_account.function.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:service-${data.google_project.current.number}@gcp-sa-eventarc.iam.gserviceaccount.com"
}

# GCS publishes object notifications to Pub/Sub.
resource "google_pubsub_topic_iam_member" "gcs_publisher" {
  count = var.enable_pubsub ? 1 : 0

  project = var.project_id
  topic   = google_pubsub_topic.dumps[0].name
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:service-${data.google_project.current.number}@gs-project-accounts.iam.gserviceaccount.com"
}
