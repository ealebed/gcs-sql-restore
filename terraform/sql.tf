resource "random_password" "cloudsql_root" {
  length           = 24
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

resource "google_sql_database_instance" "main" {
  name             = "${var.name_prefix}-mysql"
  project          = var.project_id
  region           = var.region
  database_version = "MYSQL_8_4"

  deletion_protection = false

  settings {
    tier              = var.cloudsql_tier
    edition           = "ENTERPRISE"
    availability_type = "ZONAL"
    disk_size         = 20
    disk_type         = "PD_SSD"

    ip_configuration {
      ipv4_enabled = true
    }

    backup_configuration {
      enabled = false
    }

    user_labels = local.labels
  }

  root_password = random_password.cloudsql_root.result

  depends_on = [google_project_service.services]
}
