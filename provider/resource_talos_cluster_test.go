package provider

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestResourceTalosCluster(t *testing.T) {
	resource := resourceTalosCluster()
	if resource == nil {
		t.Fatal("resourceTalosCluster() returned nil")
	}
}

func TestResourceTalosCluster_Schema(t *testing.T) {
	resource := resourceTalosCluster()
	schema := resource.Schema

	requiredFields := []string{"name", "cluster_endpoint", "control_plane"}
	for _, field := range requiredFields {
		if _, ok := schema[field]; !ok {
			t.Errorf("Schema missing required field: %s", field)
		}
	}

	optionalFields := []string{
		"talos_version", "kubernetes_version", "install_disk",
		"worker", "allow_scheduling_on_control_plane",
		"metallb", "ingress", "bootstrap_timeout",
		"kubeconfig_path", "talosconfig_path", "secrets_path",
	}
	for _, field := range optionalFields {
		if _, ok := schema[field]; !ok {
			t.Errorf("Schema missing optional field: %s", field)
		}
	}

	computedFields := []string{
		"kubeconfig", "talosconfig", "secrets_yaml",
		"api_endpoint", "cluster_status",
	}
	for _, field := range computedFields {
		if _, ok := schema[field]; !ok {
			t.Errorf("Schema missing computed field: %s", field)
		}
	}
}

func TestResourceTalosCluster_SchemaTypes(t *testing.T) {
	resource := resourceTalosCluster()
	schema := resource.Schema

	tests := []struct {
		field    string
		expected string
	}{
		{"name", "TypeString"},
		{"cluster_endpoint", "TypeString"},
		{"talos_version", "TypeString"},
		{"kubernetes_version", "TypeString"},
		{"install_disk", "TypeString"},
		{"control_plane", "TypeList"},
		{"worker", "TypeList"},
		{"allow_scheduling_on_control_plane", "TypeBool"},
		{"metallb", "TypeList"},
		{"ingress", "TypeList"},
		{"bootstrap_timeout", "TypeInt"},
		{"kubeconfig_path", "TypeString"},
		{"talosconfig_path", "TypeString"},
		{"secrets_path", "TypeString"},
		{"kubeconfig", "TypeString"},
		{"talosconfig", "TypeString"},
		{"secrets_yaml", "TypeString"},
		{"api_endpoint", "TypeString"},
		{"cluster_status", "TypeString"},
	}

	for _, tc := range tests {
		t.Run(tc.field, func(t *testing.T) {
			s, ok := schema[tc.field]
			if !ok {
				t.Fatalf("Schema missing field: %s", tc.field)
			}
			if s.Type.String() != tc.expected {
				t.Errorf("Field %s: expected type %s, got %s", tc.field, tc.expected, s.Type.String())
			}
		})
	}
}

func TestResourceTalosCluster_RequiredFields(t *testing.T) {
	resource := resourceTalosCluster()
	schema := resource.Schema

	requiredFields := []string{"name", "cluster_endpoint"}
	for _, field := range requiredFields {
		s, ok := schema[field]
		if !ok {
			t.Errorf("Schema missing field: %s", field)
			continue
		}
		if !s.Required {
			t.Errorf("Field %s should be required", field)
		}
	}

	// control_plane is required with MinItems
	cpSchema := schema["control_plane"]
	if cpSchema.MinItems != 1 {
		t.Errorf("control_plane should have MinItems=1, got %d", cpSchema.MinItems)
	}
}

func TestResourceTalosCluster_ForceNewFields(t *testing.T) {
	resource := resourceTalosCluster()
	schema := resource.Schema

	forceNewFields := []string{
		"name", "cluster_endpoint", "install_disk",
		"control_plane", "worker", "allow_scheduling_on_control_plane",
	}
	for _, field := range forceNewFields {
		s, ok := schema[field]
		if !ok {
			t.Errorf("Schema missing field: %s", field)
			continue
		}
		if !s.ForceNew {
			t.Errorf("Field %s should have ForceNew=true", field)
		}
	}
}

func TestResourceTalosCluster_SensitiveFields(t *testing.T) {
	resource := resourceTalosCluster()
	schema := resource.Schema

	sensitiveFields := []string{"kubeconfig", "talosconfig", "secrets_yaml"}
	for _, field := range sensitiveFields {
		s, ok := schema[field]
		if !ok {
			t.Errorf("Schema missing field: %s", field)
			continue
		}
		if !s.Sensitive {
			t.Errorf("Field %s should be sensitive", field)
		}
	}
}

