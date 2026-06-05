# Backup and restore

VolSync backs up PVCs listed in `settings.yaml` under `backups.volumes`.
Vault must contain `secret/backup/restic#password` and the S3 values under
`secret/backup/s3`. Keep the restic password outside the cluster; changing it
makes existing repositories unreadable.

## Setup

Backup buckets are managed externally (either using Terraform or created manually),
this is intentional to prevent human error from accidentally deleting the buckets (e.g., with `terragrunt destroy`).

Publish platform resources and secrets first:

```sh
make platform env=staging
make secrets
toolbox backup setup --env staging
```

To set up one configured volume:

```sh
toolbox backup setup --env staging --volume finance/actualbudget
```

## Restore

Recreate the target PVC first, then run:

```sh
toolbox backup restore --env staging --volume finance/actualbudget
```

Restore suspends matching Flux HelmReleases, scales selected namespaces down,
waits for pods to detach, applies a `Direct` restore with `enableFileDeletion`,
waits for VolSync to return to `WaitingForManual`, scales workloads up, then
resumes only the HelmReleases it suspended. On failure after suspension, inspect
the cluster before resuming Flux or scaling workloads back up.
