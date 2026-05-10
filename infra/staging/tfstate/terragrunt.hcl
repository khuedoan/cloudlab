include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "${find_in_parent_folders("_modules")}//tfstate"

  before_hook "bootstrap_tfstate" {
    commands = ["init", "plan", "apply"]
    execute = [
      "go", "run", ".",
      "--api-token=${include.root.locals.secrets.cloudflare_tfstate_api_token}",
      "--account-id=${include.root.locals.secrets.cloudflare_account_id}",
      "--bucket=tfstate-${include.root.locals.env}",
    ]
  }
}
