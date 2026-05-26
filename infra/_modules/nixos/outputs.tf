# Allow getting KUBECONFIG
# export KUBECONFIG="$(terragrunt output -raw kubeconfig_path)"
output "kubeconfig_path" {
  value = abspath(local_sensitive_file.kubeconfig.filename)
}
