package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	relcommon "helm.sh/helm/v4/pkg/release/common"
	release "helm.sh/helm/v4/pkg/release/v1"
)

// MockHelmClient implements HelmClient for testing
type MockHelmClient struct {
	AddRepositoryFunc      func(name, url string) error
	UpdateRepositoriesFunc func() error
	InstallOrUpgradeFunc   func(ctx context.Context, spec *ChartSpec) (*release.Release, error)
	UninstallReleaseFunc   func(name string) error
	GetReleaseFunc         func(name string) (*release.Release, error)
	ListReleasesFunc       func() ([]*release.Release, error)

	// Track calls for verification
	AddRepositoryCalls      []struct{ Name, URL string }
	InstallOrUpgradeCalls   []*ChartSpec
	UninstallReleaseCalls   []string
	GetReleaseCalls         []string
	UpdateRepositoriesCalls int
	ListReleasesCalls       int
}

func (m *MockHelmClient) AddRepository(name, url string) error {
	m.AddRepositoryCalls = append(m.AddRepositoryCalls, struct{ Name, URL string }{name, url})
	if m.AddRepositoryFunc != nil {
		return m.AddRepositoryFunc(name, url)
	}
	return nil
}

func (m *MockHelmClient) UpdateRepositories() error {
	m.UpdateRepositoriesCalls++
	if m.UpdateRepositoriesFunc != nil {
		return m.UpdateRepositoriesFunc()
	}
	return nil
}

func (m *MockHelmClient) InstallOrUpgradeChart(ctx context.Context, spec *ChartSpec) (*release.Release, error) {
	m.InstallOrUpgradeCalls = append(m.InstallOrUpgradeCalls, spec)
	if m.InstallOrUpgradeFunc != nil {
		return m.InstallOrUpgradeFunc(ctx, spec)
	}
	return &release.Release{
		Name: spec.ReleaseName,
		Info: &release.Info{
			Status: relcommon.StatusDeployed,
		},
		Version: 1,
	}, nil
}

func (m *MockHelmClient) UninstallRelease(name string) error {
	m.UninstallReleaseCalls = append(m.UninstallReleaseCalls, name)
	if m.UninstallReleaseFunc != nil {
		return m.UninstallReleaseFunc(name)
	}
	return nil
}

func (m *MockHelmClient) GetRelease(name string) (*release.Release, error) {
	m.GetReleaseCalls = append(m.GetReleaseCalls, name)
	if m.GetReleaseFunc != nil {
		return m.GetReleaseFunc(name)
	}
	return &release.Release{
		Name: name,
		Info: &release.Info{
			Status: relcommon.StatusDeployed,
		},
		Version: 1,
	}, nil
}

func (m *MockHelmClient) ListReleases() ([]*release.Release, error) {
	m.ListReleasesCalls++
	if m.ListReleasesFunc != nil {
		return m.ListReleasesFunc()
	}
	return []*release.Release{}, nil
}

// Test that MockHelmClient implements HelmClient interface
func TestMockHelmClient_ImplementsInterface(t *testing.T) {
	var _ HelmClient = (*MockHelmClient)(nil)
}

// Test that RealHelmClient implements HelmClient interface
func TestRealHelmClient_ImplementsInterface(t *testing.T) {
	var _ HelmClient = (*RealHelmClient)(nil)
}

// Test ChartSpec defaults
func TestChartSpec_Defaults(t *testing.T) {
	spec := &ChartSpec{
		ReleaseName: "test-release",
		ChartName:   "test/chart",
		Namespace:   "default",
	}

	if spec.ReleaseName != "test-release" {
		t.Errorf("expected ReleaseName 'test-release', got %q", spec.ReleaseName)
	}
	if spec.Timeout != 0 {
		t.Errorf("expected zero Timeout (will use default), got %v", spec.Timeout)
	}
	if spec.Wait {
		t.Error("expected Wait to be false by default")
	}
	if spec.Atomic {
		t.Error("expected Atomic to be false by default")
	}
}

// Test AddRepository
func TestMockHelmClient_AddRepository(t *testing.T) {
	mock := &MockHelmClient{}

	err := mock.AddRepository("metallb", "https://metallb.github.io/metallb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.AddRepositoryCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.AddRepositoryCalls))
	}

	call := mock.AddRepositoryCalls[0]
	if call.Name != "metallb" {
		t.Errorf("expected name 'metallb', got %q", call.Name)
	}
	if call.URL != "https://metallb.github.io/metallb" {
		t.Errorf("expected URL 'https://metallb.github.io/metallb', got %q", call.URL)
	}
}

