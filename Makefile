# Set sane defaults for Make
SHELL = bash
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

# Set default goal such that `make` runs `make help`
.DEFAULT_GOAL := help

# Build info
BUILDER = $(shell whoami)@$(shell hostname)
NOW = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Version control
VERSION = $(shell git describe --tags --dirty --always)
COMMIT = $(shell git rev-parse --short HEAD)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)

# Linker flags
PKG = $(shell head -n 1 go.mod | cut -c 8-)
VER = $(PKG)/pkg/version
LDFLAGS = -s -w \
	-X $(VER).Version=$(or $(VERSION),unknown) \
	-X $(VER).Commit=$(or $(COMMIT),unknown) \
	-X $(VER).Branch=$(or $(BRANCH),unknown) \
	-X $(VER).BuiltAt=$(NOW) \
	-X $(VER).Builder=$(BUILDER)
	
OS = $(shell uname -s)
ifeq ($(OS), Linux)
	OPENER=xdg-open
else
	OPENER=open
endif

# List of variables to fetch from .env or environment
VARIABLES := DEPLOY_APPNAME DEPLOY_HOSTNAME BASIC_AUTH_USERNAME BASIC_AUTH_PASSWORD

# Define a function to fetch a variable from .env or environment
define get_variable
    $(eval $1 := $(if $(value $1),$(value $1),$(shell grep -E "^$1=" .env | cut -d'=' -f2)))
endef

# Fetch the variables
$(foreach var,$(VARIABLES),$(call get_variable,$(var)))

.PHONY: all vet test build verify run up down distroless-build distroless-run local local-dev local-vet local-test local-cover local-run local-release-test local-release local-sign local-verify local-release-verify local-load-test install get-cosign-pub-key docker-login deploy-pre deploy-only deploy-post deploy-ip deploy-cert deploy-volume deploy-secrets deploy-launch deploy-first-time deploy-rollback deploy deploy-load-test pre-commit-install pre-commit-run pre-commit pre-reqs update-golang-version docs docs-generate docs-serve clean help

all: vet pre-commit clean test build verify run ## Run default workflow via Docker
local: local-update-deps local-vendor local-vet pre-commit clean local-test local-cover local-build local-sign local-verify local-run ## Run default workflow using locally installed Golang toolchain
local-dev: local-vendor local-vet clean local-build local-run ## Quick development workflow using locally installed Golang toolchain
local-release-verify: local-release local-sign local-verify ## Release and verify using locally installed Golang toolchain
pre-reqs: pre-commit-install ## Install pre-commit hooks and necessary binaries

vet: ## Run `go vet` in Docker
	docker build --target vet -f $(CURDIR)/Dockerfile -t toozej/ghouls:latest . 

test: ## Run `go test` in Docker
	docker build --target test -f $(CURDIR)/Dockerfile -t toozej/ghouls:latest . 
	@echo -e "\nStatements missing coverage"
	@grep -v -e " 1$$" c.out

build: ## Build Docker image, including running tests
	docker build -f $(CURDIR)/Dockerfile -t toozej/ghouls:latest . --no-cache

get-cosign-pub-key: ## Get ghouls Cosign public key from GitHub
	test -f $(CURDIR)/ghouls.pub || curl --silent https://raw.githubusercontent.com/toozej/ghouls/main/ghouls.pub -O

verify: get-cosign-pub-key ## Verify Docker image with Cosign
	cosign verify --key $(CURDIR)/ghouls.pub toozej/ghouls:latest

run: ## Run built Docker image
	docker run --rm --name ghouls --env-file $(CURDIR)/.env -p 8080:8080 -v ghouls:/data toozej/ghouls:latest

up: test build ## Run Docker Compose project with build Docker image
	docker compose -f docker-compose.yml down --remove-orphans
	docker compose -f docker-compose.yml pull
	docker compose -f docker-compose.yml up -d

down: ## Stop running Docker Compose project
	docker compose -f docker-compose.yml down --remove-orphans

distroless-build: ## Build Docker image using distroless as final base
	docker build -f $(CURDIR)/Dockerfile.distroless -t toozej/ghouls:distroless . 

distroless-run: ## Run built Docker image using distroless as final base
	docker run --rm --name ghouls -v $(CURDIR)/config:/config toozej/ghouls:distroless

local-update-deps: ## Run `go get -t -u ./...` to update Go module dependencies
	go get -t -u ./...

local-vet: ## Run `go vet` using locally installed golang toolchain
	go vet $(CURDIR)/...

