.POSIX:
.PHONY: default compose infra bootstrap platform apps test fmt tidy update

env ?= $(shell ls infra | fzf --prompt "Select environment: ")

default: infra platform apps

compose:
	docker compose up --build --detach

infra:
	cd infra/${env} && terragrunt apply --all

bootstrap: platform secrets

platform:
	toolbox gitops \
		--path platform/${env} \
		--hosts-file infra/_modules/nixos/hosts.json \
		--host kube-1

secrets:
	toolbox secrets \
		--settings settings.yaml \
		--hosts-file infra/_modules/nixos/hosts.json \
		--host kube-1

apps:
	# TODO multiple env
	@temporal workflow start \
		--workflow-id apps-manual \
		--task-queue cloudlab \
		--type Apps \
		--input '{ "url": "/usr/local/src/cloudlab", "revision": "master", "registry": "registry.127.0.0.1.sslip.io", "cluster": "local" }'
	@temporal workflow result --workflow-id apps-manual

test:
	cd test && go test

fmt:
	nixfmt flake.nix
	yamlfmt \
		--exclude infra/_modules/cluster/roles/secrets/vars/main.yml \
		--exclude infra/*/secrets.yaml \
		.
	terragrunt hcl format
	cd infra/_modules && tofu fmt -recursive
	cd infra/_modules/tfstate && go fmt ./...
	cd test && go fmt ./...

tidy: fmt
	cd infra && terragrunt init --backend=false --lock=false --all

update:
	nix flake update

clean:
	docker compose down --remove-orphans --volumes
	k3d cluster delete local
