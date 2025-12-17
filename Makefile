#!/usr/bin/make -f

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')

# don't override user values
ifeq (,$(VERSION))
  VERSION := $(shell git describe --tags)
  # if VERSION is empty, then populate it with branch's name and raw commit hash
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

PACKAGES_SIMTEST=$(shell go list ./... | grep '/simulation')
LEDGER_ENABLED ?= true
SDK_PACK := $(shell go list -m github.com/cosmos/cosmos-sdk | sed  's/ /\@/g')
TM_VERSION := $(shell go list -m github.com/cometbft/cometbft | sed 's:.* ::') # grab everything after the space in "github.com/cometbft/cometbft v0.37.2"
DOCKER := $(shell which docker)
DOCKER_BUF := $(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace bufbuild/buf:1.28.1
BUILDDIR ?= $(CURDIR)/build
GOBIN = $(shell go env GOPATH)/bin
GOARCH = $(shell go env GOARCH)
GOOS = $(shell go env GOOS)

# Improved Go version checking
GO_VERSION := $(shell go version | sed 's/go version go//' | cut -d' ' -f1)
GO_MAJOR_VERSION := $(shell echo $(GO_VERSION) | cut -d. -f1)
GO_MINOR_VERSION := $(shell echo $(GO_VERSION) | cut -d. -f2)
GO_MINIMUM_MAJOR_VERSION := 1
GO_MINIMUM_MINOR_VERSION := 22

export GO111MODULE = on

# process build tags
build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger
      endif
    endif
  endif
endif

ifeq (cleveldb,$(findstring cleveldb,$(VERANA_BUILD_OPTIONS)))
  build_tags += gcc cleveldb
endif
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

# process linker flags - Updated for Cosmos SDK v0.53.x
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=veranad \
         -X github.com/cosmos/cosmos-sdk/version.AppName=veranad \
         -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
         -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
         -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)" \
          -X github.com/cometbft/cometbft/version.TMCoreSemVer=$(TM_VERSION)

ifeq (cleveldb,$(findstring cleveldb,$(VERANA_BUILD_OPTIONS)))
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=cleveldb
endif
ifeq ($(LINK_STATICALLY),true)
  ldflags += -linkmode=external -extldflags "-Wl,-z,muldefs -static"
endif
ifeq (,$(findstring nostrip,$(VERANA_BUILD_OPTIONS)))
  ldflags += -w -s
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'
# check for nostrip option
ifeq (,$(findstring nostrip,$(VERANA_BUILD_OPTIONS)))
  BUILD_FLAGS += -trimpath
endif

###############################################################################
###                           Version Checking                              ###
###############################################################################

check_version:
	@echo "Checking Go version..."
	@echo "Current Go version: $(GO_VERSION)"
	@echo "Required Go version: >= $(GO_MINIMUM_MAJOR_VERSION).$(GO_MINIMUM_MINOR_VERSION)"
ifeq ($(shell [ $(GO_MAJOR_VERSION) -gt $(GO_MINIMUM_MAJOR_VERSION) ] || ([ $(GO_MAJOR_VERSION) -eq $(GO_MINIMUM_MAJOR_VERSION) ] && [ $(GO_MINOR_VERSION) -ge $(GO_MINIMUM_MINOR_VERSION) ]) && echo true),true)
	@echo "✅ Go version $(GO_VERSION) meets requirements"
else
	@echo "❌ ERROR: Go version $(GO_VERSION) is too old. Please upgrade to Go $(GO_MINIMUM_MAJOR_VERSION).$(GO_MINIMUM_MINOR_VERSION)+"
	@exit 1
endif

###############################################################################
###                              Building                                   ###
###############################################################################

all: install

go-mod-cache: go.sum
	@echo "--> Download go modules to local cache"
	@go mod download

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify

clean:
	rm -rf $(CURDIR)/artifacts/
	rm -rf $(BUILDDIR)/

distclean: clean
	rm -rf vendor/

install: check_version go.sum
	@echo "--> Installing veranad"
	go install -mod=readonly $(BUILD_FLAGS) ./cmd/veranad