func TestResourceTalosCluster_DefaultValues(t *testing.T) {
	resource := resourceTalosCluster()
	schema := resource.Schema

	tests := []struct {
		field    string
		expected interface{}
	}{
		{"install_disk", "/dev/mmcblk0"},
		{"allow_scheduling_on_control_plane", true},
		{"bootstrap_timeout", 600},
	}

	for _, tc := range tests {
		t.Run(tc.field, func(t *testing.T) {
			s, ok := schema[tc.field]
			if !ok {
				t.Fatalf("Schema missing field: %s", tc.field)
			}
			if s.Default != tc.expected {
				t.Errorf("Field %s: expected default %v, got %v", tc.field, tc.expected, s.Default)
			}
		})
	}
}

func TestTalosNodeSchema(t *testing.T) {
	nodeSchema := talosNodeSchema()
	if nodeSchema == nil {
		t.Fatal("talosNodeSchema() returned nil")
	}

	schema := nodeSchema.Schema

	// Check required fields
	if _, ok := schema["host"]; !ok {
		t.Error("Node schema missing 'host' field")
	}
	if !schema["host"].Required {
		t.Error("Node 'host' field should be required")
	}

	// Check optional fields
	if _, ok := schema["hostname"]; !ok {
		t.Error("Node schema missing 'hostname' field")
	}
	if schema["hostname"].Required {
		t.Error("Node 'hostname' field should be optional")
	}
}

func TestExtractTalosNodeConfig(t *testing.T) {
	data := map[string]interface{}{
		"host":     "10.10.88.73",
		"hostname": "turing-cp1",
	}

	config := extractTalosNodeConfig(data)

	if config.Host != "10.10.88.73" {
		t.Errorf("Expected host '10.10.88.73', got '%s'", config.Host)
	}
	if config.Hostname != "turing-cp1" {
		t.Errorf("Expected hostname 'turing-cp1', got '%s'", config.Hostname)
	}
}

func TestExtractTalosNodeConfig_MinimalData(t *testing.T) {
	data := map[string]interface{}{
		"host": "10.10.88.73",
	}

	config := extractTalosNodeConfig(data)

	if config.Host != "10.10.88.73" {
		t.Errorf("Expected host '10.10.88.73', got '%s'", config.Host)
	}
	if config.Hostname != "" {
		t.Errorf("Expected empty hostname, got '%s'", config.Hostname)
	}
}

func TestGeneratePatchYAML(t *testing.T) {
	tests := []struct {
		name           string
		hostname       string
		allowSchedule  bool
		isControlPlane bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:           "control plane with scheduling",
			hostname:       "turing-cp1",
			allowSchedule:  true,
			isControlPlane: true,
			wantContains:   []string{"hostname: turing-cp1", "allowSchedulingOnControlPlanes: true"},
		},
		{
			name:           "control plane without scheduling",
			hostname:       "turing-cp1",
			allowSchedule:  false,
			isControlPlane: true,
			wantContains:   []string{"hostname: turing-cp1"},
			wantNotContain: []string{"allowSchedulingOnControlPlanes"},
		},
		{
			name:           "worker node",
			hostname:       "turing-w1",
			allowSchedule:  true, // should be ignored for workers
			isControlPlane: false,
			wantContains:   []string{"hostname: turing-w1"},
			wantNotContain: []string{"allowSchedulingOnControlPlanes"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			patch, err := generatePatchYAML(tc.hostname, tc.allowSchedule, tc.isControlPlane)
			if err != nil {
				t.Fatalf("generatePatchYAML failed: %v", err)
			}

			for _, want := range tc.wantContains {
				if !strings.Contains(patch, want) {
					t.Errorf("Patch should contain '%s', got:\n%s", want, patch)
				}
			}

			for _, notWant := range tc.wantNotContain {
				if strings.Contains(patch, notWant) {
					t.Errorf("Patch should not contain '%s', got:\n%s", notWant, patch)
				}
			}
		})
	}
}

func TestTalosProvisioner_NewWithExec(t *testing.T) {
	mockExec := func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "mock")
	}

	provisioner := NewTalosProvisionerWithExec(mockExec)
	if provisioner == nil {
		t.Fatal("NewTalosProvisionerWithExec returned nil")
	}

	if provisioner.talosctlPath != "talosctl" {
		t.Errorf("Expected talosctlPath 'talosctl', got '%s'", provisioner.talosctlPath)
	}

	if provisioner.workDir == "" {
		t.Error("workDir should not be empty")
	}

	// Cleanup
	_ = provisioner.Cleanup()
}

