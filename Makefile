.POSIX:
.PHONY: default compose infra bootstrap vendor platform secrets test fmt tidy update

env ?= $(shell ls infra | fzf --prompt "Select environment: ")
KUBECONFIG ?= $(shell terragrunt output --working-dir infra/${env}/nixos -raw kubeconfig_path 2>/dev/null)

default: infra platform

compose:
	docker compose up --build --detach

infra:
	cd infra/${env} && terragrunt apply --all

bootstrap: vendor platform secrets

vendor:
	KUBECONFIG="${KUBECONFIG}" toolbox vendor \
		--settings settings.yaml

platform:
	KUBECONFIG="${KUBECONFIG}" toolbox gitops \
		--path platform/${env}

secrets:
	KUBECONFIG="${KUBECONFIG}" toolbox secrets \
		--settings settings.yaml

test:
	cd test && CLOUDLAB_ENV=${env} go test

fmt:
	treefmt
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
