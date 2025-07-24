resource "kubectl_manifest" "platform" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "platform"
      namespace  = helm_release.argocd.namespace
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      labels     = local.common_labels
    }
    spec = {
      project = "default" # TODO separate project
      destination = {
        name      = "in-cluster"
        namespace = helm_release.argocd.namespace
      }
      syncPolicy = local.sync_policy
      source = {
        repoURL        = "oci://registry.${var.cluster_domain}/platform" # TODO use registry var
        targetRevision = var.cluster
        path           = "."
      }
    }
  })
}
