resource "k3d_cluster" "this" {
  name  = var.name
  image = "docker.io/rancher/k3s:v1.30.2-k3s2"

  # See the comments in the mirrors below before deciding to increase the node count
  servers = 1
  agents  = 0

  k3s {
    extra_args {
      arg          = "--disable=traefik"
      node_filters = ["server:*"]
    }

    extra_args {
      arg          = "--disable-helm-controller"
      node_filters = ["server:*"]
    }
  }

  port {
    host_port      = 80
    container_port = 80
    node_filters = [
      "loadbalancer",
    ]
  }

  port {
    host_port      = 443
    container_port = 443
    node_filters = [
      "loadbalancer",
    ]
  }

  registries {
    config = yamlencode({
      mirrors = {
        # Pretent that this registry is external, but it's actually on the same
        # node. That means this development cluster can only have 1 node unless
        # we have RWX and use a DaemonSet for the in-cluster registry.
        # See also the nodePort in ../../system/registry.yaml
        "registry.127.0.0.1.sslip.io" = {
          endpoint = [
            "http://localhost:30000"
          ]
        }
      }
    })
  }
}
