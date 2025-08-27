resource "helm_release" "argocd" {
  name             = "argocd"
  namespace        = "argocd"
  create_namespace = true
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = "8.3.0"
  timeout          = 60 * 10
  max_history      = 1 # Revert with Terraform instead

  values = [
    yamlencode({
      global = {
        domain = "argocd.${var.cluster_domain}"
      }
      configs = {
        params = {
          "server.insecure"             = true
          "controller.diff.server.side" = true
        }
        cm = {
          "resource.ignoreResourceUpdatesEnabled" = true
          "resource.customizations.ignoreResourceUpdates.all" = yamlencode({
            jsonPointers = [
              "/status"
            ]
          })
          "admin.enabled" = false
          "oidc.config" = yamlencode({
            name         = "SSO"
            issuer       = "https://dex.${var.cluster_domain}"
            clientID     = "argocd"
            clientSecret = "$oidc.dex.clientSecret"
          })

        }
        rbac = {
          "policy.default" = "role:readonly"
        }
      }
      server = {
        ingress = {
          enabled          = true
          ingressClassName = "nginx"
          annotations = {
            "cert-manager.io/cluster-issuer" = "letsencrypt-prod"
          }
          tls = true
        }
      }
      dex = {
        enabled = false
      }
    })
  ]
}
