/*
resource "google_service_account" "default" {
  account_id   = "archive-deployment"
  display_name = "Deployment Service Account"
}

resource "google_project_iam_custom_role" "deployment-manager" {
  role_id     = "deployment_manager"
  title       = "Deployment Manager"
  description = "Manages the archive server's deployment"
  permissions = [
    "container.clusters.create",
    "container.clusters.get",
    "container.clusters.getCredentials",
    "container.clusters.update",
    "container.pods.list",
    "container.deployments.create",
  ]
}

resource "google_project_iam_binding" "deployment-manager-binding" {
  role = "projects/${var.project-id}/roles/${google_project_iam_custom_role.deployment-manager.role_id}"

  members = [
    "serviceAccount:${google_service_account.default.email}",
  ]
}
resource "google_project_iam_binding" "container-admin-binding" {
  role = "roles/container.admin"

  members = [
    "serviceAccount:${google_service_account.default.email}",
    "serviceAccount:archive-k8s@ultra-zenith-309512.iam.gserviceaccount.com"
  ]
}
*/