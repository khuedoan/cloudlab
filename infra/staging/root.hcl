locals {
  secrets = yamldecode(sops_decrypt_file(find_in_parent_folders("secrets.yaml")))
  env     = basename(get_parent_terragrunt_dir())
  cloud   = split("/", path_relative_to_include())[0]
}

generate "backend" {
  path              = "backend.tf.json"
  if_exists         = "overwrite"
  disable_signature = true
  contents = jsonencode({
    terraform = {
      backend = {
        s3 = {
          bucket                      = "tfstate-${local.env}"
          key                         = "${path_relative_to_include()}/tfstate.json"
          region                      = "auto"
          skip_credentials_validation = true
          skip_metadata_api_check     = true
          skip_region_validation      = true
          skip_requesting_account_id  = true
          skip_s3_checksum            = true
          use_path_style              = true
          access_key                  = local.secrets.cloudflare_tfstate_access_key
          secret_key                  = local.secrets.cloudflare_tfstate_secret_key
          endpoints = {
            s3 = "https://${local.secrets.cloudflare_account_id}.r2.cloudflarestorage.com"
          }
        }
      }
    }
  })
}

generate "provider" {
  path              = "provider.tf.json"
  if_exists         = "overwrite"
  disable_signature = true

  contents = jsonencode(lookup(
    {
      proxmox = {
        provider = {
          proxmox = {
            endpoint = "https://proxmox:8006"
            insecure = true
          }
        }
      }
    },
    local.cloud,
    {}
  ))
}
