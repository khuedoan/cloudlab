resource "kubectl_manifest" "vault_operator" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "vault-operator"
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
        repoURL        = "ghcr.io"
        chart          = "bank-vaults/helm-charts/vault-operator"
        targetRevision = "1.23.0"
      }
    }
  })
}

resource "kubectl_manifest" "vault_secrets_webhook" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "vault-secrets-webhook"
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
        repoURL        = "ghcr.io"
        chart          = "bank-vaults/helm-charts/vault-secrets-webhook"
        targetRevision = "1.22.0"
        helm = {
          valuesObject = {
            env = {
              VAULT_ADDR = "http://vault-cluster.vault.svc.cluster.local:8200"
            }
          }
        }
      }
    }
  })
}

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
        repoURL        = "https://bjw-s-labs.github.io/helm-charts"
        chart          = "app-template"
        targetRevision = "3.7.3"
        helm = {
          valuesObject = {
            rawResources = {
              cluster = {
                apiVersion = "vault.banzaicloud.com/v1alpha1"
                kind       = "Vault"
                spec = {
                  spec = {
                    size  = 1
                    image = "docker.io/hashicorp/vault:1.20.2"
                    serviceAccount = "vault"
                    config = {
                      storage = {
                        file = {
                          path = "/vault/data"
                        }
                      }
                      listener = {
                        tcp = {
                          address = "0.0.0.0:8200"
                          tls_disable = true
                        }
                      }
                      ui = true
                    }
                    unsealConfig = {
                      kubernetes = {
                        secretNamespace = "{{ .Release.Namespace }}"
                      }
                    }
                    externalConfig = {
                      secrets = [
                        {
                          path = "secret"
                          type = "kv"
                          options = {
                            version = 2
                          }
                        }
                      ]
                      policies = [
                        {
                          name = "allow_secrets"
                          # TODO make it less ugly
                          rules = file("${path.module}/vault-policies/allow_secrets.hcl")
                        }
                      ]
                      auth = [
                        {
                          type = "kubernetes"
                          roles = [
                            {
                              # TODO optimize this
                              name = "default"
                              bound_service_account_names: ["default"]
                              bound_service_account_namespaces: ["default"]
                              policies: ["allow_secrets"]
                              ttl: "1h"
                            }
                          ]
                        }
                      ]
                    }
                    volumes = [{
                      name = "vault-data"
                      persistentVolumeClaim : {
                        claimName : "vault-data"
                      }
                    }]
                    volumeMounts : [{
                      name      = "vault-data"
                      mountPath = "/vault/data"
                    }]
                    ingress = {
                      annotations = {
                        "cert-manager.io/cluster-issuer" = "letsencrypt-prod"
                      }
                      spec = {
                        ingressClassName = "nginx"
                        rules = [{
                          host = "vault.${var.cluster_domain}"
                          http = {
                            paths = [{
                              backend = {
                                service = {
                                  name = "vault-cluster"
                                  port = {
                                    "number" = 8200
                                  }
                                }
                              }
                              path = "/"
                              pathType = "Prefix"
                            }]
                          }
                        }]
                        tls = [{
                          hosts = ["vault.${var.cluster_domain}"]
                          secretName = "vault-tls-certificate"
                        }]
                      }
                    }
                  }
                }
              }
            }
            serviceAccount = {
              create = true
            }
            rbac = {
              roles = {
                vault = {
                  type = "Role"
                  rules = [{
                    apiGroups = [""]
                    resources = ["secrets"]
                    verbs = ["*"]
                  }, {
                    apiGroups = [""]
                    resources = ["pods"]
                    verbs = ["get", "update", "patch"]
                  }]
                }
              }
              bindings = {
                namespace = {
                  forceRename = "vault"
                  type = "RoleBinding"
                  roleRef = {
                    apiGroup = "rbac.authorization.k8s.io"
                    kind = "Role"
                    name = "vault"
                  }
                  subjects = [{
                    kind = "ServiceAccount"
                    namespace = "{{ .Release.Namespace }}"
                    name = "vault"
                  }]
                }
                cluster = {
                  forceRename = "vault"
                  type = "ClusterRoleBinding"
                  roleRef = {
                    apiGroup = "rbac.authorization.k8s.io"
                    kind = "ClusterRole"
                    name = "system:auth-delegator"
                  }
                  subjects = [{
                    kind = "ServiceAccount"
                    namespace = "{{ .Release.Namespace }}"
                    name = "vault"
                  }]
                }
              }
            }
            persistence = {
              data = {
                accessMode = "ReadWriteOnce"
                size       = "2Gi"
              }
            }
          }
        }
      }
    }
  })
}
