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
        container {
          image = var.docker-image
          name  = var.app
          port {
            name           = "port-8080"
            container_port = 8080
          }
          env {
            name = "CONSUL_ADDRESS"
            value = "34.89.30.15:8500"
          }
        }
      }
    }
  }
}
