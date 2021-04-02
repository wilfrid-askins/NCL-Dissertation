// https://cloud.google.com/community/tutorials/getting-started-on-gcp-with-terraform

// Configure the Google Cloud provider
provider "google" {
 credentials = file("credentials.json")
 project     = "ultra-zenith-309512"
 region      = "europe-west2-a"
}

resource "google_service_account" "default" {
 account_id   = "archive-cluster"
 display_name = "Cluster Service Account"
}

resource "google_container_cluster" "primary" {
 name     = "archive-gke-cluster"
 location = "europe-west2-a"

 # We can't create a cluster with no node pool defined, but we want to only use
 # separately managed node pools. So we create the smallest possible default
 # node pool and immediately delete it.
 remove_default_node_pool = true
 initial_node_count       = 1
}

resource "google_container_node_pool" "primary_preemptible_nodes" {
 name       = "archive-node-pool"
 location   = "europe-west2-a"
 cluster    = google_container_cluster.primary.name
 node_count = 3

 node_config {
  preemptible  = true
  machine_type = "e2-small"

  # Google recommends custom service accounts that have cloud-platform scope and permissions granted via IAM Roles.
  service_account = google_service_account.default.email
  oauth_scopes    = [
   "https://www.googleapis.com/auth/cloud-platform"
  ]
 }
}
