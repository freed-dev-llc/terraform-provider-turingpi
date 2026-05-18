package provider

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// K8sClient provides Kubernetes operations using kubectl
type K8sClient struct {
	kubeconfig     []byte
	kubeconfigPath string
}

// NewK8sClient creates a new Kubernetes client from kubeconfig bytes
func NewK8sClient(kubeconfig []byte) (*K8sClient, error) {
	// Write kubeconfig to a temp file
	tmpFile, err := os.CreateTemp("", "kubeconfig-k8s-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp kubeconfig file: %w", err)
	}

	if err := os.WriteFile(tmpFile.Name(), kubeconfig, 0600); err != nil {
		_ = os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	return &K8sClient{
		kubeconfig:     kubeconfig,
		kubeconfigPath: tmpFile.Name(),
	}, nil
}

// Close cleans up temporary files
func (c *K8sClient) Close() error {
	if c.kubeconfigPath != "" {
		return os.Remove(c.kubeconfigPath)
	}
	return nil
}

// RunKubectl executes a kubectl command and returns the output
func (c *K8sClient) RunKubectl(args ...string) (string, error) {
	cmdArgs := append([]string{"--kubeconfig", c.kubeconfigPath}, args...)
	cmd := exec.Command("kubectl", cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("kubectl %s failed: %s: %w", strings.Join(args, " "), stderr.String(), err)
	}

	return stdout.String(), nil
}

// ApplyManifest applies a YAML manifest to the cluster
func (c *K8sClient) ApplyManifest(manifest string) error {
	// Create a temp file for the manifest
	tmpFile, err := os.CreateTemp("", "manifest-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp manifest file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if err := os.WriteFile(tmpFile.Name(), []byte(manifest), 0600); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	_, err = c.RunKubectl("apply", "-f", tmpFile.Name())
	return err
}

// DeleteManifest deletes resources from a YAML manifest
func (c *K8sClient) DeleteManifest(manifest string) error {
	// Create a temp file for the manifest
	tmpFile, err := os.CreateTemp("", "manifest-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp manifest file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if err := os.WriteFile(tmpFile.Name(), []byte(manifest), 0600); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	_, err = c.RunKubectl("delete", "-f", tmpFile.Name(), "--ignore-not-found")
	return err
}

// WaitForResource waits for a resource to reach a condition
func (c *K8sClient) WaitForResource(resourceType, name, namespace, condition string, timeout string) error {
	args := []string{"wait", resourceType, name, "--for", condition, "--timeout", timeout}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	_, err := c.RunKubectl(args...)
	return err
}

// GetResource gets a resource and returns the raw output
func (c *K8sClient) GetResource(resourceType, name, namespace string) (string, error) {
	args := []string{"get", resourceType, name, "-o", "yaml"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	return c.RunKubectl(args...)
}

// ResourceExists checks if a resource exists
func (c *K8sClient) ResourceExists(resourceType, name, namespace string) bool {
	args := []string{"get", resourceType, name}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	_, err := c.RunKubectl(args...)
	return err == nil
}
