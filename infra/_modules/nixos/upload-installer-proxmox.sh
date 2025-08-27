#!/usr/bin/env sh

# TODO auto run this and avoid chicken and eggs
# Currently running ./upload-installer-proxmox.sh manually

set -euo pipefail -x

PROMOX_HOST="proxmox"

nix build -L .#nixosConfigurations.installer.config.system.build.isoImage
ISO="$(readlink -e result/iso/*.iso)"

echo "Uploading ${ISO} to Proxmox"
scp "${ISO}" root@proxmox:/var/lib/vz/template/iso/nixos-installer.iso

rm result
