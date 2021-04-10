variable "org-id" {
  type    = string
  default = "0"
}
variable "project-id" {
  type        = string
  description = "Project identifier"
  default     = "ultra-zenith-309512"
}
variable "app" {
  type        = string
  description = "Archive Server"
  default     = "arch-001"
}
variable "app-name" {
  type    = string
  default = "archive-server"
}
variable "cluster" {
  default = "archive-gke-cluster"
}
variable "zone" {
  default = "europe-west2-a"
}
variable "docker-image" {
  type        = string
  description = "central server docker image"
  default     = "gcr.io/ultra-zenith-309512/central-app:v1"
}