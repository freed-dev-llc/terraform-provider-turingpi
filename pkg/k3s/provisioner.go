// Package k3s provides K3s cluster provisioning via SSH.
package k3s

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/freed-dev-llc/turingpi-terraform-provider/pkg/ssh"
)

// NodeConfig holds SSH connection details for a K3s node
type NodeConfig struct {
	Host        string
	SSHUser     string
	SSHKey      []byte
	SSHPassword string
	SSHPort     int
}

// ClusterConfig holds the K3s cluster configuration
type ClusterConfig struct {
	Name         string
	K3sVersion   string
	ClusterToken string
	PodCIDR      string
	ServiceCIDR  string
	ControlPlane NodeConfig
	Workers      []NodeConfig
}

// Provisioner handles K3s cluster installation via SSH
type Provisioner struct {
	clientFactory func() ssh.Client
}

// NewProvisioner creates a new K3s provisioner
func NewProvisioner() *Provisioner {
	return &Provisioner{
		clientFactory: ssh.NewClient,
	}
}

// NewProvisionerWithClientFactory creates a provisioner with custom client factory (for testing)
func NewProvisionerWithClientFactory(factory func() ssh.Client) *Provisioner {
	return &Provisioner{
		clientFactory: factory,
	}
}

// GenerateClusterToken generates a random cluster token
func GenerateClusterToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based token
		return fmt.Sprintf("k3s-token-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// getSSHConfig creates ssh.Config from NodeConfig
func (n *NodeConfig) getSSHConfig() *ssh.Config {
	return &ssh.Config{
		User:       n.SSHUser,
		PrivateKey: n.SSHKey,
		Password:   n.SSHPassword,
		Timeout:    30 * time.Second,
	}
}

// runCommand executes a command on a node via SSH
func (p *Provisioner) runCommand(node NodeConfig, cmd string) (string, error) {
	client := p.clientFactory()
	if err := client.Connect(node.Host, node.SSHPort, node.getSSHConfig()); err != nil {
		return "", fmt.Errorf("SSH connection failed: %w", err)
	}
	defer func() { _ = client.Close() }()

	output, err := client.RunCommand(cmd)
	if err != nil {
		return output, fmt.Errorf("command failed: %w", err)
	}
	return output, nil
}

// InstallServer installs K3s server on the control plane node
func (p *Provisioner) InstallServer(ctx context.Context, node NodeConfig, cfg ClusterConfig, timeout time.Duration) error {
	// 1. Disable swap
	if _, err := p.runCommand(node, "swapoff -a"); err != nil {
		return fmt.Errorf("failed to disable swap: %w", err)
	}

	// 2. Create K3s config directory
	if _, err := p.runCommand(node, "mkdir -p /etc/rancher/k3s"); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 3. Check if K3s is already installed
	output, _ := p.runCommand(node, "test -f /usr/local/bin/k3s && echo 'installed' || echo 'not_installed'")
	if strings.TrimSpace(output) == "installed" {
		// K3s already installed, just ensure it's running
		if _, err := p.runCommand(node, "systemctl start k3s"); err != nil {
			return fmt.Errorf("failed to start existing K3s: %w", err)
		}
		return p.waitForReady(node, timeout)
	}

	// 4. Download K3s install script
	downloadCmd := "curl -sfL https://get.k3s.io -o /tmp/k3s-install.sh && chmod +x /tmp/k3s-install.sh"
	if _, err := p.runCommand(node, downloadCmd); err != nil {
		return fmt.Errorf("failed to download K3s install script: %w", err)
	}

	// 5. Build install command with environment variables
	var envVars []string
	if cfg.K3sVersion != "" {
		envVars = append(envVars, fmt.Sprintf("INSTALL_K3S_VERSION=%s", cfg.K3sVersion))
	}
	if cfg.ClusterToken != "" {
		envVars = append(envVars, fmt.Sprintf("K3S_TOKEN=%s", cfg.ClusterToken))
	}

	installCmd := fmt.Sprintf("%s /tmp/k3s-install.sh server", strings.Join(envVars, " "))
	if _, err := p.runCommand(node, installCmd); err != nil {
		return fmt.Errorf("failed to install K3s server: %w", err)
	}

	// 6. Wait for K3s to be ready
	return p.waitForReady(node, timeout)
}

// waitForReady waits for K3s to be ready on the control plane
func (p *Provisioner) waitForReady(node NodeConfig, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		output, err := p.runCommand(node, "k3s kubectl get nodes 2>/dev/null")
		if err == nil && strings.Contains(output, "Ready") {
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for K3s to be ready after %v", timeout)
}

// GetNodeToken retrieves the node token from the control plane
func (p *Provisioner) GetNodeToken(node NodeConfig) (string, error) {
	output, err := p.runCommand(node, "cat /var/lib/rancher/k3s/server/node-token")
	if err != nil {
		return "", fmt.Errorf("failed to get node token: %w", err)
	}
	return strings.TrimSpace(output), nil
}

// GetKubeconfig retrieves and fixes the kubeconfig from the control plane
func (p *Provisioner) GetKubeconfig(node NodeConfig) (string, error) {
	output, err := p.runCommand(node, "cat /etc/rancher/k3s/k3s.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Replace 127.0.0.1 with the actual node IP
	kubeconfig := strings.ReplaceAll(output, "127.0.0.1", node.Host)
	kubeconfig = strings.ReplaceAll(kubeconfig, "localhost", node.Host)

	return kubeconfig, nil
}

// InstallAgent installs K3s agent on a worker node
func (p *Provisioner) InstallAgent(ctx context.Context, node NodeConfig, serverURL, nodeToken, k3sVersion string, timeout time.Duration) error {
	// 1. Disable swap
	if _, err := p.runCommand(node, "swapoff -a"); err != nil {
		return fmt.Errorf("failed to disable swap: %w", err)
	}

	// 2. Create K3s config directory
	if _, err := p.runCommand(node, "mkdir -p /etc/rancher/k3s"); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 3. Check if K3s agent is already installed
	output, _ := p.runCommand(node, "test -f /usr/local/bin/k3s && echo 'installed' || echo 'not_installed'")
	if strings.TrimSpace(output) == "installed" {
		// K3s already installed, just ensure it's running
		// Ignore error - might not be configured as agent yet
		_, _ = p.runCommand(node, "systemctl start k3s-agent")
		return nil
	}

	// 4. Download K3s install script
	downloadCmd := "curl -sfL https://get.k3s.io -o /tmp/k3s-install.sh && chmod +x /tmp/k3s-install.sh"
	if _, err := p.runCommand(node, downloadCmd); err != nil {
		return fmt.Errorf("failed to download K3s install script: %w", err)
	}

	// 5. Build install command with environment variables
	var envVars []string
	envVars = append(envVars, fmt.Sprintf("K3S_URL=%s", serverURL))
	envVars = append(envVars, fmt.Sprintf("K3S_TOKEN=%s", nodeToken))
	if k3sVersion != "" {
		envVars = append(envVars, fmt.Sprintf("INSTALL_K3S_VERSION=%s", k3sVersion))
	}

	installCmd := fmt.Sprintf("%s /tmp/k3s-install.sh agent", strings.Join(envVars, " "))
	if _, err := p.runCommand(node, installCmd); err != nil {
		return fmt.Errorf("failed to install K3s agent: %w", err)
	}

	return nil
}

// WaitForNodeReady waits for a specific node to be Ready in the cluster
func (p *Provisioner) WaitForNodeReady(controlPlane NodeConfig, nodeHost string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	// Extract hostname from the node - typically the last octet or full hostname
	// K3s uses the system hostname, so we need to check what hostname the node reports
	for time.Now().Before(deadline) {
		// Get all nodes and check if our node's IP appears and is Ready
		output, err := p.runCommand(controlPlane, "k3s kubectl get nodes -o wide 2>/dev/null")
		if err == nil {
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				if strings.Contains(line, nodeHost) && strings.Contains(line, "Ready") {
					return nil
				}
			}
		}
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for node %s to be Ready after %v", nodeHost, timeout)
}

// UninstallServer removes K3s server from a node
func (p *Provisioner) UninstallServer(node NodeConfig) error {
	// Check if uninstall script exists
	output, _ := p.runCommand(node, "test -f /usr/local/bin/k3s-uninstall.sh && echo 'exists' || echo 'not_exists'")
	if strings.TrimSpace(output) != "exists" {
		return nil // K3s not installed
	}

	if _, err := p.runCommand(node, "/usr/local/bin/k3s-uninstall.sh"); err != nil {
		return fmt.Errorf("failed to uninstall K3s server: %w", err)
	}
	return nil
}

// UninstallAgent removes K3s agent from a node
func (p *Provisioner) UninstallAgent(node NodeConfig) error {
	// Check if uninstall script exists
	output, _ := p.runCommand(node, "test -f /usr/local/bin/k3s-agent-uninstall.sh && echo 'exists' || echo 'not_exists'")
	if strings.TrimSpace(output) != "exists" {
		return nil // K3s agent not installed
	}

	if _, err := p.runCommand(node, "/usr/local/bin/k3s-agent-uninstall.sh"); err != nil {
		return fmt.Errorf("failed to uninstall K3s agent: %w", err)
	}
	return nil
}

// CheckInstalled checks if K3s is installed on a node
func (p *Provisioner) CheckInstalled(node NodeConfig) (bool, error) {
	output, _ := p.runCommand(node, "test -f /usr/local/bin/k3s && echo 'installed' || echo 'not_installed'")
	return strings.TrimSpace(output) == "installed", nil
}

// GetVersion returns the installed K3s version on a node
func (p *Provisioner) GetVersion(node NodeConfig) (string, error) {
	output, err := p.runCommand(node, "k3s --version 2>/dev/null | head -1")
	if err != nil {
		return "", fmt.Errorf("failed to get K3s version: %w", err)
	}
	return strings.TrimSpace(output), nil
}

// GetClusterNodes returns the list of nodes in the cluster
func (p *Provisioner) GetClusterNodes(controlPlane NodeConfig) ([]string, error) {
	output, err := p.runCommand(controlPlane, "k3s kubectl get nodes -o jsonpath='{.items[*].metadata.name}' 2>/dev/null")
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster nodes: %w", err)
	}

	nodes := strings.Fields(strings.Trim(output, "'"))
	return nodes, nil
}
