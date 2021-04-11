provider "kubernetes" {
  config_path    = "C:\\Users\\fab\\.kube\\config"
  config_context = "gke_ultra-zenith-309512_europe-west2-a_archive-gke-cluster"
}

provider "google" {
  credentials = file("credentials.json")
  project     = "ultra-zenith-309512"
  region      = var.zone
}

//resource "kubernetes_namespace" "central" {
//  metadata {
//    name = "central-namespace"
//  }
//}
