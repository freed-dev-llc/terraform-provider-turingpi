# TODO: Future Implementation Tasks

This file tracks planned features and implementation tasks for the Terraform Turing Pi provider.

## Current Release: v1.5.0 (2026-05-18)

### Recently Completed (v1.4.1 → v1.5.0)

- [x] **Added** `firmware_url` argument to `turingpi_flash` — BMC-pull path (`opt=set&type=flash&file=<URL>`) that actually writes eMMC, vs. the deprecated `firmware_file` streaming upload that returns `Done` in ~90 s without flashing. Issues #63 / #66 closed `not planned`.
- [x] **Gosec triage cycle** — dead `pkg/` tree removed, file permission tightening, 23 → 0 open code-scanning alerts.

### Recently Completed (v1.4.0 → v1.4.1)

- [x] **Fixed** `turingpi_flash` upload failure on older BMC firmware (#46): send multipart upload with explicit `Content-Length` instead of `Transfer-Encoding: chunked`
- [x] **Refactored** provider/resource handlers from deprecated `terraform-plugin-sdk/v2` callbacks (`ConfigureFunc`, `Create`/`Read`/`Update`/`Delete`) to `*Context` variants — resolves all staticcheck SA1019 findings
- [x] **Security** — bump `helm.sh/helm/v3` 3.20.0 → 3.20.2 (CVE-2026-35206), plus `golang.org/x/crypto`, `grpc` (transitive), `terraform-plugin-sdk/v2` 2.38.2 → 2.40.1
- [x] **CI hardening** for the containerized self-hosted runner:
  - gosec: replaced `securego/gosec` Docker action with `go install` + binary (pinned v2.22.11 to avoid v2.23.0's invalid SARIF)
  - terraform-docs: replaced `terraform-docs/gh-actions@v1` Docker action with `go install` + binary (v0.20.0)
  - talos-image workflow restored after ~2.5 months of latent breakage: rclone-based R2 reads bypass Cloudflare bot challenge; rclone upgraded from apt v1.60.1 to upstream v1.74.1+; R2 token rotated to bucket-scoped credentials; `no_check_bucket = true` added for bucket-scoped tokens
- [x] **Added** `cli-smoketest` matrix job: builds the dev provider and runs `validate` under both Terraform 1.15.3 and OpenTofu 1.12.0 via dev_overrides — catches plugin-protocol / manifest / schema regressions in either tool at PR time
- [x] **OpenTofu Registry** submission in progress (#45): signing key registered under `freed-dev-llc` namespace; provider submission auto-validated and awaiting maintainer merge (opentofu/registry PR #4261)

---

## Milestone: v1.6.0 - Polish & Stability

Items carried over from the original v1.4.0 plan (still pending after v1.5.0 shipped firmware_url + gosec triage).

### Testing Infrastructure
- [ ] Add mock Kubernetes API for testing
- [ ] Create cluster integration test framework
- [ ] Add acceptance tests for all resources (`TF_ACC=1`) — requires hardware-attached CI runner
- [ ] Improve test coverage (target: 80%+)

### Documentation
- [ ] Add K3s deployment guide
- [ ] Add troubleshooting guide
- [ ] Add best practices guide
- [ ] Create video tutorials
- [ ] Add architecture diagrams

### Bug Fixes & Improvements
- [ ] Handle partial failures gracefully in cluster resources
- [ ] Improve error messages for common issues (partial: #46 fixed the most misleading one, `io: read/write on closed pipe`)
- [ ] Add retry logic for transient BMC failures

---

## Milestone: v1.6.0 - Advanced Features

### Cluster Operations
- [ ] Implement cluster upgrade support (K3s version bumps)
- [ ] Implement Talos upgrade support
- [ ] Add node add/remove operations
- [ ] Add cluster backup functionality
- [ ] Add cluster restore functionality

### Storage Setup
- [ ] Detect NVMe devices on nodes
- [ ] Partition and format NVMe for Longhorn
- [ ] Create mount points and symlinks
- [ ] Configure iSCSI for Longhorn
- [ ] Deploy Longhorn with NVMe storage class

---

## Milestone: v1.7.0 - Multi-Cluster & Observability

### Multi-Cluster Support
- [ ] Support managing multiple clusters
- [ ] Add cluster federation options
- [ ] Cross-cluster service mesh (optional)

### Observability
- [ ] Add cluster health data source
- [ ] Add node metrics data source
- [ ] Add addon status data source
- [ ] Grafana dashboard templates

---

## Milestone: v2.0.0 - NPU Support

### NPU Support (RK3588) - Pending Kernel Support
- [ ] Detect vendor kernel (6.1.x) for NPU compatibility
- [ ] Install RKNN-Toolkit2 runtime
- [ ] Install RKNN-LLM library
- [ ] Deploy rkllama service
- [ ] Download and configure AI models (DeepSeek, etc.)
- [ ] Verify NPU functionality (`/sys/kernel/debug/rknpu/version`)

---

## Research Tasks

### Ansible Integration Options
- [ ] Evaluate `hashicorp/terraform-provider-ansible`
- [ ] Evaluate embedded Ansible runner in Go
- [ ] Evaluate pure SSH-based provisioning
- [ ] Document pros/cons of each approach

### NPU Support Timeline
- [ ] Monitor mainline kernel NPU driver progress
- [ ] Track Rockchip open-source driver efforts
- [ ] Evaluate custom kernel builds for Talos
- [ ] Update documentation when status changes

---

## Completed Milestones

### v1.4.1 - Stability & Tooling Recovery ✅ (2026-05-17)
See "Recently Completed" above for the full list.

### v1.4.0 - Org Migration ✅ (2026-03-07)
- [x] Migrate organization from `jfreed-dev` to `freed-dev-llc`
- [x] Update all GitHub URLs, Terraform Registry sources, documentation
- [x] Update Go module path to `github.com/freed-dev-llc/turingpi-terraform-provider`
- [x] Include `manifest.json` in SHA256SUMS checksum (Terraform Registry compatibility)

### v1.3.x - CI/CD & Maintenance ✅
- [x] TFLint workflow for validating Terraform examples
- [x] terraform-docs workflow for auto-generating documentation
- [x] Trivy vulnerability and license scanning
- [x] Codecov integration for test coverage
- [x] Pre-commit hooks configuration
- [x] golangci-lint v2 configuration
- [x] PR template and .gitignore security enhancements
- [x] Apache 2.0 license

### v1.2.x - BMC API Compatibility ✅
- [x] Support for legacy and new BMC firmware response formats
- [x] Flash resource implementation with progress monitoring
- [x] BMC firmware upgrade with file upload support

### v1.1.x - Cluster Support ✅
- [x] K3s cluster resource with SSH-based provisioning
- [x] Talos cluster resource with talosctl integration
- [x] MetalLB and NGINX Ingress addon deployment
- [x] Helm integration via mittwald/go-helm-client
- [x] SSH client with key-based auth
- [x] Kubeconfig parsing and validation

### v1.0.x - Foundation ✅
- [x] Power management for nodes 1-4
- [x] UART read/write access
- [x] USB routing configuration
- [x] USB boot mode for CM4
- [x] Network reset resource
- [x] SD card storage monitoring
- [x] TLS flexibility for self-signed certs

---

## Items Completed Out-of-Provider

These items appeared in earlier roadmaps under provider milestones but were actually delivered in the sister `freed-dev-llc/terraform-turingpi-modules` repo. Listed here so the provider TODO doesn't show them as outstanding.

### Talos Image Factory Integration → `terraform-turingpi-modules/modules/talos-image`
The original plan was to build `provider/talos_factory_client.go` in this repo. Instead, the integration shipped as a pure-Terraform module that calls `factory.talos.dev` via the `http` provider. No Go client needed.

### Addon Deployments → `terraform-turingpi-modules/modules/addons/*`
The original v1.5.0 "Additional Addons" milestone listed Prometheus stack, Portainer agent, and Ingress resources. All shipped as Terraform modules:
- `modules/addons/monitoring` (Prometheus + Grafana, kube-prometheus-stack)
- `modules/addons/portainer`
- `modules/addons/ingress-nginx`
- Plus: `modules/addons/metallb`, `modules/addons/longhorn`, `modules/addons/cert-manager`

---

## Notes

### Key Differences: K3s vs Talos

| Feature | K3s (Armbian) | Talos |
|---------|---------------|-------|
| Base OS | Armbian (Debian) | Talos Linux |
| Management | SSH + Ansible | talosctl API |
| Mutability | Mutable (install packages) | Immutable |
| NPU Support | ✅ Yes (vendor kernel) | ❌ No (mainline kernel) |
| GPU Support | Limited | ❌ No |
| Complexity | Higher (more config) | Lower (opinionated) |
| Recovery | SSH access | API only |

### Network Configuration (Default)

| Component | IP/Range |
|-----------|----------|
| BMC | 10.10.88.70 |
| Control Plane | 10.10.88.73 |
| Workers | 10.10.88.74-76 |
| MetalLB Pool | 10.10.88.80-89 |
| Pod Network | 10.244.0.0/16 |
| Service Network | 10.96.0.0/12 |

### Hardware Reference
- Board: Turing Pi 2.5
- Compute: 4x RK1 (RK3588 SoC)
- CPU: 8-core ARM64 (4x A76 + 4x A55) per node
- RAM: 16-32GB per node
- Storage: 32GB eMMC + 500GB NVMe per worker
- NPU: 6 TOPS per node (K3s only)
