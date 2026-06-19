package provider

import (
	"context"
	"fmt"
	"os"
	"time"

	helmclient "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v4/pkg/kube"
	relcommon "helm.sh/helm/v4/pkg/release/common"
	release "helm.sh/helm/v4/pkg/release/v1"
	repo "helm.sh/helm/v4/pkg/repo/v1"
)

// HelmClient interface for Helm operations - allows mocking in tests
type HelmClient interface {
	AddRepository(name, url string) error
	UpdateRepositories() error
	InstallOrUpgradeChart(ctx context.Context, spec *ChartSpec) (*release.Release, error)
	UninstallRelease(name string) error
	GetRelease(name string) (*release.Release, error)
	ListReleases() ([]*release.Release, error)
}

// ChartSpec defines a Helm chart deployment specification
type ChartSpec struct {
	ReleaseName     string                 // Name of the Helm release
	ChartName       string                 // Chart name (e.g., "metallb/metallb" or local path)
	Namespace       string                 // Target namespace
	Version         string                 // Chart version (optional, empty = latest)
	Values          map[string]interface{} // Inline values
	ValuesYaml      string                 // YAML string of values
	CreateNamespace bool                   // Create namespace if it doesn't exist
	Wait            bool                   // Wait for resources to be ready
	Timeout         time.Duration          // Timeout for wait operations
	Atomic          bool                   // Rollback on failure
}

// RealHelmClient implements HelmClient using mittwald/go-helm-client
type RealHelmClient struct {
	client    helmclient.Client
	namespace string
}

// NewHelmClient creates a new Helm client from a kubeconfig file path
func NewHelmClient(kubeconfigPath, namespace string) (HelmClient, error) {
	kubeconfig, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	return NewHelmClientFromBytes(kubeconfig, namespace)
}

// NewHelmClientFromBytes creates a new Helm client from kubeconfig bytes
func NewHelmClientFromBytes(kubeconfig []byte, namespace string) (HelmClient, error) {
	if namespace == "" {
		namespace = "default"
	}

	opt := &helmclient.KubeConfClientOptions{
		Options: &helmclient.Options{
			Namespace:        namespace,
			RepositoryCache:  "/tmp/.helmcache",
			RepositoryConfig: "/tmp/.helmrepo",
			Debug:            false,
			Linting:          false,
		},
		KubeConfig: kubeconfig,
	}

	client, err := helmclient.NewClientFromKubeConf(opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create Helm client: %w", err)
	}

	return &RealHelmClient{
		client:    client,
		namespace: namespace,
	}, nil
}

// AddRepository adds or updates a Helm chart repository
func (c *RealHelmClient) AddRepository(name, url string) error {
	chartRepo := repo.Entry{
		Name: name,
		URL:  url,
	}

	if err := c.client.AddOrUpdateChartRepo(chartRepo); err != nil {
		return fmt.Errorf("failed to add repository %s: %w", name, err)
	}

	return nil
}

// UpdateRepositories updates all configured Helm repositories
func (c *RealHelmClient) UpdateRepositories() error {
	if err := c.client.UpdateChartRepos(); err != nil {
		return fmt.Errorf("failed to update repositories: %w", err)
	}
	return nil
}

// InstallOrUpgradeChart installs or upgrades a Helm chart
func (c *RealHelmClient) InstallOrUpgradeChart(ctx context.Context, spec *ChartSpec) (*release.Release, error) {
	if spec.Timeout == 0 {
		spec.Timeout = 5 * time.Minute
	}

	chartSpec := helmclient.ChartSpec{
		ReleaseName:       spec.ReleaseName,
		ChartName:         spec.ChartName,
		Namespace:         spec.Namespace,
		Version:           spec.Version,
		ValuesYaml:        spec.ValuesYaml,
		CreateNamespace:   spec.CreateNamespace,
		Timeout:           spec.Timeout,
		RollbackOnFailure: spec.Atomic,
		CleanupOnFail:     spec.Atomic, // Clean up on failure if atomic
	}
	if spec.Wait {
		// helm v4 replaced the bool Wait with a strategy; "watcher" is the
		// modern equivalent of waiting for resources to become ready.
		chartSpec.WaitStrategy = kube.StatusWatcherStrategy
	}

	rel, err := c.client.InstallOrUpgradeChart(ctx, &chartSpec, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to install/upgrade chart %s: %w", spec.ChartName, err)
	}

	return rel, nil
}

// UninstallRelease uninstalls a Helm release
func (c *RealHelmClient) UninstallRelease(name string) error {
	if err := c.client.UninstallReleaseByName(name); err != nil {
		return fmt.Errorf("failed to uninstall release %s: %w", name, err)
	}
	return nil
}

// GetRelease returns information about an installed release
func (c *RealHelmClient) GetRelease(name string) (*release.Release, error) {
	rel, err := c.client.GetRelease(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get release %s: %w", name, err)
	}
	return rel, nil
}

// ListReleases lists all releases in the configured namespace
func (c *RealHelmClient) ListReleases() ([]*release.Release, error) {
	releases, err := c.client.ListDeployedReleases()
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}
	return releases, nil
}

// DeployHelmChart is a high-level convenience function for deploying a chart
func DeployHelmChart(ctx context.Context, kubeconfigPath string, spec *ChartSpec) error {
	client, err := NewHelmClient(kubeconfigPath, spec.Namespace)
	if err != nil {
		return err
	}

	_, err = client.InstallOrUpgradeChart(ctx, spec)
	return err
}

// DeployHelmChartWithClient deploys a chart using a provided client (for testing)
func DeployHelmChartWithClient(ctx context.Context, client HelmClient, spec *ChartSpec) error {
	_, err := client.InstallOrUpgradeChart(ctx, spec)
	return err
}

// DeployFromRepository adds a repo and deploys a chart in one call
func DeployFromRepository(ctx context.Context, kubeconfigPath, repoName, repoURL string, spec *ChartSpec) error {
	client, err := NewHelmClient(kubeconfigPath, spec.Namespace)
	if err != nil {
		return err
	}

	if err := client.AddRepository(repoName, repoURL); err != nil {
		return fmt.Errorf("failed to add repository: %w", err)
	}

	_, err = client.InstallOrUpgradeChart(ctx, spec)
	return err
}

// DeployFromRepositoryWithClient adds a repo and deploys using a provided client (for testing)
func DeployFromRepositoryWithClient(ctx context.Context, client HelmClient, repoName, repoURL string, spec *ChartSpec) error {
	if err := client.AddRepository(repoName, repoURL); err != nil {
		return fmt.Errorf("failed to add repository: %w", err)
	}

	_, err := client.InstallOrUpgradeChart(ctx, spec)
	return err
}

// WaitForHelmRelease waits for a release to reach deployed status
func WaitForHelmRelease(kubeconfigPath, name, namespace string, timeout time.Duration) error {
	client, err := NewHelmClient(kubeconfigPath, namespace)
	if err != nil {
		return err
	}

	return WaitForHelmReleaseWithClient(client, name, timeout)
}

// WaitForHelmReleaseWithClient waits for a release using a provided client (for testing)
func WaitForHelmReleaseWithClient(client HelmClient, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		rel, err := client.GetRelease(name)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		switch rel.Info.Status {
		case relcommon.StatusDeployed:
			return nil
		case relcommon.StatusFailed:
			return fmt.Errorf("release %s failed: %s", name, rel.Info.Description)
		case relcommon.StatusPendingInstall, relcommon.StatusPendingUpgrade, relcommon.StatusPendingRollback:
			// Still in progress, keep waiting
		default:
			// Unknown status, keep waiting
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for release %s after %v", name, timeout)
}
