locals {
  labels = {
    project     = var.name_prefix
    managed_by  = "terraform"
    environment = "poc"
  }

  function_env = {
    GCP_PROJECT       = var.project_id
    CLOUDSQL_INSTANCE = google_sql_database_instance.main.name
    IMPORTED_PREFIX   = var.imported_prefix
  }
}
