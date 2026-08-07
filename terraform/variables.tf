variable "project_id" {
  description = "GCP project ID where PoC resources are created"
  type        = string
  default     = "ylebi-rnd"
}

variable "region" {
  description = "Primary GCP region for Cloud SQL, Cloud Functions, and related resources"
  type        = string
  default     = "europe-west3"
}

variable "name_prefix" {
  description = "Prefix applied to resource names"
  type        = string
  default     = "gcs-sql-restore"
}

variable "enable_pubsub" {
  description = "When true, GCS notifications publish to Pub/Sub and the function is triggered from that topic"
  type        = bool
  default     = true
}

variable "cloudsql_tier" {
  description = "Cloud SQL machine tier for the PoC instance"
  type        = string
  default     = "db-f1-micro"
}

variable "function_memory" {
  description = "Cloud Run Function memory limit"
  type        = string
  default     = "512Mi"
}

variable "function_timeout_seconds" {
  description = "Cloud Run Function timeout in seconds. Event-triggered functions are capped at 540s by GCP."
  type        = number
  default     = 540

  validation {
    condition     = var.function_timeout_seconds >= 60 && var.function_timeout_seconds <= 540
    error_message = "function_timeout_seconds must be between 60 and 540 for event-triggered functions."
  }
}

variable "dump_object_prefix" {
  description = "Optional GCS object prefix filter for notifications (empty = entire bucket)"
  type        = string
  default     = ""
}

variable "imported_prefix" {
  description = "GCS prefix where dumps are moved after a successful import"
  type        = string
  default     = "imported"
}

variable "vpc_subnet_cidr" {
  description = "CIDR range for the PoC VPC subnet"
  type        = string
  default     = "10.20.0.0/24"
}

variable "private_services_prefix_length" {
  description = "Prefix length for the Private Services Access allocated range used by Cloud SQL"
  type        = number
  default     = 16

  validation {
    condition     = var.private_services_prefix_length >= 16 && var.private_services_prefix_length <= 24
    error_message = "private_services_prefix_length must be between 16 and 24."
  }
}
