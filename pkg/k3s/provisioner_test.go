package k3s

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/freed-dev-llc/turingpi-terraform-provider/pkg/ssh"
)

// MockSSHClient implements ssh.Client for testing
type MockSSHClient struct {
	ConnectFunc    func(host string, port int, config *ssh.Config) error
	RunCommandFunc func(cmd string) (string, error)
	CloseFunc      func() error

	// Track calls for verification
	ConnectCalls []struct {
		Host string
		Port int
	}
	RunCommandCalls []string
	CloseCalls      int
}

func (m *MockSSHClient) Connect(host string, port int, config *ssh.Config) error {
	m.ConnectCalls = append(m.ConnectCalls, struct {
		Host string
		Port int
	}{host, port})
	if m.ConnectFunc != nil {
		return m.ConnectFunc(host, port, config)
	}
	return nil
}

func (m *MockSSHClient) RunCommand(cmd string) (string, error) {
	m.RunCommandCalls = append(m.RunCommandCalls, cmd)
	if m.RunCommandFunc != nil {
		return m.RunCommandFunc(cmd)
	}
	return "", nil
}

func (m *MockSSHClient) Close() error {
	m.CloseCalls++
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// Test that Provisioner can be created
func TestNewProvisioner(t *testing.T) {
	p := NewProvisioner()
	if p == nil {
		t.Fatal("expected non-nil provisioner")
	}
	if p.clientFactory == nil {
		t.Fatal("expected non-nil client factory")
	}
}

// Test that GenerateClusterToken produces valid tokens
func TestGenerateClusterToken(t *testing.T) {
	token1 := GenerateClusterToken()
	token2 := GenerateClusterToken()

	if token1 == "" {
		t.Error("expected non-empty token")
	}
	if len(token1) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("expected 64 char token, got %d chars", len(token1))
	}
	if token1 == token2 {
		t.Error("expected unique tokens")
	}
}

// Test NodeConfig.getSSHConfig
func TestNodeConfig_getSSHConfig(t *testing.T) {
	node := NodeConfig{
		Host:        "192.168.1.100",
		SSHUser:     "root",
		SSHKey:      []byte("test-key"),
		SSHPassword: "test-pass",
		SSHPort:     22,
	}

	config := node.getSSHConfig()
	if config.User != "root" {
		t.Errorf("expected user 'root', got %q", config.User)
	}
	if string(config.PrivateKey) != "test-key" {
		t.Errorf("expected PrivateKey 'test-key', got %q", string(config.PrivateKey))
	}
	if config.Password != "test-pass" {
		t.Errorf("expected Password 'test-pass', got %q", config.Password)
	}
	if config.Timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", config.Timeout)
	}
}

