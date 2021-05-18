// Configure the Google Cloud provider
provider "google" {
 credentials = file("credentials.json")
 project     = var.project-id
 region      = var.zone
}

resource "google_container_cluster" "primary" {
 name     = var.cluster
 location = var.zone
 initial_node_count = 3

 node_config {
  service_account = google_service_account.default.email
  oauth_scopes = [
   "https://www.googleapis.com/auth/logging.write",
   "https://www.googleapis.com/auth/monitoring",
   "https://www.googleapis.com/auth/cloud-platform",
   "https://www.googleapis.com/auth/devstorage.read_only"
  ]

  machine_type = "e2-standard-2"

  metadata = {
   disable-legacy-endpoints = "true"
  }

  labels = {
   app = var.app-name
  }

  tags = ["app", var.app-name]
 }

 timeouts {
  create = "30m"
  update = "40m"
 }
}