// Test AddRepository error handling
func TestMockHelmClient_AddRepository_Error(t *testing.T) {
	mock := &MockHelmClient{
		AddRepositoryFunc: func(name, url string) error {
			return fmt.Errorf("network error")
		},
	}

	err := mock.AddRepository("test", "https://example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Test InstallOrUpgradeChart
func TestMockHelmClient_InstallOrUpgradeChart(t *testing.T) {
	mock := &MockHelmClient{}
	ctx := context.Background()

	spec := &ChartSpec{
		ReleaseName:     "metallb",
		ChartName:       "metallb/metallb",
		Namespace:       "metallb-system",
		CreateNamespace: true,
		Wait:            true,
	}

	rel, err := mock.InstallOrUpgradeChart(ctx, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rel.Name != "metallb" {
		t.Errorf("expected release name 'metallb', got %q", rel.Name)
	}
	if rel.Info.Status != relcommon.StatusDeployed {
		t.Errorf("expected status Deployed, got %v", rel.Info.Status)
	}

	if len(mock.InstallOrUpgradeCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.InstallOrUpgradeCalls))
	}

	if mock.InstallOrUpgradeCalls[0].ReleaseName != "metallb" {
		t.Errorf("expected ReleaseName 'metallb', got %q", mock.InstallOrUpgradeCalls[0].ReleaseName)
	}
}

// Test InstallOrUpgradeChart with custom return
func TestMockHelmClient_InstallOrUpgradeChart_CustomReturn(t *testing.T) {
	mock := &MockHelmClient{
		InstallOrUpgradeFunc: func(ctx context.Context, spec *ChartSpec) (*release.Release, error) {
			return &release.Release{
				Name:    spec.ReleaseName,
				Version: 5,
				Info: &release.Info{
					Status:      relcommon.StatusDeployed,
					Description: "Custom install",
				},
			}, nil
		},
	}
	ctx := context.Background()

	spec := &ChartSpec{
		ReleaseName: "custom-release",
		ChartName:   "custom/chart",
		Namespace:   "default",
	}

	rel, err := mock.InstallOrUpgradeChart(ctx, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rel.Version != 5 {
		t.Errorf("expected version 5, got %d", rel.Version)
	}
	if rel.Info.Description != "Custom install" {
		t.Errorf("expected description 'Custom install', got %q", rel.Info.Description)
	}
}

// Test InstallOrUpgradeChart error handling
func TestMockHelmClient_InstallOrUpgradeChart_Error(t *testing.T) {
	mock := &MockHelmClient{
		InstallOrUpgradeFunc: func(ctx context.Context, spec *ChartSpec) (*release.Release, error) {
			return nil, fmt.Errorf("chart not found")
		},
	}
	ctx := context.Background()

	spec := &ChartSpec{
		ReleaseName: "test",
		ChartName:   "nonexistent/chart",
		Namespace:   "default",
	}

	_, err := mock.InstallOrUpgradeChart(ctx, spec)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Test UninstallRelease
func TestMockHelmClient_UninstallRelease(t *testing.T) {
	mock := &MockHelmClient{}

	err := mock.UninstallRelease("test-release")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.UninstallReleaseCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.UninstallReleaseCalls))
	}

	if mock.UninstallReleaseCalls[0] != "test-release" {
		t.Errorf("expected 'test-release', got %q", mock.UninstallReleaseCalls[0])
	}
}

// Test GetRelease
func TestMockHelmClient_GetRelease(t *testing.T) {
	mock := &MockHelmClient{}

	rel, err := mock.GetRelease("test-release")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rel.Name != "test-release" {
		t.Errorf("expected name 'test-release', got %q", rel.Name)
	}

	if len(mock.GetReleaseCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.GetReleaseCalls))
	}
}

// Test ListReleases
func TestMockHelmClient_ListReleases(t *testing.T) {
	mock := &MockHelmClient{
		ListReleasesFunc: func() ([]*release.Release, error) {
			return []*release.Release{
				{Name: "release1", Info: &release.Info{Status: relcommon.StatusDeployed}},
				{Name: "release2", Info: &release.Info{Status: relcommon.StatusDeployed}},
			}, nil
		},
	}

	releases, err := mock.ListReleases()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(releases) != 2 {
		t.Errorf("expected 2 releases, got %d", len(releases))
	}

	if mock.ListReleasesCalls != 1 {
		t.Errorf("expected 1 call, got %d", mock.ListReleasesCalls)
	}
}

// Test UpdateRepositories
func TestMockHelmClient_UpdateRepositories(t *testing.T) {
	mock := &MockHelmClient{}

	err := mock.UpdateRepositories()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.UpdateRepositoriesCalls != 1 {
		t.Errorf("expected 1 call, got %d", mock.UpdateRepositoriesCalls)
	}
}

// Test DeployHelmChartWithClient
func TestDeployHelmChartWithClient(t *testing.T) {
	mock := &MockHelmClient{}
	ctx := context.Background()

	spec := &ChartSpec{
		ReleaseName:     "test-release",
		ChartName:       "test/chart",
		Namespace:       "default",
		CreateNamespace: true,
		Wait:            true,
	}

	err := DeployHelmChartWithClient(ctx, mock, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.InstallOrUpgradeCalls) != 1 {
		t.Fatalf("expected 1 install call, got %d", len(mock.InstallOrUpgradeCalls))
	}
}

