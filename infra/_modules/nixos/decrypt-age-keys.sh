#!/usr/bin/env sh

set -euo pipefail -x

mkdir -p ./var/lib/secrets

umask 0177
sops \
    --extract '["age_key"]' \
    --decrypt "${SOPS_FILE}" \
    > ./var/lib/secrets/age
umask 0022
