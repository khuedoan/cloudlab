resource "kubectl_manifest" "ingress_nginx" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "ingress-nginx"
      namespace  = helm_release.argocd.namespace
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      labels     = local.common_labels
    }
    spec = {
      project = "default"
      destination = {
        name      = "in-cluster"
        namespace = "ingress-nginx"
      }
      syncPolicy = local.sync_policy
      source = {
        repoURL        = "https://kubernetes.github.io/ingress-nginx"
        chart          = "ingress-nginx"
        targetRevision = "4.13.0"
        helm = {
          valuesObject = {
            controller = {
              podLabels = {
                "istio.io/dataplane-mode" = "ambient"
              }
              admissionWebhooks = {
                timeoutSeconds = 30
              }
            }
          }
        }
      }
    }
  })
}
