terraform {
}

variable "kube_context" {
  default = "minikube"
}

provider "kubernetes" {
  config_path    = "~/.kube/config"
  config_context = var.kube_context
}

provider "helm" {
  kubernetes = {
    config_path    = "~/.kube/config"
    config_context = var.kube_context
  }
}

resource "kubernetes_namespace_v1" "highway" {
  metadata {
    name = "highway"
  }
}

resource "helm_release" "argocd" {
  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  namespace        = "argocd"
  create_namespace = true

  set = [{
    name  = "server.service.type"
    value = "NodePort"
  }]
}

resource "helm_release" "argocd_app" {
  name       = "argocd-app"
  chart      = "../charts/argocd-app"
  namespace  = "argocd"
  depends_on = [helm_release.argocd]
}
