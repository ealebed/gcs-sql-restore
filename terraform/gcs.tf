resource "google_storage_bucket" "dumps" {
  name                        = "${var.name_prefix}-dumps-${var.project_id}"
  location                    = var.region
  project                     = var.project_id
  uniform_bucket_level_access = true
  force_destroy               = true

  labels = local.labels

  depends_on = [google_project_service.services]
}

resource "google_storage_bucket" "function_source" {
  name                        = "${var.name_prefix}-fn-src-${var.project_id}"
  location                    = var.region
  project                     = var.project_id
  uniform_bucket_level_access = true
  force_destroy               = true

  labels = local.labels

  depends_on = [google_project_service.services]
}

data "archive_file" "function_source" {
  type        = "zip"
  source_dir  = "${path.module}/.."
  output_path = "${path.module}/../function-source.zip"

  excludes = [
    ".git",
    ".terraform",
    "terraform",
    "docs",
    "tasks",
    "bin",
    "function-source.zip",
    ".idea",
    ".vscode",
    "coverage.out",
    ".DS_Store",
  ]
}

resource "google_storage_bucket_object" "function_source" {
  name   = "function-source-${data.archive_file.function_source.output_md5}.zip"
  bucket = google_storage_bucket.function_source.name
  source = data.archive_file.function_source.output_path
}