local-vendor: ## Run `go mod vendor` using locally installed golang toolchain
	go mod tidy
	go mod vendor

local-test: ## Run `go test` using locally installed golang toolchain
	go test -coverprofile c.out -v $(CURDIR)/...
	@echo -e "\nStatements missing coverage"
	@grep -v -e " 1$$" c.out

local-cover: ## View coverage report in web browser
	go tool cover -html=c.out

local-build: ## Run `go build` using locally installed golang toolchain
	CGO_ENABLED=0 go build -o $(CURDIR)/out/ghouls -ldflags="$(LDFLAGS)" $(CURDIR)/

local-run: ## Run locally built binary
	if test -e $(CURDIR)/.env; then \
		export `cat $(CURDIR)/.env | xargs` && touch $(CURDIR)/data.json && $(CURDIR)/out/ghouls; \
	else \
		echo "no environment variables for ghouls project found in ./.env file. Exiting."; \
	fi

local-release-test: ## Build assets and test goreleaser config using locally installed golang toolchain and goreleaser
	goreleaser check
	goreleaser build --clean --snapshot

local-release: local-test docker-login ## Release assets using locally installed golang toolchain and goreleaser
	if test -e $(CURDIR)/ghouls.key && test -e $(CURDIR)/.env; then \
		export `cat $(CURDIR)/.env | xargs` && goreleaser release --clean; \
	else \
		echo "no cosign private key found at $(CURDIR)/ghouls.key. Cannot release."; \
	fi

local-sign: local-test ## Sign locally installed golang toolchain and cosign
	if test -e $(CURDIR)/ghouls.key && test -e $(CURDIR)/.env; then \
		export `cat $(CURDIR)/.env | xargs` && cosign sign-blob --key=$(CURDIR)/ghouls.key --output-signature=$(CURDIR)/ghouls.sig $(CURDIR)/out/ghouls; \
	else \
		echo "no cosign private key found at $(CURDIR)/ghouls.key. Cannot release."; \
	fi

local-verify: get-cosign-pub-key ## Verify locally compiled binary
	# cosign here assumes you're using Linux AMD64 binary
	cosign verify-blob --key $(CURDIR)/ghouls.pub --signature $(CURDIR)/ghouls.sig $(CURDIR)/out/ghouls

local-load-test: ## Run Vegeta binary to load test locally compiled binary
	echo "GET http://$(BASIC_AUTH_USERNAME):$(BASIC_AUTH_PASSWORD)@localhost:8080/" | vegeta attack -duration=5s | tee results.bin | vegeta report

install: local-build local-verify ## Install compiled binary to local machine
	sudo cp $(CURDIR)/out/ghouls /usr/local/bin/ghouls
	sudo chmod 0755 /usr/local/bin/ghouls

docker-login: ## Login to Docker registries used to publish images to
	if test -e $(CURDIR)/.env; then \
		export `cat $(CURDIR)/.env | xargs`; \
		echo $${DOCKERHUB_TOKEN} | docker login docker.io --username $${DOCKERHUB_USERNAME} --password-stdin; \
		echo $${QUAY_TOKEN} | docker login quay.io --username $${QUAY_USERNAME} --password-stdin; \
		echo $${GITHUB_GHCR_TOKEN} | docker login ghcr.io --username $${GITHUB_USERNAME} --password-stdin; \
	else \
		echo "No container registry credentials found, need to add them to ./.env. See README.md for more info"; \
	fi

deploy-pre:
	flyctl auth whoami || flyctl auth login
	sed -i 's/ghouls-example/$(DEPLOY_APPNAME)/g' fly.toml 

deploy-post:
	sed -i 's/$(DEPLOY_APPNAME)/ghouls-example/g' fly.toml 

deploy-ip: ## Allocate an IP address for deployment in fly.io
	flyctl ips list | grep v4 || flyctl ips allocate-v4

deploy-cert: ## Provision a SSL certificate for deployment in fly.io
	flyctl certs list | grep $(DEPLOY_HOSTNAME) || flyctl certs create "$(DEPLOY_HOSTNAME)" --yes

deploy-volume:
	flyctl volumes list | grep ghouls_data || flyctl volume create ghouls_data --region sea --size 1 --count 1 --yes

