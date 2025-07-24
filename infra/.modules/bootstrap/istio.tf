resource "kubectl_manifest" "istio_cni" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "istio-cni"
      namespace  = helm_release.argocd.namespace
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      labels     = local.common_labels
    }
    spec = {
      project = "default"
      destination = {
        name      = "in-cluster"
        namespace = "istio-system"
      }
      syncPolicy = local.sync_policy
      source = {
        repoURL        = "https://istio-release.storage.googleapis.com/charts"
        chart          = "cni"
        targetRevision = "1.25.1"
        helm = {
          valuesObject = {
            global = {
              platform = var.platform
            }
            profile = "ambient"
          }
        }
      }
    }
  })
}

resource "kubectl_manifest" "ztunnel" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "ztunnel"
      namespace  = helm_release.argocd.namespace
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      labels     = local.common_labels
    }
    spec = {
      project = "default"
      destination = {
        name      = "in-cluster"
        namespace = "istio-system"
      }
      syncPolicy = local.sync_policy
      source = {
        repoURL        = "https://istio-release.storage.googleapis.com/charts"
        chart          = "ztunnel"
        targetRevision = "1.25.1"
        helm = {
          valuesObject = {
            profile = "ambient"
          }
        }
      }
    }
  })
}

resource "kubectl_manifest" "istio_base" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "istio-base"
      namespace  = helm_release.argocd.namespace
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      labels     = local.common_labels
    }
    spec = {
      project = "default"
      destination = {
        name      = "in-cluster"
        namespace = "istio-system"
      }
      syncPolicy = local.sync_policy
      source = {
        repoURL        = "https://istio-release.storage.googleapis.com/charts"
        chart          = "base"
        targetRevision = "1.25.1"
      }
    }
  })
}

resource "kubectl_manifest" "istiod" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "istiod"
      namespace  = helm_release.argocd.namespace
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      labels     = local.common_labels
    }
    spec = {
      project = "default"
      destination = {
        name      = "in-cluster"
        namespace = "istio-system"
      }
      syncPolicy = local.sync_policy
      source = {
        repoURL        = "https://istio-release.storage.googleapis.com/charts"
        chart          = "istiod"
        targetRevision = "1.25.1"
        helm = {
          valuesObject = {
            profile = "ambient"
          }
        }
      }
    }
  })
}
