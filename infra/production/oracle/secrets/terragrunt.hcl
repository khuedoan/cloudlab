include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "../../../modules//secrets"
}

dependency "cluster" {
  config_path = "../cluster"
}

inputs = {
  credentials = {
    host                   = dependency.cluster.outputs.credentials.host
    client_certificate     = dependency.cluster.outputs.credentials.client_certificate
    client_key             = dependency.cluster.outputs.credentials.client_key
    cluster_ca_certificate = dependency.cluster.outputs.credentials.cluster_ca_certificate
  }

  sources = {
    dex_admin_password_hash    = { value = include.root.locals.secrets.dex_admin_password_hash }
    dex_argocd_client_secret   = { random = true }
    dex_grafana_client_secret  = { random = true }
    dex_kiali_client_secret    = { random = true }
    dex_temporal_client_secret = { random = true }
    silverbullet_user          = { value = include.root.locals.secrets.silverbullet_user }
    wireguard_config           = { value = include.root.locals.secrets.wireguard_config }
  }

  destinations = {
    "dex/dex-secrets" = {
      data = {
        "ARGOCD_CLIENT_SECRET"   = "dex_argocd_client_secret"
        "GRAFANA_CLIENT_SECRET"  = "dex_grafana_client_secret"
        "KIALI_CLIENT_SECRET"    = "dex_kiali_client_secret"
        "TEMPORAL_CLIENT_SECRET" = "dex_temporal_client_secret"
        "ADMIN_PASSWORD_HASH"    = "dex_admin_password_hash"
      }
    }
    "argocd/argocd-secrets" = {
      data = {
        "oidc.dex.clientSecret" = "dex_argocd_client_secret"
      }
    }
    "monitoring/grafana-secrets" = {
      data = {
        "SSO_CLIENT_SECRET" = "dex_grafana_client_secret"
      }
    }
    "istio-system/kiali" = {
      data = {
        "oidc-secret" = "dex_kiali_client_secret"
      }
    }
    "temporal/temporal-web" = {
      data = {
        "TEMPORAL_AUTH_CLIENT_SECRET" = "dex_temporal_client_secret"
      }
    }
    "notes/silverbullet" = {
      data = {
        "SB_USER" = "silverbullet_user"
      }
    }
    "wireguard/wireguard-secret" = {
      data = {
        "wg0.conf" = "wireguard_config"
      }
    }
  }
}