deploy-secrets: ## Deploy secrets to fly.io
	@if test -e $(CURDIR)/.env; then \
		echo "Trying to load app secrets from .env file"; \
		while read -r SECRET; do \
			if [[ "$${SECRET}" =~ .*BASIC_AUTH.*|.*SECRET.* ]]; then \
				flyctl secrets set --stage $${SECRET}; \
			fi; \
		done < $(CURDIR)/.env; \
	else \
		echo "Trying to load app secrets from environment"; \
		while read -r SECRET; do \
			if [[ "$${SECRET}" =~ .*BASIC_AUTH.*|.*SECRET.* ]]; then \
				flyctl secrets set --stage $${SECRET}; \
			fi; \
		done < <(env); \
	fi
	flyctl config env

deploy-launch:
	flyctl launch --local-only --now
	flyctl scale count 1 --yes

deploy-first-time: deploy-pre deploy-volume deploy-secrets deploy-launch deploy-ip deploy-cert deploy-post ## First time deploy to fly.io

deploy-only: ## Deploy built runtime image to fly.io
	flyctl deploy $(CURDIR) --remote-only --now
	flyctl status

deploy-rollback: deploy-pre ## Rollback fly.io to last working image
	ROLLBACK_IMAGE=`flyctl releases --image | grep -v failed | sed -n '2p' | awk '{print $$7}'`
	flyctl deploy -i $$ROLLBACK_IMAGE --now 

deploy: deploy-pre deploy-secrets deploy-only deploy-post ## Deploy to fly.io

deploy-load-test: ## Run Vegeta binary to load test deployed site
	echo "GET http://$(BASIC_AUTH_USERNAME):$(BASIC_AUTH_PASSWORD)@$(DEPLOY_HOSTNAME)/" | vegeta attack -duration=5s | tee results.bin | vegeta report

pre-commit: pre-commit-install pre-commit-run ## Install and run pre-commit hooks

pre-commit-install: ## Install pre-commit hooks and necessary binaries
	# golangci-lint
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	# goimports
	go install golang.org/x/tools/cmd/goimports@latest
	# gosec
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	# staticcheck
	go install honnef.co/go/tools/cmd/staticcheck@latest
	# go-critic
	go install github.com/go-critic/go-critic/cmd/gocritic@latest
	# structslop
	# go install github.com/orijtech/structslop/cmd/structslop@latest
	# shellcheck
	command -v shellcheck || sudo dnf install -y ShellCheck || sudo apt install -y shellcheck
	# checkmake
	go install github.com/mrtazz/checkmake/cmd/checkmake@latest
	# goreleaser
	go install github.com/goreleaser/goreleaser/v2@latest
	# syft
	command -v syft || curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin
	# cosign
	go install github.com/sigstore/cosign/cmd/cosign@latest
	# go-licenses
	go install github.com/google/go-licenses@latest
	# go vuln check
	go install golang.org/x/vuln/cmd/govulncheck@latest
	# vegeta load testing tool
	go install github.com/tsenart/vegeta@latest
	# gokart static vulnerability check
	go install github.com/praetorian-inc/gokart@latest
	# install and update pre-commits
	pre-commit install
	pre-commit autoupdate

pre-commit-run: ## Run pre-commit hooks against all files
	pre-commit run --all-files
	# manually run the following checks since their pre-commits aren't working or don't exist
	# gokart disabled until https://github.com/praetorian-inc/gokart/issues/92 is fixed
	# gokart ./...
	go-licenses report github.com/toozej/ghouls/cmd/ghouls --ignore `go list std | awk 'NR > 1 { printf(",") } { printf("%s",$$0) } END { print "" }'`
	govulncheck ./...

update-golang-version: ## Update to latest Golang version across the repo
	@VERSION=`curl -s "https://go.dev/dl/?mode=json" | jq -r '.[0].version' | sed 's/go//' | cut -d '.' -f 1,2`; \
	echo "Updating Golang to $$VERSION"; \
	./scripts/update_golang_version.sh $$VERSION

docs: docs-generate docs-serve ## Generate and serve documentation

docs-generate:
	docker build -f $(CURDIR)/Dockerfile.docs -t toozej/ghouls:docs . 
	docker run --rm --name ghouls-docs -v $(CURDIR):/package -v $(CURDIR)/docs:/docs toozej/ghouls:docs

docs-serve: ## Serve documentation on http://localhost:9000
	docker run -d --rm --name ghouls-docs-serve -p 9000:3080 -v $(CURDIR)/docs:/data thomsch98/markserv
	$(OPENER) http://localhost:9000/docs.md
	@echo -e "to stop docs container, run:\n"
	@echo "docker kill ghouls-docs-serve"

clean: ## Remove any locally compiled binaries
	rm -f $(CURDIR)/ghouls

help: ## Display help text
	@grep -E '^[a-zA-Z_-]+ ?:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