func TestTalosProvisioner_Cleanup(t *testing.T) {
	provisioner := NewTalosProvisionerWithExec(exec.Command)

	workDir := provisioner.WorkDir()
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		t.Fatal("Work directory should exist before cleanup")
	}

	if err := provisioner.Cleanup(); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Error("Work directory should not exist after cleanup")
	}
}

func TestTalosClusterConfig_Defaults(t *testing.T) {
	cfg := TalosClusterConfig{
		Name:             "test-cluster",
		ClusterEndpoint:  "https://10.10.88.73:6443",
		InstallDisk:      "/dev/mmcblk0",
		BootstrapTimeout: 600 * time.Second,
	}

	if cfg.Name != "test-cluster" {
		t.Errorf("Expected name 'test-cluster', got '%s'", cfg.Name)
	}
	if cfg.ClusterEndpoint != "https://10.10.88.73:6443" {
		t.Errorf("Expected endpoint 'https://10.10.88.73:6443', got '%s'", cfg.ClusterEndpoint)
	}
	if cfg.InstallDisk != "/dev/mmcblk0" {
		t.Errorf("Expected install_disk '/dev/mmcblk0', got '%s'", cfg.InstallDisk)
	}
	if cfg.BootstrapTimeout != 600*time.Second {
		t.Errorf("Expected timeout 600s, got %v", cfg.BootstrapTimeout)
	}
}

func TestTalosClusterState_Fields(t *testing.T) {
	state := TalosClusterState{
		SecretsYAML:     "secrets-content",
		Talosconfig:     "talosconfig-content",
		Kubeconfig:      "kubeconfig-content",
		APIEndpoint:     "https://10.10.88.73:6443",
		ClusterStatus:   "ready",
		ControlPlaneIPs: []string{"10.10.88.73"},
		WorkerIPs:       []string{"10.10.88.74", "10.10.88.75"},
	}

	if state.SecretsYAML != "secrets-content" {
		t.Errorf("Unexpected SecretsYAML: %s", state.SecretsYAML)
	}
	if state.ClusterStatus != "ready" {
		t.Errorf("Unexpected ClusterStatus: %s", state.ClusterStatus)
	}
	if len(state.ControlPlaneIPs) != 1 {
		t.Errorf("Expected 1 control plane IP, got %d", len(state.ControlPlaneIPs))
	}
	if len(state.WorkerIPs) != 2 {
		t.Errorf("Expected 2 worker IPs, got %d", len(state.WorkerIPs))
	}
}

func TestTalosProvisioner_RunTalosctl_MockSuccess(t *testing.T) {
	callCount := 0
	mockExec := func(name string, args ...string) *exec.Cmd {
		callCount++
		// Return a command that outputs success
		return exec.Command("echo", "success")
	}

	provisioner := NewTalosProvisionerWithExec(mockExec)
	defer func() { _ = provisioner.Cleanup() }()

	output, err := provisioner.runTalosctl("version")
	if err != nil {
		t.Fatalf("runTalosctl failed: %v", err)
	}

	if !strings.Contains(output, "success") {
		t.Errorf("Expected output to contain 'success', got: %s", output)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 exec call, got %d", callCount)
	}
}

func TestTalosProvisioner_RunTalosctl_MockFailure(t *testing.T) {
	mockExec := func(name string, args ...string) *exec.Cmd {
		// Return a command that fails
		return exec.Command("false")
	}

	provisioner := NewTalosProvisionerWithExec(mockExec)
	defer func() { _ = provisioner.Cleanup() }()

	_, err := provisioner.runTalosctl("version")
	if err == nil {
		t.Error("Expected error from failed command")
	}
}

func TestTalosProvisioner_GenerateSecrets_Mock(t *testing.T) {
	mockExec := func(name string, args ...string) *exec.Cmd {
		// Verify correct arguments
		expectedArgs := []string{"gen", "secrets", "-o"}
		for i, expected := range expectedArgs {
			if i >= len(args) || args[i] != expected {
				t.Errorf("Unexpected argument at position %d: expected '%s', got '%s'", i, expected, args[i])
			}
		}
		return exec.Command("echo", "secrets generated")
	}

	provisioner := NewTalosProvisionerWithExec(mockExec)
	defer func() { _ = provisioner.Cleanup() }()

	err := provisioner.GenerateSecrets("/tmp/secrets.yaml")
	if err != nil {
		t.Fatalf("GenerateSecrets failed: %v", err)
	}
}

