resource "random_password" "cloudsql_admin" {
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
      ipv4_enabled                                  = false
      private_network                               = google_compute_network.main.id
      enable_private_path_for_google_cloud_services = true
    }

    backup_configuration {
      enabled = false
    }

    user_labels = local.labels
  }

  depends_on = [
    google_project_service.services,
    google_service_networking_connection.private_vpc_connection,
  ]
}

# Cloud SQL Studio rejects MySQL root@%. Use a dedicated admin with cloudsqlsuperuser.
resource "google_sql_user" "studio" {
  name     = "sqladmin"
  instance = google_sql_database_instance.main.name
  host     = "%"
  project  = var.project_id
  password = random_password.cloudsql_admin.result

  # MySQL 8+ predefined role for full admin access in Studio / clients.
  database_roles = ["cloudsqlsuperuser"]
}
