resource "kubernetes_service" "configuration-external" {
  metadata {
    name = var.config-id
  }
  spec {
    selector = {
      app = kubernetes_deployment.configuration.metadata.0.labels.app
    }
    port {
      port        = 8500
      target_port = 8500
    }

    type = "LoadBalancer"
  }
}

resource "kubernetes_service" "database-external" {
  metadata {
    name = var.database-id
  }
  spec {
    selector = {
      app = kubernetes_deployment.database.metadata.0.labels.app
    }
    port {
      port        = 3306
      target_port = 3306
    }

    type = "LoadBalancer"
  }
}
