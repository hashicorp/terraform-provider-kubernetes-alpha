provider "kubernetes-alpha" {
  config_file          = var.kubeconfig
  server_side_planning = true
}
