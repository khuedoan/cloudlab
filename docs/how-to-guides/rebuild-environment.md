# Rebuild environment

This is the runbook for destroying and recreating an environment such as
`staging`.

## Before you start

- Run the rebuild from a Linux host that can apply the NixOS parts.
- Make sure the repo on that host contains the changes you want to deploy.
- Make sure `infra/<env>/secrets.yaml` exists and can be decrypted.
- Make sure `settings.yaml` is ready for `toolbox secrets`.
- Expect a destructive rebuild to rotate the node SSH host key.
- Start a `tmux` session before running the long-lived commands in this guide.
  `terragrunt destroy --all`, `terragrunt apply --all`, and the `toolbox`
  publish steps can take long enough that you do not want them tied to a single
  terminal or SSH connection.

## Recreate infra

For example in `staging`:

```sh
pushd ~/Documents/cloudlab/infra/staging
terragrunt destroy --all
popd
make infra bootstrap platform env=staging
```

During bootstrap, it will ask you to input some secrets, open `tmux` session to
do that (you may want to rotate your API keys as well).

## Verify the rebuild

Run the smoke tests:

```sh
make test env=staging
```

## Known failure modes

### Flux says a release is ready, but the workload is missing

This happened with `cert-manager`, `dex`, and the Istio stack during rebuilds.
The live workload was gone, but helm-controller still had the old release state
in `flux-system`.

Fix:

1. Delete the stale `HelmRelease`
2. Delete the matching Helm storage secret in `flux-system`
3. Reconcile `platform`

Example:

```sh
kubectl -n flux-system delete helmrelease dex
kubectl -n flux-system delete secret sh.helm.release.v1.dex.v1
kubectl -n flux-system annotate kustomization platform \
  reconcile.fluxcd.io/requestedAt="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --overwrite
```

### A `HelmRelease` is stuck in terminal failed state

This showed up with `cert-manager`. The workload was healthy, but
helm-controller refused to retry because the release was marked `RetriesExceeded`.

Fix:

```sh
ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)
kubectl -n flux-system annotate helmrelease cert-manager \
  reconcile.fluxcd.io/resetAt="$ts" \
  reconcile.fluxcd.io/requestedAt="$ts" \
  --overwrite
```

### New pods fail with `istio-cni ... Unauthorized`

This means the Istio CNI state on the node survived, but the Istio workloads in
`istio-system` did not.

Symptoms:

- pods stay in `ContainerCreating` or `Init`
- pod events contain `plugin type="istio-cni" ... Unauthorized`
- `istio-system` is empty or partially missing

Fix:

1. Delete stale Istio `HelmRelease` objects
2. Delete the matching Helm storage secrets
3. Reconcile `platform`
4. Recreate any pods that were created while CNI was broken

### Forgejo cannot bootstrap OIDC through the public Dex URL

Forgejo uses the in-cluster Dex service for bootstrap instead of
hairpinning through the public gateway.

If Forgejo init fails, check:

```sh
kubectl -n forgejo logs deploy/forgejo -c configure-gitea
kubectl get all -n dex
```

### Public `443` is broken even though Gateway and HTTPRoutes are healthy

This showed up after restart when:

- `gateway-istio` was healthy
- `HTTPRoute`s were accepted
- public `curl` to `https://*.staging.khuedoan.com` still failed

Root cause:

- the host IPv6 DNAT rule for `2a01:4f9:c013:e5ee::1:443` still pointed to an
  old gateway pod IP

Checks:

```sh
kubectl -n kube-system get pod -l svccontroller.k3s.cattle.io/svcname=gateway-istio -o wide
ip6tables -t nat -S | grep '2a01:4f9:c013:e5ee::1/128 -p tcp -m tcp --dport 443'
```

If the DNAT target points at a dead pod IP, either recycle the `svclb` pod or
rewrite the top `PREROUTING` and `OUTPUT` rules to the live
`gateway-istio` Service ClusterIP.

## Current staging-specific notes

- `platform/staging` is wrapped in Flux resources to avoid the earlier
  bootstrap deadlocks around namespaces and privileged resources
- Dex client secrets and password hashes live in Vault, not plain YAML
- Forgejo bootstraps against the in-cluster Dex service
