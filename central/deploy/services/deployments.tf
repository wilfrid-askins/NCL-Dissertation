resource "kubernetes_deployment" "database" {
  metadata {
    name = var.database-id
    labels = {
      app = var.database-id
    }
  }

  spec {
    selector {
      match_labels = {
        app = var.database-id
      }
    }

    template {
      metadata {
        labels = {
          app = var.database-id
        }
      }

      spec {
        container {
          image = "mysql:latest"
          name  = var.database-id
          port {
            name           = "port-3306"
            container_port = 3306
          }
          env {
            name = "MYSQL_ROOT_PASSWORD"
            value = "8fn194nq7sm6l2d8n2n6"
          }
          env {
            name = "MYSQL_DATABASE"
            value = "central"
          }
          env {
            name = "MYSQL_USER"
            value = "central-user"
          }
          env {
            name = "MYSQL_PASSWORD"
            value = "2w5j8a0fh4ja39s5"
          }
        }
      }
    }
  }
}

resource "kubernetes_deployment" "cache" {
  metadata {
    name = var.cache-id
    labels = {
      app = var.cache-id
    }
  }

  spec {
    selector {
      match_labels = {
        app = var.cache-id
      }
    }

    template {
      metadata {
        labels = {
          app = var.cache-id
        }
      }

      spec {
        container {
          image = "redis:latest"
          name  = var.cache-id
          port {
            name           = "port-6379"
            container_port = 6379
          }
        }
      }
    }
  }
}

resource "kubernetes_deployment" "configuration" {
  metadata {
    name = var.config-id
    labels = {
      app = var.config-id
    }
  }

  spec {
    selector {
      match_labels = {
        app = var.config-id
      }
    }

    template {
      metadata {
        labels = {
          app = var.config-id
        }
      }

      spec {
        container {
          image = "consul:latest"
          name  = var.config-id
          port {
            name           = "port-8500"
            container_port = 8500
          }
        }
      }
    }
  }
}