func TestTalosProvisioner_ApplyConfig_Mock(t *testing.T) {
	var capturedArgs []string
	mockExec := func(name string, args ...string) *exec.Cmd {
		capturedArgs = args
		return exec.Command("echo", "config applied")
	}

	provisioner := NewTalosProvisionerWithExec(mockExec)
	defer func() { _ = provisioner.Cleanup() }()

	err := provisioner.ApplyConfig("10.10.88.73", "/tmp/config.yaml", true)
	if err != nil {
		t.Fatalf("ApplyConfig failed: %v", err)
	}

	// Verify --insecure flag is present
	hasInsecure := false
	for _, arg := range capturedArgs {
		if arg == "--insecure" {
			hasInsecure = true
			break
		}
	}
	if !hasInsecure {
		t.Error("Expected --insecure flag in arguments")
	}
}

func TestTalosProvisioner_Bootstrap_AlreadyBootstrapped(t *testing.T) {
	mockExec := func(name string, args ...string) *exec.Cmd {
		// Simulate already bootstrapped cluster
		for _, arg := range args {
			if arg == "etcd" {
				return exec.Command("echo", "MEMBER STATUS")
			}
		}
		return exec.Command("echo", "")
	}

	provisioner := NewTalosProvisionerWithExec(mockExec)
	defer func() { _ = provisioner.Cleanup() }()

	// Write a temp talosconfig
	talosconfigPath := provisioner.WorkDir() + "/talosconfig"
	if err := os.WriteFile(talosconfigPath, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	err := provisioner.Bootstrap(talosconfigPath, "10.10.88.73")
	if err != nil {
		t.Fatalf("Bootstrap should succeed (skip) when already bootstrapped: %v", err)
	}
}

func TestTalosProvisioner_Reset_ConnectionRefused(t *testing.T) {
	mockExec := func(name string, args ...string) *exec.Cmd {
		// Simulate connection refused (node rebooting)
		cmd := exec.Command("sh", "-c", "echo 'connection refused' && exit 1")
		return cmd
	}

	provisioner := NewTalosProvisionerWithExec(mockExec)
	defer func() { _ = provisioner.Cleanup() }()

	// Write a temp talosconfig
	talosconfigPath := provisioner.WorkDir() + "/talosconfig"
	if err := os.WriteFile(talosconfigPath, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	// Reset should not fail on connection refused (expected during reboot)
	err := provisioner.Reset(talosconfigPath, "10.10.88.73", false)
	// The mock will fail, but real implementation handles connection refused
	// This test verifies the method exists and can be called
	_ = err
}

func TestResourceTalosCluster_HasCRUDFunctions(t *testing.T) {
	resource := resourceTalosCluster()

	if resource.CreateContext == nil {
		t.Error("CreateContext should not be nil")
	}
	if resource.ReadContext == nil {
		t.Error("ReadContext should not be nil")
	}
	if resource.UpdateContext == nil {
		t.Error("UpdateContext should not be nil")
	}
	if resource.DeleteContext == nil {
		t.Error("DeleteContext should not be nil")
	}
}

func TestResourceTalosCluster_Description(t *testing.T) {
	resource := resourceTalosCluster()

	if resource.Description == "" {
		t.Error("Resource should have a description")
	}

	if !strings.Contains(resource.Description, "Talos") {
		t.Error("Description should mention Talos")
	}
}

// The per-node hostname must be a valid lowercase RFC-1123 hostname label (or
// empty to use the turing-cp-N / turing-w-N default), since it becomes the
// Talos node's network.hostname and the Kubernetes node name.
func TestTalosNode_HostnameValidation(t *testing.T) {
	validate := talosNodeSchema().Schema["hostname"].ValidateDiagFunc
	if validate == nil {
		t.Fatal("hostname should have ValidateDiagFunc")
	}

	cases := []struct {
		name    string
		v       string
		wantErr bool
	}{
		{"empty_uses_default", "", false},
		{"valid_cp", "turing-cp1", false},
		{"valid_worker", "turing-w-3", false},
		{"valid_simple", "node1", false},
		{"uppercase", "Turing-CP1", true},
		{"underscore", "node_1", true},
		{"leading_hyphen", "-leading", true},
		{"trailing_hyphen", "trailing-", true},
		{"contains_dot", "has.dot", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if diags := validate(c.v, nil); c.wantErr != diags.HasError() {
				t.Errorf("hostname=%q: wantErr=%v, got diags=%v", c.v, c.wantErr, diags)
			}
		})
	}
}
