resource "google_service_account" "default" {
  account_id   = "archive-cluster"
  display_name = "Cluster Service Account"
}

/*
resource "google_project_iam_custom_role" "cluster-manager" {
  role_id     = "cluster_manager"
  title       = "Cluster Manager"
  description = "Manages the archive server's cluster"
  permissions = [
    "container.clusters.create",
    "container.clusters.get",
    "container.clusters.getCredentials",
    "container.clusters.update",
    "container.operations.get"
  ]
}

resource "google_project_iam_binding" "cluster-manager-binding" {
  role = "projects/${var.project-id}/roles/${google_project_iam_custom_role.cluster-manager.role_id}"

  members = [
    "serviceAccount:${google_service_account.default.email}",
  ]
}
*/