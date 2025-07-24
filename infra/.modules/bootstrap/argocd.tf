resource "helm_release" "argocd" {
  name             = "argocd"
  namespace        = "argocd"
  create_namespace = true
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = "8.1.3"
  timeout          = 60 * 10
  max_history      = 1 # Revert with Terraform instead

  values = [
    yamlencode({
      global = {
        domain = "argocd.${var.cluster_domain}"
        image = {
          # TODO override to use the latest development version for first class OCI support
          # Remove once ArgoCD 3.1.0 stable is released
          tag = "v3.1.0-rc1"
        }
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
          # TODO remove k3d-specific logic?
          "policy.default" = var.platform == "k3d" ? "role:admin" : "role:readonly"
        }
      }
      server = {
        ingress = {
          enabled          = true
          ingressClassName = "nginx"
          annotations = {
            "cert-manager.io/cluster-issuer" = "letsencrypt-prod"
          }
          tls = !(var.platform == "k3d")
        }
      }
      repoServer = {
        hostNetwork = var.platform == "k3d"
        dnsPolicy   = var.platform == "k3d" ? "ClusterFirstWithHostNet" : "ClusterFirst"
      }
      dex = {
        enabled = false
      }
    })
  ]
}