// Test InstallServer with already installed K3s
func TestProvisioner_InstallServer_AlreadyInstalled(t *testing.T) {
	mock := &MockSSHClient{
		RunCommandFunc: func(cmd string) (string, error) {
			if strings.Contains(cmd, "test -f /usr/local/bin/k3s") {
				return "installed", nil
			}
			if strings.Contains(cmd, "systemctl start k3s") {
				return "", nil
			}
			if strings.Contains(cmd, "k3s kubectl get nodes") {
				return "node1   Ready", nil
			}
			return "", nil
		},
	}

	p := NewProvisionerWithClientFactory(func() ssh.Client { return mock })

	node := NodeConfig{Host: "192.168.1.100", SSHUser: "root", SSHPort: 22}
	cfg := ClusterConfig{Name: "test-cluster"}

	err := p.InstallServer(context.Background(), node, cfg, 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test InstallServer failure - swap disable fails
func TestProvisioner_InstallServer_SwapDisableFails(t *testing.T) {
	mock := &MockSSHClient{
		RunCommandFunc: func(cmd string) (string, error) {
			if strings.Contains(cmd, "swapoff") {
				return "", errors.New("permission denied")
			}
			return "", nil
		},
	}

	p := NewProvisionerWithClientFactory(func() ssh.Client { return mock })

	node := NodeConfig{Host: "192.168.1.100", SSHUser: "root", SSHPort: 22}
	cfg := ClusterConfig{Name: "test-cluster"}

	err := p.InstallServer(context.Background(), node, cfg, 30*time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to disable swap") {
		t.Errorf("expected swap error, got: %v", err)
	}
}

// Test GetNodeToken
func TestProvisioner_GetNodeToken(t *testing.T) {
	expectedToken := "K10abc123::server:xyz789"
	mock := &MockSSHClient{
		RunCommandFunc: func(cmd string) (string, error) {
			if strings.Contains(cmd, "cat /var/lib/rancher/k3s/server/node-token") {
				return expectedToken + "\n", nil
			}
			return "", nil
		},
	}

	p := NewProvisionerWithClientFactory(func() ssh.Client { return mock })
	node := NodeConfig{Host: "192.168.1.100", SSHUser: "root", SSHPort: 22}

	token, err := p.GetNodeToken(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != expectedToken {
		t.Errorf("expected token %q, got %q", expectedToken, token)
	}
}

// Test GetKubeconfig
func TestProvisioner_GetKubeconfig(t *testing.T) {
	mock := &MockSSHClient{
		RunCommandFunc: func(cmd string) (string, error) {
			if strings.Contains(cmd, "cat /etc/rancher/k3s/k3s.yaml") {
				return `apiVersion: v1
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: default
`, nil
			}
			return "", nil
		},
	}

	p := NewProvisionerWithClientFactory(func() ssh.Client { return mock })
	node := NodeConfig{Host: "192.168.1.100", SSHUser: "root", SSHPort: 22}

	kubeconfig, err := p.GetKubeconfig(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have replaced 127.0.0.1 with node host
	if strings.Contains(kubeconfig, "127.0.0.1") {
		t.Error("expected 127.0.0.1 to be replaced with node host")
	}
	if !strings.Contains(kubeconfig, "192.168.1.100") {
		t.Error("expected kubeconfig to contain node host")
	}
}

// Test UninstallServer - not installed
func TestProvisioner_UninstallServer_NotInstalled(t *testing.T) {
	mock := &MockSSHClient{
		RunCommandFunc: func(cmd string) (string, error) {
			if strings.Contains(cmd, "test -f /usr/local/bin/k3s-uninstall.sh") {
				return "not_exists", nil
			}
			return "", nil
		},
	}

	p := NewProvisionerWithClientFactory(func() ssh.Client { return mock })
	node := NodeConfig{Host: "192.168.1.100", SSHUser: "root", SSHPort: 22}

	err := p.UninstallServer(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test UninstallServer - installed
func TestProvisioner_UninstallServer_Installed(t *testing.T) {
	uninstallCalled := false
	mock := &MockSSHClient{
		RunCommandFunc: func(cmd string) (string, error) {
			if strings.Contains(cmd, "test -f /usr/local/bin/k3s-uninstall.sh") {
				return "exists", nil
			}
			if strings.Contains(cmd, "k3s-uninstall.sh") {
				uninstallCalled = true
				return "", nil
			}
			return "", nil
		},
	}

	p := NewProvisionerWithClientFactory(func() ssh.Client { return mock })
	node := NodeConfig{Host: "192.168.1.100", SSHUser: "root", SSHPort: 22}

	err := p.UninstallServer(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !uninstallCalled {
		t.Error("expected uninstall script to be called")
	}
}

// Test CheckInstalled
func TestProvisioner_CheckInstalled(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{"installed", "installed", true},
		{"not_installed", "not_installed", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockSSHClient{
				RunCommandFunc: func(cmd string) (string, error) {
					return tt.output, nil
				},
			}

			p := NewProvisionerWithClientFactory(func() ssh.Client { return mock })
			node := NodeConfig{Host: "192.168.1.100", SSHUser: "root", SSHPort: 22}

			installed, err := p.CheckInstalled(node)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if installed != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, installed)
			}
		})
	}
}

// Test GetVersion
func TestProvisioner_GetVersion(t *testing.T) {
	mock := &MockSSHClient{
		RunCommandFunc: func(cmd string) (string, error) {
			if strings.Contains(cmd, "k3s --version") {
				return "k3s version v1.28.4+k3s1 (abc123)\n", nil
			}
			return "", nil
		},
	}

	p := NewProvisionerWithClientFactory(func() ssh.Client { return mock })
	node := NodeConfig{Host: "192.168.1.100", SSHUser: "root", SSHPort: 22}

	version, err := p.GetVersion(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "k3s version v1.28.4+k3s1 (abc123)" {
		t.Errorf("unexpected version: %q", version)
	}
}

// Test GetClusterNodes
func TestProvisioner_GetClusterNodes(t *testing.T) {
	mock := &MockSSHClient{
		RunCommandFunc: func(cmd string) (string, error) {
			if strings.Contains(cmd, "kubectl get nodes") {
				return "'node1 node2 node3'", nil
			}
			return "", nil
		},
	}

	p := NewProvisionerWithClientFactory(func() ssh.Client { return mock })
	node := NodeConfig{Host: "192.168.1.100", SSHUser: "root", SSHPort: 22}

	nodes, err := p.GetClusterNodes(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(nodes))
	}
}
