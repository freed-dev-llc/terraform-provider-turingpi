# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.3.10] - 2026-01-25

### Fixed
- **BMC Firmware 2.0.5+ Compatibility**: Fixed flash status parsing for new `Transferring` response format (fixes #23)
  - BMC now returns an object with `id`, `process_name`, `size`, `cancelled`, `bytes_written` instead of array
  - Provider now handles both legacy array format and new object format
  - Added progress percentage display during transfer phase

## [1.3.9] - 2026-01-19

### Changed
- Synchronized release with terraform-turingpi-modules v1.3.9
- Documentation updates

### Verified
- All data sources tested on physical cluster (BMC firmware v2.3.4)
  - turingpi_info: Firmware version, network interfaces, storage
  - turingpi_about: API v1.1, daemon v2.3.4, Buildroot 2024.05.1
  - turingpi_power: All 4 nodes power status verified
  - turingpi_usb: Mode, node, route attributes
  - turingpi_sdcard: Storage capacity readings
  - turingpi_uart: UART configuration

## [1.3.8] - 2026-01-19

### Changed
- Synchronized release with terraform-turingpi-modules v1.3.8
- Documentation updates and cross-references to modules helper scripts

## [1.3.7] - 2026-01-19

### Changed
- Synchronized release with terraform-turingpi-modules v1.3.7

## [1.3.6] - 2026-01-18

### Fixed
- **BMC 2.0.x - 2.1.0 Compatibility**: Fixed `turingpi_usb` data source to handle "UsbA" route value returned by BMC firmware 2.0.x through 2.1.0 (fixes #21)
  - Added "UsbA" to recognized USB-A route values
  - Added test coverage for BMC 2.0.x response format

## [1.3.5] - 2026-01-18

### Added
- **Cluster Import Support**: `turingpi_k3s_cluster` resource now supports importing existing clusters
  - Import format: `cluster_name:control_plane_host:ssh_user:ssh_key_path`
  - Example: `terraform import turingpi_k3s_cluster.mycluster "mycluster:10.10.88.73:root:/home/user/.ssh/id_ed25519"`
- **Progress Feedback**: Added tflog structured logging throughout cluster creation
  - Logs cluster creation progress, worker installation status, addon deployment
  - Helps diagnose issues during long-running operations
- **K8s Client**: New `k8s_client.go` for applying Kubernetes manifests

### Fixed
- **MetalLB CRD Creation**: `deployMetalLB` now actually creates IPAddressPool and L2Advertisement CRDs
  - Previously only installed Helm chart without configuring IP allocation
  - Added `waitForMetalLBReady` to ensure CRDs are available before configuration
  - Added `applyMetalLBConfig` to create required Custom Resources

### Changed
- Improved error messages with more context for debugging
- Added context cancellation checks in long-running operations

## [1.3.4] - 2026-01-18

### Fixed
- **BMC Firmware 2.3.4 Compatibility**: Updated all data sources to support both legacy and new API response formats
  - `turingpi_info`: Fixed JSON parsing for network interfaces, storage devices, and version info
  - `turingpi_about`: Fixed JSON parsing for version information
  - `turingpi_usb`: Fixed JSON parsing for USB routing status, including new "Node X" format
  - All data sources now use `json.RawMessage` for flexible response handling

### Changed
- Updated unit tests to work with new response parsing approach

## [1.3.3] - 2026-01-18

### Changed
- Synchronized release with terraform-turingpi-modules v1.3.3

## [1.3.2] - 2025-01-18

### Changed
- **Dependencies**
  - Updated all Go modules to latest versions
  - Updated golang.org/x/crypto to v0.47.0
  - Updated Helm to v3.19.5
  - Updated Kubernetes client libraries to v0.35.0
  - Updated gRPC to v1.78.0

- **CI/CD**
  - Bumped github/codeql-action to 4.31.10
  - Bumped aquasecurity/trivy-action to 0.33.1
  - Bumped actions/checkout to 6.0.1
  - Bumped terraform-linters/setup-tflint to 6
  - Bumped codecov/codecov-action to 5.5.2

### Added
- Pull request template with Terraform provider-specific checklist
- Enhanced .gitignore with security patterns for keys, kubeconfig, talosconfig

## [1.3.1] - 2025-12-30

### Added
- **CI/CD Enhancements**
  - TFLint workflow for validating Terraform example files
  - terraform-docs workflow for auto-generating example documentation
  - Trivy vulnerability and license scanning in security workflow
  - Codecov integration for test coverage tracking
  - Pre-commit hooks configuration (`.pre-commit-config.yaml`)
    - Go: gofmt, go vet, go-mod-tidy, golangci-lint
    - Terraform: terraform_fmt, terraform_tflint, terraform_docs
    - General: trailing whitespace, end-of-file, YAML/JSON validation
  - golangci-lint v2 configuration (`.golangci.yml`)
  - TFLint configuration (`.tflint.hcl`)
  - terraform-docs configuration (`.terraform-docs.yml`)

- **Documentation**
  - Branch protection guidance in CONTRIBUTING.md
  - Pre-commit setup instructions in CONTRIBUTING.md
  - Auto-generated README.md for all examples

### Changed
- Updated GitHub Actions to latest versions (checkout v6, upload-artifact v6)
- Updated Helm dependency to v3.19.4
- Fixed `state` attribute in basic example (bool `true` → string `"on"`)
- Added Terraform files to `.gitignore` (`.terraform/`, `*.tfstate`)

## [1.3.0] - 2025-12-30

### Added
- **BMC API Compatibility** - Support for both legacy and new BMC firmware response formats
  - Power status now handles legacy `[[nodeName, status], ...]` and new `[{"result": [...]}]` formats
  - Added `parsePowerValue()` helper for flexible type conversion (bool, int, float, string)

- **Flash Resource Implementation** - Complete rewrite of `turingpi_flash` resource
  - Actual flash functionality via BMC API (previously stub)
  - Automatic node power-off before flashing
  - Streaming multipart firmware upload
  - Real-time flash progress monitoring with status updates
  - 25-minute timeout with configurable status polling

### Changed
- `data_source_power.go` - Uses `json.RawMessage` for flexible API response parsing
- `resource_flash.go` - Full implementation replacing placeholder code

## [1.2.2] - 2025-12-29

### Changed
- Updated provider version in README and docs to `>= 1.2.0`
- Added k3s-cluster, longhorn, monitoring, portainer to modules list in documentation
- Updated all examples to use `>= 1.2.0` version constraint
- Synchronized documentation with terraform-turingpi-modules repo

## [1.2.1] - 2025-12-29

### Added
- Related modules section in provider documentation (`docs/index.md`)
- Cross-references between provider and modules on Terraform Registry
- GitHub topics for discoverability (terraform, turingpi, kubernetes, talos, homelab)

### Changed
- Updated GitHub repo descriptions to link provider and modules
- Updated provider version references in documentation to v1.2.0

## [1.2.0] - 2025-12-29

### Deprecated
- `turingpi_k3s_cluster` resource - Will be removed in v2.0.0
  - Migrate to [terraform-turingpi-modules](https://registry.terraform.io/modules/freed-dev-llc/modules/turingpi)
- `turingpi_talos_cluster` resource - Will be removed in v2.0.0
  - Migrate to [terraform-turingpi-modules](https://registry.terraform.io/modules/freed-dev-llc/modules/turingpi)

### Added
- **New pkg/ Subpackages** - Extracted reusable provisioner code
  - `pkg/ssh/` - SSH client interface and helpers
  - `pkg/helm/` - Helm client interface for chart deployment
  - `pkg/kubeconfig/` - Kubeconfig utilities
  - `pkg/k3s/` - K3s cluster provisioner
  - `pkg/talos/` - Talos cluster provisioner

- **Terraform Modules** - Published to [Terraform Registry](https://registry.terraform.io/modules/freed-dev-llc/modules/turingpi)
  - `modules/flash-nodes` - Flash firmware to Turing Pi nodes
  - `modules/talos-cluster` - Deploy Talos Kubernetes cluster (native Talos provider)
  - `modules/addons/metallb` - MetalLB load balancer
  - `modules/addons/ingress-nginx` - NGINX Ingress controller

- **Documentation**
  - `docs/MIGRATION.md` - Migration guide from deprecated resources to modules
  - Updated README with modules reference

### Changed
- Cluster resources now use extracted pkg/ subpackages internally
- Deprecation warnings displayed when using cluster resources

## [1.1.4] - 2025-12-29

### Added
- **New Resource**
  - `turingpi_talos_cluster` - Deploy Talos Kubernetes clusters on pre-flashed Turing Pi nodes
    - Control plane and worker node configuration with custom hostnames
    - Bootstrap safety (prevents re-bootstrap by checking etcd status)
    - MetalLB load balancer deployment with configurable IP range
    - NGINX Ingress controller deployment
    - Kubeconfig and talosconfig output to files and Terraform state
    - Cluster secrets (PKI) stored in state for recovery
    - Cluster reset on destroy

- **Infrastructure**
  - `provider/talos_provisioner.go` - Talos provisioning via talosctl CLI
    - Interface-based exec.Command for testable design
    - Secrets generation, config generation, patching
    - Apply config, bootstrap, health checks, reset

- **Documentation**
  - `docs/resources/talos_cluster.md` - Talos cluster resource documentation
  - `examples/talos-cluster/` - Example configuration with MetalLB and Ingress

### Changed
- Updated all documentation and examples to reference v1.1.4

### Note
- Requires `talosctl` binary installed on machine running Terraform
- Talos uses mainline kernel (no Rockchip NPU driver support)

## [1.1.3] - 2025-12-29

### Added
- **New Resource**
  - `turingpi_k3s_cluster` - Deploy K3s Kubernetes clusters on pre-flashed Turing Pi nodes
    - K3s server installation on control plane via SSH
    - K3s agent installation on worker nodes
    - MetalLB load balancer deployment with configurable IP range
    - NGINX Ingress controller deployment
    - Kubeconfig output to file and Terraform state
    - Cluster token generation and management
    - Cluster uninstall on destroy

- **Infrastructure (v1.1.2)**
  - `provider/ssh_client.go` - SSH client interface with key-based and password authentication
  - `provider/cluster_helpers.go` - WaitForSSH, RunSSHCommand utilities
  - `provider/kubeconfig.go` - LoadKubeconfig, WaitForKubeAPI, ExtractClusterEndpoint
  - `provider/helm_client.go` - Helm client using mittwald/go-helm-client for addon deployment
  - `provider/k3s_provisioner.go` - K3s installation logic via SSH

- **Documentation**
  - `docs/resources/k3s_cluster.md` - K3s cluster resource documentation
  - `examples/k3s-cluster/` - Example configuration with MetalLB and Ingress

### Security
- Updated `github.com/containerd/containerd` from v1.7.28 to v1.7.29
  - Fixed local privilege escalation via wide permissions on CRI directory (high)
  - Fixed host memory exhaustion through Attach goroutine leak (medium)

## [1.1.1] - 2025-12-29

### Added
- **New Resources**
  - `turingpi_usb_boot` - Enable USB boot mode for nodes (pulls nRPIBOOT pin low for CM4s)
  - `turingpi_node_to_msd` - Reboot node into USB Mass Storage Device mode
  - `turingpi_clear_usb_boot` - Clear USB boot status for nodes
  - `turingpi_bmc_reload` - Restart BMC daemon (bmcd) with readiness monitoring

- **New Data Sources**
  - `turingpi_sdcard` - MicroSD card info (total/used/free bytes, GB values, usage percentage)
  - `turingpi_about` - BMC version info (API, daemon, buildroot, firmware, build time)

- **Documentation**
  - `docs/FUTURE_MODULES.md` - Comprehensive roadmap for K3s and Talos cluster modules
  - `TODO.md` - Implementation milestones (v1.1.2 - v1.1.5)

### Planned
- v1.1.4: `turingpi_talos_cluster` resource with Talos Image Factory integration
- v1.1.5: K3s enhancements (NPU support, Longhorn storage, cluster upgrades)

## [1.1.0] - 2025-12-29

### Added
- **New Data Sources**
  - `turingpi_info` - BMC version, network interfaces, storage devices, and node power status
  - `turingpi_power` - Current power status of all nodes with aggregated counts
  - `turingpi_usb` - Current USB routing configuration
  - `turingpi_uart` - Read buffered UART output from nodes (clears buffer on read)

- **New Resources**
  - `turingpi_usb` - Configure USB routing between nodes and USB-A/BMC
  - `turingpi_network_reset` - Trigger network switch reset
  - `turingpi_bmc_firmware` - Upgrade BMC firmware (upload or local file)
  - `turingpi_uart` - Write commands to node UART (serial console)
  - `turingpi_bmc_reboot` - Trigger BMC reboot with readiness monitoring

- **Enhanced Resources**
  - `turingpi_power` - Added `reset` state for node reboot, added `current_state` computed attribute

### Changed
- All new resources use Context-aware CRUD functions (CreateContext, etc.)
- Added input validation (ValidateDiagFunc) to all new resources
- Comprehensive unit tests for all new resources and data sources

## [1.0.10] - 2025-12-24

### Fixed
- Fix golangci-lint v2.7.2 errcheck violations (resp.Body.Close, os.Setenv/Unsetenv)

## [1.0.9] - 2025-12-24

### Added
- `make release VERSION=x.y.z` - automated release workflow
- `make release-prep VERSION=x.y.z` - update version in docs/examples only

### Changed
- Update all documentation and examples to v1.0.9

## [1.0.8] - 2025-12-24

### Changed
- Add security workflow badge to README
- Update all documentation examples to v1.0.7
- Update examples/basic, examples/flash-firmware, examples/full-provisioning to v1.0.7

## [1.0.7] - 2025-12-24

### Changed
- Bump `terraform-plugin-sdk/v2` from 2.35.0 to 2.38.1
- Bump `actions/checkout` from 4.3.1 to 6.0.1
- Bump `actions/setup-go` from 5.6.0 to 6.1.0
- Bump `golangci/golangci-lint-action` from 6.5.2 to 9.2.0
- Bump `github/codeql-action` from 3.28.0 to 4.31.9
- Bump `actions/dependency-review-action` from 4.5.0 to 4.8.2

## [1.0.6] - 2025-12-24

### Security
- Pin all GitHub Actions to SHA commits (supply chain protection)
- Add Dependabot for automated security updates (Go modules + Actions)
- Add gosec security scanner with SARIF reporting
- Add dependency-review-action for PR vulnerability scanning
- Enable branch protection (signed commits, required reviews, status checks)

### Added
- `.github/CODEOWNERS` for mandatory code review
- `.github/dependabot.yml` for automated dependency updates
- `.github/workflows/security.yml` for security scanning
- GPG signature verification documentation in SECURITY.md

### Changed
- Workflows now use `go-version-file: go.mod` instead of hardcoded version
- Enhanced SECURITY.md with release verification instructions

## [1.0.5] - 2025-12-22

### Added
- `boot_check_pattern` option for turingpi_node resource
- Configurable pattern matching for boot verification (default: `"login:"`)
- Support for Talos Linux boot detection (`"machine is running and ready"`)

## [1.0.4] - 2025-12-22

### Added
- `insecure` provider option to skip TLS certificate verification
- Useful for self-signed or expired BMC certificates
- Environment variable support via `TURINGPI_INSECURE`

### Changed
- Shared HTTP client for all API requests with configurable TLS settings

## [1.0.3] - 2025-12-22

### Added
- Terraform Registry documentation (docs/index.md, docs/resources/)
- Provider overview and authentication docs
- Resource documentation for turingpi_power, turingpi_flash, turingpi_node

## [1.0.2] - 2025-12-22

### Changed
- Provider source updated to `freed-dev-llc/turingpi` for Terraform Registry
- Simplified installation instructions (auto-download from registry)
- Consolidated GoReleaser config with GPG signing

### Added
- Terraform Registry badge to README
- Published to public Terraform Registry

## [1.0.1] - 2025-12-22

### Added
- Example Terraform configurations (basic, flash-firmware, full-provisioning)
- Build, release, and license badges to README
- Terraform Registry manifest for registry publishing
- GPG signing support for releases

### Fixed
- All golangci-lint issues (unchecked errors, deprecated APIs)
- GoReleaser config for unsigned releases

## [1.0.0] - 2025-12-22

### Added
- Initial release
- `turingpi_power` resource for node power control
- `turingpi_flash` resource for firmware flashing
- `turingpi_node` resource for comprehensive node management
- BMC authentication with username/password
- Environment variable support for credentials
- Configurable endpoint URL
- UART monitoring for boot verification
- Comprehensive test suite
- CONTRIBUTING.md with contribution guidelines
- SECURITY.md with security policy
- CODE_OF_CONDUCT.md (Contributor Covenant)
- GitHub issue and PR templates
- Makefile for build automation
- Release automation workflow with GoReleaser
- Multi-platform binaries (linux/darwin/windows, amd64/arm64)

[Unreleased]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.3.10...HEAD
[1.3.10]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.3.9...v1.3.10
[1.3.9]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.3.8...v1.3.9
[1.3.8]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.3.7...v1.3.8
[1.3.7]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.3.6...v1.3.7
[1.3.6]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.3.5...v1.3.6
[1.3.5]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.3.4...v1.3.5
[1.3.4]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.3.3...v1.3.4
[1.3.3]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.3.2...v1.3.3
[1.3.2]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.3.1...v1.3.2
[1.3.1]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.2.2...v1.3.0
[1.2.2]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.2.1...v1.2.2
[1.2.1]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.1.4...v1.2.0
[1.1.4]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.1.3...v1.1.4
[1.1.3]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.1.1...v1.1.3
[1.1.1]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.0.10...v1.1.0
[1.0.10]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.0.9...v1.0.10
[1.0.9]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.0.8...v1.0.9
[1.0.8]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.0.7...v1.0.8
[1.0.7]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.0.6...v1.0.7
[1.0.6]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.0.5...v1.0.6
[1.0.5]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.0.4...v1.0.5
[1.0.4]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.0.3...v1.0.4
[1.0.3]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/freed-dev-llc/terraform-provider-turingpi/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/freed-dev-llc/terraform-provider-turingpi/releases/tag/v1.0.0
