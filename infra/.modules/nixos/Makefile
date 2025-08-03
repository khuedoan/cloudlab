.POSIX:

.PHONY: *
default: switch

switch:
	sudo nixos-rebuild \
		--flake '.#netamos' \
		--impure \
		switch

update:
	nix flake update

test:
	@nix \
		--experimental-features 'nix-command flakes' \
		run \
		'.#nixosConfigurations.test.config.system.build.vmWithDisko'