build: check_version go.sum
	@echo "--> Building veranad"
	go build $(BUILD_FLAGS) -o $(BUILDDIR)/veranad ./cmd/veranad

build-linux: check_version go.sum
	@echo "--> Building veranad for Linux"
	LEDGER_ENABLED=false GOOS=linux GOARCH=amd64 $(MAKE) build

release: install
	@echo "--> Creating release package"
	mkdir -p release
ifeq (${OS},Windows_NT)
	tar -czvf release/veranad-${GOOS}-${GOARCH}.tar.gz --directory=$(GOBIN) veranad.exe
else
	tar -czvf release/veranad-${GOOS}-${GOARCH}.tar.gz --directory=$(GOBIN) veranad
endif

###############################################################################
###                                Linting                                  ###
###############################################################################

lint:
	@echo "--> Running linter"
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.2 run --timeout=10m

format:
	@echo "--> Formatting Go files"
	@go install mvdan.cc/gofumpt@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.2
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" -not -path "./tests/mocks/*" -not -name "*.pb.go" -not -name "*.pb.gw.go" -not -name "*.pulsar.go" -not -path "./crypto/keys/secp256k1/*" | xargs gofumpt -w -l
	golangci-lint run --fix

###############################################################################
###                           Tests & Simulation                            ###
###############################################################################

test: test-unit
test-all: test-unit test-ledger-mock test-race test-cover

TEST_PACKAGES=./...
TEST_TARGETS := test-unit test-unit-amino test-unit-proto test-ledger-mock test-race test-ledger test-race

# Test runs-specific rules. To add a new test target, just add
# a new rule, customise ARGS or TEST_PACKAGES ad libitum, and
# append the new rule to the TEST_TARGETS list.
test-unit: ARGS=-tags='cgo ledger test_ledger_mock norace'
test-unit-amino: ARGS=-tags='ledger test_ledger_mock test_amino norace'
test-ledger: ARGS=-tags='cgo ledger norace'
test-ledger-mock: ARGS=-tags='ledger test_ledger_mock norace'
test-race: ARGS=-race -tags='cgo ledger test_ledger_mock'
test-race: TEST_PACKAGES=$(PACKAGES_NOSIMULATION)
$(TEST_TARGETS): run-tests

# check-* compiles and collects tests without running them
# note: go test -c doesn't support multiple packages yet (https://github.com/golang/go/issues/15513)
CHECK_TEST_TARGETS := check-test-unit check-test-unit-amino
check-test-unit: ARGS=-tags='cgo ledger test_ledger_mock norace'
check-test-unit-amino: ARGS=-tags='ledger test_ledger_mock test_amino norace'
$(CHECK_TEST_TARGETS): EXTRA_ARGS=-run=none
$(CHECK_TEST_TARGETS): run-tests

run-tests:
ifneq (,$(shell which tparse 2>/dev/null))
	go test -mod=readonly -json $(ARGS) $(EXTRA_ARGS) $(TEST_PACKAGES) | tparse
else
	go test -mod=readonly $(ARGS)  $(EXTRA_ARGS) $(TEST_PACKAGES)
endif

test-coverage:
	@echo "Running tests with coverage..."
	@export VERSION=$(VERSION); bash -x scripts/test_cover.sh

benchmark:
	@go test -mod=readonly -bench=. $(PACKAGES_NOSIMULATION)

.PHONY: run-tests test test-all $(TEST_TARGETS) test-coverage benchmark

###############################################################################
###                              Protobuf                                   ###
###############################################################################

# Updated protobuf generation using custom script
PROTO_BUILDER_IMAGE := ghcr.io/cosmos/proto-builder:0.17.0

proto-all: proto-clean proto-deps proto-gen proto-swagger proto-format
	@echo "🎉 Protobuf generation completed successfully!"

proto-gen:
	@./scripts/protocgen.sh

proto-swagger:
	@echo "Generating Swagger documentation..."
	@cd proto && buf generate --template buf.gen.swagger.yaml

proto-ts:
	@echo "Generating TypeScript protobuf package..."
	@cd proto && buf generate --template buf.gen.ts.yaml
	@cd ts-proto && npm run build

proto-deps:
	@echo "Updating proto dependencies..."
	@cd proto && buf dep update

proto-clean:
	@echo "Cleaning generated protobuf files..."
	@find x -name "*.pb.go" -delete 2>/dev/null || true
	@find x -name "*.pb.gw.go" -delete 2>/dev/null || true
	@find api -name "*.pulsar.go" -delete 2>/dev/null || true
	@find api -name "*_grpc.pb.go" -delete 2>/dev/null || true
	@rm -f docs/static/openapi.yml

proto-format:
	@echo "Formatting generated code..."
	@find x -name "*.pb.go" -exec gofmt -w {} + 2>/dev/null || true
	@find api -name "*.pulsar.go" -exec gofmt -w {} + 2>/dev/null || true

proto-lint:
	@echo "🔍 Linting protobuf files..."
	@$(DOCKER) run --rm -v $(CURDIR):/workspace -w /workspace bufbuild/buf:1.28.1 lint

proto-format-buf:
	@echo "🎨 Formatting protobuf files..."
	@$(DOCKER) run --rm -v $(CURDIR):/workspace -w /workspace bufbuild/buf:1.28.1 format -w

proto-breaking:
	@echo "🔍 Checking for breaking changes..."
	@$(DOCKER) run --rm -v $(CURDIR):/workspace -w /workspace bufbuild/buf:1.28.1 breaking --against '.git#branch=main'

# Legacy proto generation (using docker directly)
proto-gen-docker:
	@echo "🏗️  Generating protobuf files with docker..."
	@$(DOCKER) run --rm -v $(CURDIR):/workspace -w /workspace $(PROTO_BUILDER_IMAGE) \
		find proto -name '*.proto' -path "*/verana/*" -exec \
		protoc \
		--proto_path=proto \
		--proto_path=third_party/proto \
		--gocosmos_out=plugins=interfacetype+grpc,Mgoogle/protobuf/any.proto=github.com/cosmos/cosmos-sdk/codec/types:. \
		{} \;

.PHONY: proto-all proto-gen proto-swagger proto-ts proto-deps proto-clean proto-format proto-lint proto-format-buf proto-breaking proto-gen-docker

###############################################################################
###                              Simulation                                 ###
###############################################################################

test-sim-nondeterminism:
	@echo "Running non-determinism test..."
	@go test -mod=readonly $(SIMAPP) -run TestAppStateDeterminism -Enabled=true \
       -NumBlocks=100 -BlockSize=200 -Commit=true -Period=0 -v -timeout 24h

test-sim-custom-genesis-fast:
	@echo "Running custom genesis simulation..."
	@echo "By default, ${HOME}/.verana/config/genesis.json will be used."
	@go test -mod=readonly $(SIMAPP) -run TestFullAppSimulation -Genesis=${HOME}/.verana/config/genesis.json \
       -Enabled=true -NumBlocks=100 -BlockSize=200 -Commit=true -Seed=99 -Period=5 -v -timeout 24h

test-sim-import-export: runsim
	@echo "Running application import/export simulation. This may take several minutes..."
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 5 TestAppImportExport

test-sim-after-import: runsim
	@echo "Running application simulation-after-import. This may take several minutes..."
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 5 TestAppSimulationAfterImport

test-sim-custom-genesis-multi-seed: runsim
	@echo "Running multi-seed custom genesis simulation..."
	@echo "By default, ${HOME}/.verana/config/genesis.json will be used."
	@$(BINDIR)/runsim -Genesis=${HOME}/.verana/config/genesis.json -SimAppPkg=$(SIMAPP) -ExitOnFail 400 5 TestFullAppSimulation

test-sim-multi-seed-long: runsim
	@echo "Running long multi-seed application simulation. This may take awhile!"
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 500 50 TestFullAppSimulation

test-sim-multi-seed-short: runsim
	@echo "Running short multi-seed application simulation. This may take awhile!"
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 10 TestFullAppSimulation

test-sim-benchmark-invariants:
	@echo "Running simulation invariant benchmarks..."
	@go test -mod=readonly $(SIMAPP) -benchmem -bench=BenchmarkInvariants -run=^$ \
    -Enabled=true -NumBlocks=1000 -BlockSize=200 \
    -Period=1 -Commit=true -Seed=57 -v -timeout 24h

SIM_NUM_BLOCKS ?= 500
SIM_BLOCK_SIZE ?= 200
SIM_COMMIT ?= true

test-sim-benchmark:
	@echo "Running application benchmark for numBlocks=$(SIM_NUM_BLOCKS), blockSize=$(SIM_BLOCK_SIZE). This may take awhile!"
	@go test -mod=readonly -benchmem -run=^$$ $(SIMAPP) -bench ^BenchmarkFullAppSimulation$$  \
       -Enabled=true -NumBlocks=$(SIM_NUM_BLOCKS) -BlockSize=$(SIM_BLOCK_SIZE) -Commit=$(SIM_COMMIT) -timeout 24h

test-sim-profile:
	@echo "Running application benchmark for numBlocks=$(SIM_NUM_BLOCKS), blockSize=$(SIM_BLOCK_SIZE). This may take awhile!"
	@go test -mod=readonly -benchmem -run=^$$ $(SIMAPP) -bench ^BenchmarkFullAppSimulation$$ \
       -Enabled=true -NumBlocks=$(SIM_NUM_BLOCKS) -BlockSize=$(SIM_BLOCK_SIZE) -Commit=$(SIM_COMMIT) -timeout 24h -cpuprofile cpu.out -memprofile mem.out

.PHONY: \
test-sim-nondeterminism \
test-sim-custom-genesis-fast \
test-sim-import-export \
test-sim-after-import \
test-sim-custom-genesis-multi-seed \
test-sim-multi-seed-short \
test-sim-multi-seed-long \
test-sim-benchmark-invariants \
test-sim-profile \
test-sim-benchmark

###############################################################################
###                                 Docker                                  ###
###############################################################################

docker-build:
	@echo "Building Docker image..."
	docker build -t verana:local .

docker-build-debug:
	@echo "Building Docker debug image..."
	docker build -t verana:local-debug -f Dockerfile.debug .

.PHONY: docker-build docker-build-debug

###############################################################################
###                            Local Network                                ###
###############################################################################

localnet-init:
	@echo "Initializing local network..."
	./scripts/localnet-init.sh

localnet-start:
	@echo "Starting local network..."
	./scripts/localnet-start.sh

localnet-stop:
	@echo "Stopping local network..."
	./scripts/localnet-stop.sh

.PHONY: localnet-init localnet-start localnet-stop

###############################################################################
###                                 Help                                    ###
###############################################################################

help:
	@echo "Available targets:"
	@echo ""
	@echo "Building:"
	@echo "  install           - Install veranad binary"
	@echo "  build             - Build veranad binary"
	@echo "  build-linux       - Build veranad for Linux"
	@echo "  clean             - Clean build artifacts"
	@echo "  release           - Create release package"
	@echo ""
	@echo "Development:"
	@echo "  lint              - Run linter"
	@echo "  format            - Format code"
	@echo "  test              - Run unit tests"
	@echo "  test-all          - Run all tests"
	@echo "  test-coverage     - Run tests with coverage"
	@echo ""
	@echo "Protobuf:"
	@echo "  proto-all         - Generate all protobuf files"
	@echo "  proto-gen         - Generate Go protobuf files"
	@echo "  proto-swagger     - Generate Swagger docs"
	@echo "  proto-ts          - Generate TypeScript proto package (ts-proto)"
	@echo "  proto-clean       - Clean generated files"
	@echo "  proto-lint        - Lint protobuf files"
	@echo ""
	@echo "Simulation:"
	@echo "  test-sim-*        - Various simulation tests"
	@echo ""
	@echo "Local Network:"
	@echo "  localnet-init     - Initialize local network"
	@echo "  localnet-start    - Start local network"
	@echo "  localnet-stop     - Stop local network"

.PHONY: help check_version go-mod-cache go.sum clean distclean install build build-linux release lint format all