// Test DeployFromRepositoryWithClient
func TestDeployFromRepositoryWithClient(t *testing.T) {
	mock := &MockHelmClient{}
	ctx := context.Background()

	spec := &ChartSpec{
		ReleaseName:     "metallb",
		ChartName:       "metallb/metallb",
		Namespace:       "metallb-system",
		CreateNamespace: true,
		Wait:            true,
	}

	err := DeployFromRepositoryWithClient(ctx, mock, "metallb", "https://metallb.github.io/metallb", spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify repo was added
	if len(mock.AddRepositoryCalls) != 1 {
		t.Fatalf("expected 1 AddRepository call, got %d", len(mock.AddRepositoryCalls))
	}
	if mock.AddRepositoryCalls[0].Name != "metallb" {
		t.Errorf("expected repo name 'metallb', got %q", mock.AddRepositoryCalls[0].Name)
	}

	// Verify chart was installed
	if len(mock.InstallOrUpgradeCalls) != 1 {
		t.Fatalf("expected 1 install call, got %d", len(mock.InstallOrUpgradeCalls))
	}
}

// Test DeployFromRepositoryWithClient repo error
func TestDeployFromRepositoryWithClient_RepoError(t *testing.T) {
	mock := &MockHelmClient{
		AddRepositoryFunc: func(name, url string) error {
			return fmt.Errorf("network error")
		},
	}
	ctx := context.Background()

	spec := &ChartSpec{
		ReleaseName: "test",
		ChartName:   "test/chart",
		Namespace:   "default",
	}

	err := DeployFromRepositoryWithClient(ctx, mock, "test", "https://example.com", spec)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Chart should not be installed if repo add fails
	if len(mock.InstallOrUpgradeCalls) != 0 {
		t.Errorf("expected 0 install calls, got %d", len(mock.InstallOrUpgradeCalls))
	}
}

// Test WaitForHelmReleaseWithClient success
func TestWaitForHelmReleaseWithClient_Success(t *testing.T) {
	mock := &MockHelmClient{
		GetReleaseFunc: func(name string) (*release.Release, error) {
			return &release.Release{
				Name: name,
				Info: &release.Info{
					Status: relcommon.StatusDeployed,
				},
			}, nil
		},
	}

	err := WaitForHelmReleaseWithClient(mock, "test-release", 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test WaitForHelmReleaseWithClient with pending then deployed
func TestWaitForHelmReleaseWithClient_PendingThenDeployed(t *testing.T) {
	callCount := 0
	mock := &MockHelmClient{
		GetReleaseFunc: func(name string) (*release.Release, error) {
			callCount++
			if callCount < 2 {
				return &release.Release{
					Name: name,
					Info: &release.Info{
						Status: relcommon.StatusPendingInstall,
					},
				}, nil
			}
			return &release.Release{
				Name: name,
				Info: &release.Info{
					Status: relcommon.StatusDeployed,
				},
			}, nil
		},
	}

	err := WaitForHelmReleaseWithClient(mock, "test-release", 15*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount < 2 {
		t.Errorf("expected at least 2 calls, got %d", callCount)
	}
}

// Test WaitForHelmReleaseWithClient failure
func TestWaitForHelmReleaseWithClient_Failed(t *testing.T) {
	mock := &MockHelmClient{
		GetReleaseFunc: func(name string) (*release.Release, error) {
			return &release.Release{
				Name: name,
				Info: &release.Info{
					Status:      relcommon.StatusFailed,
					Description: "install failed",
				},
			}, nil
		},
	}

	err := WaitForHelmReleaseWithClient(mock, "test-release", 10*time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Test WaitForHelmReleaseWithClient timeout
func TestWaitForHelmReleaseWithClient_Timeout(t *testing.T) {
	mock := &MockHelmClient{
		GetReleaseFunc: func(name string) (*release.Release, error) {
			return nil, fmt.Errorf("release not found")
		},
	}

	err := WaitForHelmReleaseWithClient(mock, "test-release", 1*time.Second)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// Test ChartSpec with ValuesYaml
func TestChartSpec_ValuesYaml(t *testing.T) {
	mock := &MockHelmClient{}
	ctx := context.Background()

	valuesYaml := `
controller:
  replicaCount: 2
speaker:
  enabled: true
`

	spec := &ChartSpec{
		ReleaseName:     "metallb",
		ChartName:       "metallb/metallb",
		Namespace:       "metallb-system",
		ValuesYaml:      valuesYaml,
		CreateNamespace: true,
	}

	_, err := mock.InstallOrUpgradeChart(ctx, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.InstallOrUpgradeCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.InstallOrUpgradeCalls))
	}

	if mock.InstallOrUpgradeCalls[0].ValuesYaml != valuesYaml {
		t.Error("ValuesYaml not passed correctly")
	}
}
