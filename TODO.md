# TODO: Future Implementation Tasks

This file tracks planned features and implementation tasks for the Terraform Turing Pi provider.

## Current Release: v1.4.0

### Recently Completed
- [x] Updated all Go modules to latest versions
- [x] Updated CI action versions (checkout v6, codecov v5.5.2, trivy v0.33.1)
- [x] Added PR template with Terraform provider checklist
- [x] Enhanced .gitignore with security patterns
- [x] Changed license from MIT to Apache 2.0
- [x] Added Go version and downloads badges

---

## Milestone: v1.4.0 - Polish & Stability

### Testing Infrastructure
- [ ] Add mock Kubernetes API for testing
- [ ] Create cluster integration test framework
- [ ] Add acceptance tests for all resources
- [ ] Improve test coverage (target: 80%+)

### Documentation
- [ ] Add K3s deployment guide
- [ ] Add troubleshooting guide
- [ ] Add best practices guide
- [ ] Create video tutorials
- [ ] Add architecture diagrams

### Bug Fixes & Improvements
- [ ] Handle partial failures gracefully in cluster resources
- [ ] Improve error messages for common issues
- [ ] Add retry logic for transient BMC failures

---

## Milestone: v1.5.0 - Advanced Features

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

### Additional Addons
- [ ] Deploy Prometheus stack (optional)
- [ ] Deploy Portainer agent (optional)
- [ ] Create Ingress resources for addon UIs

---

## Milestone: v1.6.0 - Multi-Cluster & Observability

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
- [ ] Verify NPU functionality (/sys/kernel/debug/rknpu/version)

### Talos Image Factory Integration
- [ ] Create `provider/talos_factory_client.go`
- [ ] Implement schematic submission to factory.talos.dev
- [ ] Parse schematic response for image URL
- [ ] Download and cache Talos images
- [ ] Support custom extensions (iscsi-tools, util-linux-tools)
- [ ] Support sbc-rockchip overlay

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
