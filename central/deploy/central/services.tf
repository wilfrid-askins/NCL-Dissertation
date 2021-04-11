resource "kubernetes_service" "app" {
  metadata {
    name = var.app
  }
  spec {
    selector = {
      app = kubernetes_deployment.app.metadata.0.labels.app
    }
    port {
      port        = 80
      target_port = 5000
    }

    type = "LoadBalancer"
  }
}

resource "kubernetes_network_policy" "app-network-policy" {
  metadata {
    name      = "${var.app}-network-policy"
    namespace = "default"
  }

  spec {
    pod_selector {
      match_labels = {
        app = kubernetes_deployment.app.metadata.0.labels.app
      }
    }

    ingress {
      ports {
        port     = "80"
        protocol = "TCP"
      }
      ports {
        port     = "8080"
        protocol = "TCP"
      }

      from {
        ip_block {
          cidr = "0.0.0.0/0"
        }
      }
    }
    egress {}
    policy_types = ["Ingress", "Egress"]
  }
}
