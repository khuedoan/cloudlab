.POSIX:
.PHONY: default compose infra platform apps test fmt tidy update

env ?= local

default: infra platform apps

compose:
	docker compose up --build --detach

infra:
	cd infra/${env} && terragrunt apply --all


platform:
	# TODO don't hard code registry
	cd platform/${env} && oras push --format=json docker.io/khuedoan/platform-manifests:${env} .

apps:
	# TODO multiple env
	@temporal workflow start \
		--workflow-id apps-manual \
		--task-queue cloudlab \
		--type Apps \
		--input '{ "url": "/usr/local/src/cloudlab", "revision": "master", "registry": "registry.127.0.0.1.sslip.io", "cluster": "local" }'
	@temporal workflow result --workflow-id apps-manual

test:
	cd controller && go test ./...
	cd test && go test

fmt:
	nixfmt flake.nix
	yamlfmt \
		--exclude infra/modules/cluster/roles/secrets/vars/main.yml \
		--exclude infra/*/secrets.yaml \
		.
	terragrunt hcl format
	tofu fmt -recursive
	cd controller && go fmt ./...
	cd infra/modules/tfstate && go fmt ./...
	cd test && go fmt ./...

tidy: fmt
	cd infra && terragrunt init --backend=false --lock=false --all

update:
	nix flake update

clean:
	docker compose down --remove-orphans --volumes
	k3d cluster delete local
