# Verana Blockchain

[![Project Status: Active -- The project has reached a stable, usable state and is being actively developed.](https://img.shields.io/badge/repo%20status-Active-green.svg?style=flat-square)](https://www.repostatus.org/#active)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue?style=flat-square&logo=go)](https://pkg.go.dev/github.com/verana-labs/verana)
[![Go Report Card](https://goreportcard.com/badge/github.com/verana-labs/verana?style=flat-square)](https://goreportcard.com/report/github.com/verana-labs/verana)
[![Version](https://img.shields.io/github/tag/verana-labs/verana.svg?style=flat-square)](https://github.com/verana-labs/verana/releases/latest)
[![License: Apache-2.0](https://img.shields.io/github/license/verana-labs/verana.svg?style=flat-square)](https://github.com/verana-labs/verana/blob/main/LICENSE)
[![Discord](https://badgen.net/badge/icon/discord?icon=discord&label)](https://discord.com/invite/edjaFn252q)

Verana Blockchain is a Verifiable Public Registry (VPR) providing foundational infrastructure for decentralized trust ecosystems. As a sovereign Layer 1 appchain built on the Cosmos SDK, Verana enables trustless verification of credentials and services across ecosystems through a standardized trust registry framework.

Verana serves as a public registry of registries, allowing organizations to create and manage their own ecosystems with defined credential schemas, roles for issuers, verifiers, and grantors, and custom business models. The platform facilitates trust resolution, enabling applications to validate roles and permissions in real time through a standardized API.

Key capabilities:

- **Ecosystem Management**: Create and govern trust ecosystems, a public registry of registries, across trust domains
- **Corporation & Governance Framework**: Register on-chain corporations and publish versioned governance framework documents
- **Credential Schemas**: Define credential schemas with custom issuance and verification policies
- **Participant Roles & Permissions**: Manage issuer, verifier, and grantor roles and their permissions
- **Delegated Operations**: Authorize operators to execute messages on behalf of an authority (AUTHZ-CHECK delegation)
- **Tokenized Trust Deposit**: Built-in economic incentives, yield, and slashing for ecosystem participants
- **Supporting Registries**: On-chain exchange-rate and content-digest primitives
- **Trust Resolution (roadmap)**: Standard API supporting the Trust Registry Query Protocol (TRQP)

Verana is designed to bridge the gap between centralized trust models and the decentralized web, enabling trustworthy digital interactions across ecosystems while preserving privacy and sovereignty.

For frontend transaction signing debugging, see the Amino sign bench notes in `ts-proto/test/README.md`.

## Modules

Verana is composed of the following Cosmos SDK modules (spec v4-rc2):

| Module | Path | Spec | Responsibility |
|--------|------|------|----------------|
| Ecosystem | `x/ec` | MOD-EC | Trust ecosystems; a registry of registries (formerly Trust Registry) |
| Corporation | `x/co` | MOD-CO | On-chain corporations that own ecosystems and govern via group policies |
| Governance Framework | `x/gf` | MOD-GF | Governance framework documents and versioning |
| Credential Schema | `x/cs` | MOD-CS | Credential schema definitions with issuance/verification policies |
| Participant | `x/pp` | MOD-PP | Participant roles and permissions: issuer, verifier, grantor (formerly Permission) |
| Delegation | `x/de` | MOD-DE | Operator authorizations that delegate message execution (AUTHZ-CHECK) |
| Trust Deposit | `x/td` | MOD-TD | Economic trust deposits, yield, and slashing |
| Exchange Rate | `x/xr` | MOD-XR | Token/credit exchange-rate management |
| Digest | `x/di` | MOD-DI | Content digest registry |

## ⚠️ Devnet Status (Temporary)

> **This section is temporary.** It reflects module readiness for the current devnet
> bring-up against spec v4-rc2 and will be removed as the remaining v4-rc2 PRs merge.

All nine modules are wired into the chain, so every transaction is submittable on devnet.
What differs is how far each module has been aligned with the v4-rc2 entity model:

**Built to the v4-rc2 entity model (current devnet focus)**

- **Ecosystem** (`x/ec`): create, update, and archive ecosystems; attach governance framework documents
- **Corporation** (`x/co`): create and update corporations
- **Governance Framework** (`x/gf`): add governance framework documents and promote active versions
- **Delegation** (`x/de`): grant and revoke operator authorizations (authz bootstrap only)

**Implemented but predating v4-rc2, under active rework (use with caution)**

- **Credential Schema** (`x/cs`)
- **Participant** (`x/pp`, formerly Permission)
- **Trust Deposit** (`x/td`)
- **Digest** (`x/di`)
- **Exchange Rate** (`x/xr`)

These predate the v4-rc2 entity rename and are being reworked and re-journeyed
(issues #308-#316, #329). Once that lands, this section is removed and the modules fold
into the descriptions above.

## System Requirements

These system specifications have been tested and are recommended for running a Verana node:

- Quad Core or larger AMD or Intel (amd64) CPU
- 32GB RAM
- 1TB SSD/NVMe Storage
- 50MBPS+ bidirectional internet connection

While Verana can run on lower-spec hardware, you may experience reduced performance or stability issues.

## Prerequisites

- **Go 1.22+** ([Installation Guide](https://golang.org/doc/install))
- **Docker** (optional, for local multi-validator network)
- **jq** (optional, for JSON parsing in scripts)

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/verana-labs/verana.git
cd verana
```

### 2. Install Dependencies

```bash
# Download Go modules
go mod download

# Verify dependencies
go mod verify
```

### 3. Build and Install

```bash
# Install the veranad binary to $GOPATH/bin
make install

# Or build without installing
make build

# Verify installation
veranad version
```

The `veranad` binary will be installed to `$GOPATH/bin`. Make sure `$GOPATH/bin` is in your `PATH`.

## Development

### For Developers Modifying Protobuf Files

**Note:** The following steps are only required if you are modifying `.proto` files or contributing to the codebase. Most users can skip this section.

If you need to modify protobuf definitions or regenerate generated code:

#### Install Ignite CLI v28.10.0

```bash
# Download Ignite v28.10.0
curl https://get.ignite.com/cli@v28.10.0 | bash

# Move to a location in your PATH
sudo mv ignite /usr/local/bin/ignite
# Or on some systems:
# mv ignite ~/.local/bin/ignite

# Verify installation
ignite version
```

You should see Ignite CLI version `v28.x.y` and Cosmos SDK v0.50.x.

#### Generate Protobuf Files

After making changes to any `.proto` files:

```bash
# Generate all protobuf files (Go, Swagger, TypeScript)
make proto-all

# Or generate individually:
make proto-gen          # Generate Go protobuf files
make proto-swagger      # Generate Swagger/OpenAPI docs
make proto-ts          # Generate TypeScript proto package
make proto-clean        # Clean generated files
```

#### Generate OpenAPI Documentation

```bash
# Generate OpenAPI documentation
ignite generate openapi --clear-cache --enable-proto-vendor

# Update version in generated file
VER=$(veranad version)
FILE="./docs/static/openapi.yml"

sed -i '' -E \
  -e "s/(\"version\"[[:space:]]*:[[:space:]]*)\"version not set\"/\\1\"$VER\"/" \
  -e "s/^([[:space:]]*version[[:space:]]*:[[:space:]]*)\"?version not set\"?/\\1\"$VER\"/" \
  "$FILE"
```

## Starting the Blockchain

### Option 1: Single Validator (Quick Start)

```bash
# Initialize and start a single validator chain
./scripts/setup_primary_validator.sh
```

This script will:
- Initialize the chain with chain ID `vna-local-1`
- Create a validator account (`cooluser`)
- Fund the account with genesis tokens
- Configure gas prices and CORS
- Start the blockchain node

The chain will be accessible at:
- **RPC**: `http://localhost:26657`
- **REST API**: `http://localhost:1317`
- **gRPC**: `localhost:9090`
- **gRPC-Web**: `localhost:9091`

### Option 2: Multi-Validator Network (Docker)

For testing with multiple validators:

```bash
# Build Docker image
cd local-test
./build.sh

# Start 3-validator network
./setup-validators.sh

# Stop network
./cleanup.sh
```

See [local-test/setup-guide.md](local-test/setup-guide.md) for detailed instructions.

### Option 3: Manual Setup

```bash
# Initialize chain
veranad init mymoniker --chain-id vna-local-1

# Add validator key
veranad keys add validator --keyring-backend test

# Add genesis account
veranad genesis add-genesis-account validator 1000000000000000000000uvna --keyring-backend test

# Create genesis transaction
veranad genesis gentx validator 1000000000uvna --chain-id vna-local-1 --keyring-backend test

# Collect genesis transactions
veranad genesis collect-gentxs

# Start the chain
veranad start
```

## Common Make Commands

```bash
# Building
make install          # Install veranad binary
make build            # Build binary to build/ directory
make build-linux      # Build for Linux
make clean            # Clean build artifacts

Notes:
- `make build` on macOS/Linux keeps Ledger enabled when your toolchain supports it.
- `make build-linux` cross-compiles to Linux from macOS with `LEDGER_ENABLED=false` to avoid missing CGO/HID deps.
- Release binaries are produced without Ledger; build from source on Linux with `LEDGER_ENABLED=true make build` if you need Ledger support.

# Development
make lint             # Run linter
make format           # Format code
make test             # Run unit tests
make test-all         # Run all tests
make test-coverage    # Run tests with coverage

# Protobuf
make proto-all        # Generate all protobuf files
make proto-gen        # Generate Go protobuf files
make proto-swagger    # Generate Swagger docs
make proto-ts         # Generate TypeScript proto package
make proto-clean      # Clean generated files
make proto-lint       # Lint protobuf files

# Help
make help             # Show all available commands
```

## Git Hooks (Lint-Only)

To catch new lint issues without reformatting files, enable the repo hook:

```bash
git config core.hooksPath scripts/githooks
```

The hook runs `golangci-lint` with `--new-from-rev=<merge-base>` (against `origin/main`) and does not modify files. It also blocks whitespace-only diffs for any staged files. For TypeScript, it checks staged files with `eslint` or `prettier` if available.

## Testing

### Unit Tests

```bash
# Run unit tests
make test

# Run all tests (unit, ledger, race)
make test-all

# Run with coverage
make test-coverage
```

### Test Harness

The Verana test harness is a comprehensive end-to-end testing framework that validates all Verana blockchain modules and their interactions through realistic user journeys. It includes:

- **19 Journey Tests**: Complete end-to-end workflows covering trust registry creation, credential issuance, permission management, DID operations, and more
- **TD Yield Simulations**: Economic simulations that test Trust Deposit yield distribution under different funding scenarios and verify protocol health
- **Automated Test Execution**: Scripts to run individual journeys or execute the full test suite sequentially

The test harness simulates real-world usage patterns, ensuring that all Verana features work correctly together. Each journey represents a complete user workflow, from account setup through complex multi-step operations.

For detailed information on running journeys, configuring the test environment, and understanding simulation results, see the **[Test Harness & Simulations Guide](testharness/README.md)**.

**Quick Start:**

```bash
# Run a specific journey
cd testharness
./scripts/setup_accounts.sh
go run cmd/main.go 1

# Run all journeys (1-19)
./scripts/run_all.sh
```

## Joining the Mainnet

Instructions for joining the Verana Mainnet will be provided prior to the network launch.

## Contributing

Contributing guidelines will be available in the repository once the project reaches its public development phase.

## License

This project is licensed under the Apache-2.0 License - see the LICENSE file for details.
