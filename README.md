# Verana Blockchain

[![Project Status: Active -- The project has reached a stable, usable state and is being actively developed.](https://img.shields.io/badge/repo%20status-Active-green.svg?style=flat-square)](https://www.repostatus.org/#active)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue?style=flat-square&logo=go)](https://pkg.go.dev/github.com/verana-labs/verana)
[![Go Report Card](https://goreportcard.com/badge/github.com/verana-labs/verana?style=flat-square)](https://goreportcard.com/report/github.com/verana-labs/verana)
[![Version](https://img.shields.io/github/tag/verana-labs/verana.svg?style=flat-square)](https://github.com/verana-labs/verana/releases/latest)
[![License: Apache-2.0](https://img.shields.io/github/license/verana-labs/verana.svg?style=flat-square)](https://github.com/verana-labs/verana/blob/main/LICENSE)
[![Lines Of Code](https://img.shields.io/tokei/lines/github/verana-labs/verana?style=flat-square)](https://github.com/verana-labs/verana)
[![Discord](https://badgen.net/badge/icon/discord?icon=discord&label)](https://discord.gg/verana)

Verana Blockchain is a Verifiable Public Registry (VPR) providing foundational infrastructure for decentralized trust ecosystems. As a sovereign Layer 1 appchain built on the Cosmos SDK, Verana enables trustless verification of credentials and services across ecosystems through a standardized trust registry framework.

Verana serves as a public registry of registries, allowing organizations to create and manage their own trust registries with defined credential schemas, roles for issuers, verifiers, and grantors, and custom business models. The platform facilitates trust resolution, enabling applications to validate roles and permissions in real time through a standardized API.

Key features include:

- **Trust Registry Management**: Create and control trust registries for different ecosystems
- **Credential Schema Management**: Define credential schemas with custom issuance and verification policies
- **Role-Based Permissions**: Grant and manage permissions for issuers, verifiers, and grantors
- **Tokenized Business Model**: Built-in economic incentives for trust ecosystem participants
- **DID Directory**: Public directory of service identifiers for better service discovery
- **Trust Resolution API**: Standard API supporting the Trust Registry Query Protocol (TRQP)

Verana is designed to bridge the gap between centralized trust models and the decentralized web, enabling trustworthy digital interactions across ecosystems while preserving privacy and sovereignty.

## System Requirements

These system specifications have been tested and are recommended for running a Verana node:

- Quad Core or larger AMD or Intel (amd64) CPU
- 32GB RAM
- 1TB SSD/NVMe Storage
- 50MBPS+ bidirectional internet connection

While Verana can run on lower-spec hardware, you may experience reduced performance or stability issues.

## Documentation

Documentation is currently under development and will be available soon.

## Joining the Mainnet

Instructions for joining the Verana Mainnet will be provided prior to the network launch.

## Development

### Protobuf Generation

After making changes to any `.proto` files, you need to regenerate the protobuf files and related code. Verana uses Cosmos SDK v0.50.13 and requires Ignite CLI v28.x for compatibility.

#### Prerequisites

First, install Ignite CLI v28.10.0 and rename it to `ignite_eden`:

1. **Download Ignite v28.10.0:**
   ```bash
   curl https://get.ignite.com/cli@v28.10.0 | bash
   ```

2. **Rename the binary:**
   ```bash
   mv /usr/local/bin/ignite /usr/local/bin/ignite_eden
   ```
   *Note: Adjust the path as needed for your environment (sometimes `~/.local/bin/ignite`).*

3. **Verify installation:**
   ```bash
   ignite_eden version
   ```
   You should see Ignite CLI version `v28.x.y` and Cosmos SDK v0.50.x.

#### Generating Protobuf Files

To regenerate protobuf files after making changes:

```bash
ignite_eden chain build
```

#### Generating OpenAPI Documentation

To generate OpenAPI documentation:

```bash
ignite_eden generate openapi --clear-cache --enable-proto-vendor
```

**Important:** Always run these commands after modifying any `.proto` files to ensure your changes are properly compiled and integrated into the codebase.

## Local Development Environment

A local development environment for Verana is planned and will be made available in the future.

## Contributing

Contributing guidelines will be available in the repository once the project reaches its public development phase.