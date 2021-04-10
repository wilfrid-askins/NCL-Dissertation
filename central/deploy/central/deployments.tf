resource "kubernetes_deployment" "app" {
  metadata {
    name = var.app
    labels = {
      app = var.app
    }
  }

  spec {
    replicas = 3

    selector {
      match_labels = {
        app = var.app
      }
    }

    template {
      metadata {
        labels = {
          app = var.app
        }
      }

      spec {
        // service_account_name = google_service_account.default.account_id
        container {
          image = var.docker-image
          name  = var.app
          port {
            name           = "port-5000"
            container_port = 5000
          }
        }
      }
    }
  }
}
