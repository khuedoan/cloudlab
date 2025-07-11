resource "kubectl_manifest" "registry" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "registry"
      namespace  = helm_release.argocd.namespace
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      labels     = local.common_labels
    }
    spec = {
      project = "default"
      destination = {
        name      = "in-cluster"
        namespace = "registry"
      }
      syncPolicy = local.sync_policy
      source = {
        repoURL        = "http://zotregistry.dev/helm-charts"
        chart          = "zot"
        targetRevision = "0.1.67"
        helm = {
          valuesObject = {
            nameOverride = "registry" # Otherwise it will render registry-zot as the service name
            image = {
              repository = "ghcr.io/project-zot/zot"
            }
            podLabels = {
              "istio.io/dataplane-mode" = "ambient"
            }
            # TODO separate logic for k3d
            service = {
              type = "NodePort"
              port = 80
              # HACK Use node port for k3s registry mirror.
              # See also ../../cluster/roles/k3s/templates/registries.yaml.j2
              # The range of valid ports is 30000-32767
              nodePort = 30000
            }
            # TODO enable auth and ingress
            # ingress = {
            #   enabled   = true
            #   className = "nginx"
            #   annotations = {
            #     "cert-manager.io/cluster-issuer" = "letsencrypt-prod"
            #     "nginx.ingress.kubernetes.io/proxy-body-size" = "0"
            #   }
            #   pathtype = "Prefix"
            #   hosts = [{
            #     host = "registry.${var.cluster_domain}"
            #     paths = [{
            #       path = "/"
            #     }]
            #   }]
            #   tls = [{
            #     hosts = ["registry.${var.cluster_domain}"]
            #     secretName = "registry-tls-certificate"
            #   }]
            # }
            persistence = true
            pvc = {
              create  = true
              storage = "10Gi"
            }
          }
        }
      }
    }
  })
}
