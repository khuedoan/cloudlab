resource "kubectl_manifest" "csi_secrets_store" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "csi-secrets-store"
      namespace  = helm_release.argocd.namespace
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      labels     = local.common_labels
    }
    spec = {
      project = "default"
      destination = {
        name      = "in-cluster"
        namespace = "kube-system"
      }
      syncPolicy = local.sync_policy
      source = {
        repoURL        = "https://kubernetes-sigs.github.io/secrets-store-csi-driver/charts"
        chart          = "secrets-store-csi-driver"
        targetRevision = "1.5.1"
      }
    }
  })
}

# TODO auto unseal
resource "kubectl_manifest" "vault" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "vault"
      namespace  = helm_release.argocd.namespace
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      labels     = local.common_labels
    }
    spec = {
      project = "default"
      destination = {
        name      = "in-cluster"
        namespace = "vault"
      }
      syncPolicy = local.sync_policy
      source = {
        repoURL        = "https://openbao.github.io/openbao-helm"
        chart          = "openbao"
        targetRevision = "0.15.0"
        helm = {
          valuesObject = {
            injector = {
              enabled = false
            }
            server = {
              ingress = {
                enabled          = true
                ingressClassName = "nginx"
                annotations = {
                  "cert-manager.io/cluster-issuer" = "letsencrypt-prod"
                }
                hosts = [{
                  host = "vault.${var.cluster_domain}"
                  paths = [
                    "/"
                  ]
                }]
                tls = [{
                  hosts      = ["vault.${var.cluster_domain}"]
                  secretName = "vault-tls-certificate"
                }]
              }
            }
            ui = {
              enabled = true
            }
            csi = {
              enabled = true
            }
          }
        }
      }
    }
  })
}
