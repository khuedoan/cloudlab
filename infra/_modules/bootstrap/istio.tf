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

resource "kubectl_manifest" "istio_addons" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "istio-addons"
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
        repoURL        = "https://bjw-s-labs.github.io/helm-charts"
        chart          = "app-template"
        targetRevision = "3.7.3"
        helm = {
          valuesObject = {
            rawResources = {
              # From https://github.com/istio/istio/blob/master/samples/addons/extras/prometheus-operator.yaml
              component-monitor = {
                apiVersion = "monitoring.coreos.com/v1"
                kind       = "ServiceMonitor"
                spec = {
                  spec = {
                    endpoints = [
                      {
                        interval = "15s"
                        port     = "http-monitoring"
                      },
                    ]
                    jobLabel = "istio"
                    namespaceSelector = {
                      "any" = true
                    }
                    selector = {
                      matchExpressions = [
                        {
                          key      = "istio"
                          operator = "In"
                          values = [
                            "pilot",
                          ]
                        },
                      ]
                    }
                    targetLabels = [
                      "app",
                    ]
                  }
                }
              }
              envoy-stats-monitor = {
                apiVersion = "monitoring.coreos.com/v1"
                kind       = "PodMonitor"
                spec = {
                  spec = {
                    jobLabel = "envoy-stats"
                    namespaceSelector = {
                      "any" = true
                    }
                    podMetricsEndpoints = [
                      {
                        interval = "15s"
                        path     = "/stats/prometheus"
                        relabelings = [
                          {
                            action = "keep"
                            regex  = "istio-proxy"
                            sourceLabels = [
                              "__meta_kubernetes_pod_container_name",
                            ]
                          },
                          {
                            action = "keep"
                            sourceLabels = [
                              "__meta_kubernetes_pod_annotationpresent_prometheus_io_scrape",
                            ]
                          },
                          {
                            action      = "replace"
                            regex       = "(\\d+);(([A-Fa-f0-9]{1,4}::?){1,7}[A-Fa-f0-9]{1,4})"
                            replacement = "[$2]:$1"
                            sourceLabels = [
                              "__meta_kubernetes_pod_annotation_prometheus_io_port",
                              "__meta_kubernetes_pod_ip",
                            ]
                            targetLabel = "__address__"
                          },
                          {
                            action      = "replace"
                            regex       = "(\\d+);((([0-9]+?)(\\.|$)){4})"
                            replacement = "$2:$1"
                            sourceLabels = [
                              "__meta_kubernetes_pod_annotation_prometheus_io_port",
                              "__meta_kubernetes_pod_ip",
                            ]
                            targetLabel = "__address__"
                          },
                          {
                            action = "labeldrop"
                            regex  = "__meta_kubernetes_pod_label_(.+)"
                          },
                          {
                            action = "replace"
                            sourceLabels = [
                              "__meta_kubernetes_namespace",
                            ]
                            targetLabel = "namespace"
                          },
                          {
                            action = "replace"
                            sourceLabels = [
                              "__meta_kubernetes_pod_name",
                            ]
                            targetLabel = "pod"
                          },
                        ]
                      },
                    ]
                    selector = {
                      matchExpressions = [
                        {
                          key      = "istio-prometheus-ignore"
                          operator = "DoesNotExist"
                        },
                      ]
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  })
}
