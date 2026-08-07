output "project_id" {
  description = "GCP project ID"
  value       = var.project_id
}

output "region" {
  description = "Deployment region"
  value       = var.region
}

output "dumps_bucket_name" {
  description = "GCS bucket that accepts WordPress SQL dumps"
  value       = google_storage_bucket.dumps.name
}

output "cloudsql_instance_name" {
  description = "Cloud SQL instance ID"
  value       = google_sql_database_instance.main.name
}

output "cloudsql_connection_name" {
  description = "Cloud SQL connection name (project:region:instance)"
  value       = google_sql_database_instance.main.connection_name
}

output "cloudsql_public_ip" {
  description = "Cloud SQL public IP address"
  value       = google_sql_database_instance.main.public_ip_address
}

output "function_name" {
  description = "Cloud Run Function name"
  value       = google_cloudfunctions2_function.restore.name
}

output "function_service_account" {
  description = "Service account email used by the restore function"
  value       = google_service_account.function.email
}

output "pubsub_topic" {
  description = "Pub/Sub topic receiving GCS object finalize notifications (null when enable_pubsub=false)"
  value       = var.enable_pubsub ? google_pubsub_topic.dumps[0].name : null
}

output "pubsub_dlq_topic" {
  description = "Pub/Sub dead-letter topic name (null when enable_pubsub=false)"
  value       = var.enable_pubsub ? google_pubsub_topic.dumps_dlq[0].name : null
}

output "cloudsql_root_password" {
  description = "Generated Cloud SQL root password (PoC only — rotate or store in Secret Manager for real use)"
  value       = random_password.cloudsql_root.result
  sensitive   = true
}
