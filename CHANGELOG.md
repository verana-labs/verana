# Changelog

## [0.11.0](https://github.com/verana-labs/verana/compare/v0.10.0...v0.11.0) (2026-06-09)


### ⚠ BREAKING CHANGES

* **de,pp:** v4-rc2 VSOA rebase (#309-#313) ([#328](https://github.com/verana-labs/verana/issues/328))
* **co,de:** implement AUTHZ-CHECK-5 (Corporation Registration check) on delegable messages ([#327](https://github.com/verana-labs/verana/issues/327))
* **pp:** rename x/perm to x/pp (Permission → Participant) per spec v4-rc2 ([#326](https://github.com/verana-labs/verana/issues/326))
* **ec:** rename x/tr to x/ec (TrustRegistry → Ecosystem) per v4-rc2 spec ([#322](https://github.com/verana-labs/verana/issues/322))
* **gf:** implement MOD-GF Governance Framework module per spec v4 ([#318](https://github.com/verana-labs/verana/issues/318))

### Features

* align TR/CS/PERM/TD ledger surface with spec v4 draft 13 ([#280](https://github.com/verana-labs/verana/issues/280)) ([#281](https://github.com/verana-labs/verana/issues/281)) ([deffad8](https://github.com/verana-labs/verana/commit/deffad89dcf3009b95792c73623a9845c3f88715))
* **co,de:** implement AUTHZ-CHECK-5 (Corporation Registration check) on delegable messages ([#327](https://github.com/verana-labs/verana/issues/327)) ([786811d](https://github.com/verana-labs/verana/commit/786811de1eb99f4e306d32efeafc0aa33a58a125))
* **co:** implement MOD-CO Corporation module  ([#319](https://github.com/verana-labs/verana/issues/319)) ([5bf7e7d](https://github.com/verana-labs/verana/commit/5bf7e7d26ec8a18157c5fc3a541aa901db4c9578))
* **cs:** Implement Credential Schema module ([#258](https://github.com/verana-labs/verana/issues/258)) ([d8ac62d](https://github.com/verana-labs/verana/commit/d8ac62d1824997007317139de84bcc9d071e43a6))
* **de,pp:** v4-rc2 VSOA rebase ([#309](https://github.com/verana-labs/verana/issues/309)-[#313](https://github.com/verana-labs/verana/issues/313)) ([#328](https://github.com/verana-labs/verana/issues/328)) ([9123601](https://github.com/verana-labs/verana/commit/9123601d58040994ae360dcfde2846783b235e16))
* **gf:** implement MOD-GF Governance Framework module per spec v4 ([#318](https://github.com/verana-labs/verana/issues/318)) ([f9ae598](https://github.com/verana-labs/verana/commit/f9ae598faae4c426edc177e68e000b95b661f967))
* **perm:** Implement Start Permission VP [#207](https://github.com/verana-labs/verana/issues/207) ([#261](https://github.com/verana-labs/verana/issues/261)) ([1211f13](https://github.com/verana-labs/verana/commit/1211f13b029318865162ae09521bf3dcb1d47dfa))
* spec v4 mod de msg5 6 ([#275](https://github.com/verana-labs/verana/issues/275)) ([bafdb98](https://github.com/verana-labs/verana/commit/bafdb98d45df7ddd6676267596576332d514aba8))
* spec v4 mod td impl ([#272](https://github.com/verana-labs/verana/issues/272)) ([cad2179](https://github.com/verana-labs/verana/commit/cad2179855440d0a735ba4ac1b7c5d066fb06f61))
* **ts-proto:** port npm packaging and amino signing surface ([#278](https://github.com/verana-labs/verana/issues/278)) ([a8b2831](https://github.com/verana-labs/verana/commit/a8b2831de82793959891890ca87a0389150403b2))


### Bug Fixes

* **ci:** bump Node to 24 in buildBinaries workflow ([#282](https://github.com/verana-labs/verana/issues/282)) ([259ed8d](https://github.com/verana-labs/verana/commit/259ed8d836d0864c23be73748bbfead2547eeb7e))
* **ci:** login to Docker Hub before Buildx setup ([#276](https://github.com/verana-labs/verana/issues/276)) ([b5f15c0](https://github.com/verana-labs/verana/commit/b5f15c05b2caf7656a81c823c7467029cbe72394))
* **perm:** align CreateRootPermission and RenewPermissionVP with spec v4 draft 13 ([#285](https://github.com/verana-labs/verana/issues/285)) ([b8b3fd3](https://github.com/verana-labs/verana/commit/b8b3fd3833c21da4b78e0868c05b7471d7c5a32e))
* standardize CD with automated versioning via resolve-version ([#262](https://github.com/verana-labs/verana/issues/262)) ([b8cc665](https://github.com/verana-labs/verana/commit/b8cc6656ce977c6e872fc35f2e22a5ec2aa188ef))


### Code Refactoring

* **ec:** rename x/tr to x/ec (TrustRegistry → Ecosystem) per v4-rc2 spec ([#322](https://github.com/verana-labs/verana/issues/322)) ([1b328a1](https://github.com/verana-labs/verana/commit/1b328a1d852283fbd3222ec1f53ce9d566920ec6))
* **pp:** rename x/perm to x/pp (Permission → Participant) per spec v4-rc2 ([#326](https://github.com/verana-labs/verana/issues/326)) ([618561b](https://github.com/verana-labs/verana/commit/618561b5cd9701d17e7f036f7268947673f54ddf))
