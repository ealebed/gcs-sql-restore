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

output "cloudsql_private_ip" {
  description = "Cloud SQL private IP address (VPC-only; no public IP)"
  value       = google_sql_database_instance.main.private_ip_address
}

output "vpc_network_name" {
  description = "VPC network used for Cloud SQL private IP"
  value       = google_compute_network.main.name
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

output "cloudsql_studio_user" {
  description = "MySQL username for Cloud SQL Studio (root@% is not supported by Studio)"
  value       = google_sql_user.studio.name
}

output "cloudsql_studio_password" {
  description = "Password for sqladmin (PoC only — rotate or use Secret Manager for real use)"
  value       = random_password.cloudsql_admin.result
  sensitive   = true
}
