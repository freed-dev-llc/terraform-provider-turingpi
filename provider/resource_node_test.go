package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// powerRecorder captures the URLs of power set-requests a test BMC received so
// tests can assert what was actually sent (node index and on/off value).
type powerRecorder struct {
	mu   sync.Mutex
	sets []string
}

func (p *powerRecorder) add(u string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sets = append(p.sets, u)
}

func (p *powerRecorder) contains(substr string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, u := range p.sets {
		if strings.Contains(u, substr) {
			return true
		}
	}
	return false
}

// newPowerMockServer returns a test BMC that answers power get/set requests and
// a recorder capturing every set-request URL, so resource_node CRUD paths
// (which make real BMC calls) can run offline and be asserted against.
func newPowerMockServer(t *testing.T) (*httptest.Server, *powerRecorder) {
	t.Helper()
	rec := &powerRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// type=get power returns a status payload; set requests are recorded.
		if r.URL.Query().Get("opt") == "get" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"response": [][]interface{}{
					{"node1", float64(1)},
					{"node2", float64(0)},
					{"node3", float64(0)},
					{"node4", float64(0)},
				},
			})
			return
		}
		rec.add(r.URL.String())
		w.WriteHeader(http.StatusOK)
	}))
	return srv, rec
}

func TestResourceNode(t *testing.T) {
	r := resourceNode()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("resource internal validation failed: %s", err)
	}
}

func TestResourceNode_Schema(t *testing.T) {
	r := resourceNode()

	expectedFields := []string{
		"node",
		"firmware_url",
		"firmware_file",
		"power_state",
		"boot_check",
		"login_prompt_timeout",
		"boot_check_pattern",
	}

	for _, field := range expectedFields {
		if _, ok := r.Schema[field]; !ok {
			t.Errorf("schema missing '%s' field", field)
		}
	}
}

func TestResourceNode_SchemaTypes(t *testing.T) {
	r := resourceNode()

	tests := []struct {
		field    string
		expected schema.ValueType
	}{
		{"node", schema.TypeInt},
		{"firmware_url", schema.TypeString},
		{"firmware_file", schema.TypeString},
		{"power_state", schema.TypeString},
		{"boot_check", schema.TypeBool},
		{"login_prompt_timeout", schema.TypeInt},
		{"boot_check_pattern", schema.TypeString},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if r.Schema[tt.field].Type != tt.expected {
				t.Errorf("expected %s to be type %v, got %v", tt.field, tt.expected, r.Schema[tt.field].Type)
			}
		})
	}
}

// firmware_url must look like an http(s) URL — the BMC only accepts that, and
// it is the reliable flash path (vs the deprecated firmware_file).
func TestResourceNode_FirmwareURLValidation(t *testing.T) {
	r := resourceNode()
	validate := r.Schema["firmware_url"].ValidateDiagFunc
	if validate == nil {
		t.Fatal("firmware_url should have ValidateDiagFunc")
	}

	cases := []struct {
		v       string
		wantErr bool
	}{
		{"http://example.com/img.xz", false},
		{"https://example.com/img.xz", false},
		{"ftp://example.com/img.xz", true},
		{"/local/path", true},
	}
	for _, c := range cases {
		t.Run(c.v, func(t *testing.T) {
			if diags := validate(c.v, nil); c.wantErr != diags.HasError() {
				t.Errorf("firmware_url=%q: wantErr=%v, got diags=%v", c.v, c.wantErr, diags)
			}
		})
	}
}

// firmware_url and firmware_file are mutually exclusive via ConflictsWith.
func TestResourceNode_FirmwareConflictsWith(t *testing.T) {
	r := resourceNode()
	if got := r.Schema["firmware_url"].ConflictsWith; len(got) != 1 || got[0] != "firmware_file" {
		t.Errorf("firmware_url ConflictsWith should be [firmware_file], got %v", got)
	}
	if got := r.Schema["firmware_file"].ConflictsWith; len(got) != 1 || got[0] != "firmware_url" {
		t.Errorf("firmware_file ConflictsWith should be [firmware_url], got %v", got)
	}
	if r.Schema["firmware_file"].Deprecated == "" {
		t.Error("firmware_file should be marked Deprecated")
	}
}

func TestResourceNode_RequiredFields(t *testing.T) {
	r := resourceNode()

	if !r.Schema["node"].Required {
		t.Error("node should be required")
	}
}

func TestResourceNode_OptionalFields(t *testing.T) {
	r := resourceNode()

	optionalFields := []string{
		"firmware_url",
		"firmware_file",
		"power_state",
		"boot_check",
		"login_prompt_timeout",
		"boot_check_pattern",
	}

	for _, field := range optionalFields {
		if !r.Schema[field].Optional {
			t.Errorf("%s should be optional", field)
		}
	}
}

func TestResourceNode_DefaultValues(t *testing.T) {
	r := resourceNode()

	// power_state defaults to "on"
	if r.Schema["power_state"].Default != "on" {
		t.Errorf("power_state should default to 'on', got %v", r.Schema["power_state"].Default)
	}

	// boot_check defaults to false
	if r.Schema["boot_check"].Default != false {
		t.Errorf("boot_check should default to false, got %v", r.Schema["boot_check"].Default)
	}

	// login_prompt_timeout defaults to 60
	if r.Schema["login_prompt_timeout"].Default != 60 {
		t.Errorf("login_prompt_timeout should default to 60, got %v", r.Schema["login_prompt_timeout"].Default)
	}

	// boot_check_pattern defaults to "login:"
	if r.Schema["boot_check_pattern"].Default != "login:" {
		t.Errorf("boot_check_pattern should default to 'login:', got %v", r.Schema["boot_check_pattern"].Default)
	}
}

func TestResourceNode_HasCRUDFunctions(t *testing.T) {
	r := resourceNode()

	if r.CreateContext == nil {
		t.Error("resource should have CreateContext function")
	}

	if r.ReadContext == nil {
		t.Error("resource should have ReadContext function")
	}

	if r.UpdateContext == nil {
		t.Error("resource should have UpdateContext function")
	}

	if r.DeleteContext == nil {
		t.Error("resource should have DeleteContext function")
	}
}

func TestResourceNodeProvision_SetsId(t *testing.T) {
	server, _ := newPowerMockServer(t)
	defer server.Close()

	r := resourceNode()
	d := r.TestResourceData()

	_ = d.Set("node", 1)
	_ = d.Set("power_state", "on")
	_ = d.Set("boot_check", false)

	config := &ProviderConfig{
		Token:    "test-token",
		Endpoint: server.URL,
	}

	diags := resourceNodeCreate(context.Background(), d, config)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}

	expectedId := "node-1"
	if d.Id() != expectedId {
		t.Errorf("expected ID %s, got %s", expectedId, d.Id())
	}
}

func TestResourceNodeProvision_DifferentNodes(t *testing.T) {
	server, _ := newPowerMockServer(t)
	defer server.Close()

	r := resourceNode()

	testCases := []struct {
		node       int
		expectedId string
	}{
		{1, "node-1"},
		{2, "node-2"},
		{3, "node-3"},
		{4, "node-4"},
	}

	config := &ProviderConfig{
		Token:    "test-token",
		Endpoint: server.URL,
	}

	for _, tc := range testCases {
		t.Run(tc.expectedId, func(t *testing.T) {
			d := r.TestResourceData()
			_ = d.Set("node", tc.node)
			_ = d.Set("power_state", "on")
			_ = d.Set("boot_check", false)

			diags := resourceNodeCreate(context.Background(), d, config)
			if diags.HasError() {
				t.Fatalf("unexpected error: %s", diags[0].Summary)
			}

			if d.Id() != tc.expectedId {
				t.Errorf("expected ID %s, got %s", tc.expectedId, d.Id())
			}
		})
	}
}

func TestResourceNodeProvision_PowerStateOn(t *testing.T) {
	server, rec := newPowerMockServer(t)
	defer server.Close()

	r := resourceNode()
	d := r.TestResourceData()

	_ = d.Set("node", 1)
	_ = d.Set("power_state", "on")
	_ = d.Set("boot_check", false)

	config := &ProviderConfig{
		Token:    "test-token",
		Endpoint: server.URL,
	}

	diags := resourceNodeCreate(context.Background(), d, config)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if !rec.contains("node1=1") {
		t.Error("expected a power-on request (node1=1) to the BMC")
	}
}

func TestResourceNodeProvision_PowerStateOff(t *testing.T) {
	server, rec := newPowerMockServer(t)
	defer server.Close()

	r := resourceNode()
	d := r.TestResourceData()

	_ = d.Set("node", 1)
	_ = d.Set("power_state", "off")
	_ = d.Set("boot_check", false)

	config := &ProviderConfig{
		Token:    "test-token",
		Endpoint: server.URL,
	}

	diags := resourceNodeCreate(context.Background(), d, config)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if !rec.contains("node1=0") {
		t.Error("expected a power-off request (node1=0) to the BMC")
	}
}

func TestResourceNodeProvision_FirmwareFileNotFound(t *testing.T) {
	r := resourceNode()
	d := r.TestResourceData()

	_ = d.Set("node", 1)
	_ = d.Set("firmware_file", "/nonexistent/firmware.img")
	_ = d.Set("power_state", "on")
	_ = d.Set("boot_check", false)

	config := &ProviderConfig{
		Token:    "test-token",
		Endpoint: "https://test.local",
	}

	// Providing firmware_file now triggers a real flash attempt, so a missing
	// file must surface as an error rather than silently succeeding.
	diags := resourceNodeCreate(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error for missing firmware file, got nil")
	}
}

func TestResourceNodeProvision_WithBootCheck(t *testing.T) {
	// Create mock server that returns login prompt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Boot complete\nlogin:"))
	}))
	defer server.Close()

	r := resourceNode()
	d := r.TestResourceData()

	_ = d.Set("node", 1)
	_ = d.Set("power_state", "on")
	_ = d.Set("boot_check", true)
	_ = d.Set("login_prompt_timeout", 1)
	_ = d.Set("boot_check_pattern", "login:")

	config := &ProviderConfig{
		Token:    "test-token",
		Endpoint: server.URL,
	}

	diags := resourceNodeCreate(context.Background(), d, config)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
}

func TestResourceNodeProvision_BootCheckTimeout(t *testing.T) {
	// Create mock server that never returns login prompt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Still booting..."))
	}))
	defer server.Close()

	r := resourceNode()
	d := r.TestResourceData()

	_ = d.Set("node", 1)
	_ = d.Set("power_state", "on")
	_ = d.Set("boot_check", true)
	_ = d.Set("login_prompt_timeout", 1)
	_ = d.Set("boot_check_pattern", "login:")

	config := &ProviderConfig{
		Token:    "test-token",
		Endpoint: server.URL,
	}

	diags := resourceNodeCreate(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error for boot check timeout, got nil")
	}
}

func TestResourceNodeProvision_CustomBootCheckPattern(t *testing.T) {
	// Create mock server that returns Talos boot message
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[talos] machine is running and ready"))
	}))
	defer server.Close()

	r := resourceNode()
	d := r.TestResourceData()

	_ = d.Set("node", 1)
	_ = d.Set("power_state", "on")
	_ = d.Set("boot_check", true)
	_ = d.Set("login_prompt_timeout", 1)
	_ = d.Set("boot_check_pattern", "machine is running and ready")

	config := &ProviderConfig{
		Token:    "test-token",
		Endpoint: server.URL,
	}

	diags := resourceNodeCreate(context.Background(), d, config)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
}

func TestResourceNodeStatus_SetsPowerState(t *testing.T) {
	server, _ := newPowerMockServer(t)
	defer server.Close()

	r := resourceNode()
	d := r.TestResourceData()

	_ = d.Set("node", 1)
	d.SetId("node-1")

	config := &ProviderConfig{Token: "test-token", Endpoint: server.URL}
	diags := resourceNodeStatus(context.Background(), d, config)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}

	// Mock reports node1 powered on.
	powerState := d.Get("power_state").(string)
	if powerState != "on" {
		t.Errorf("expected power_state 'on', got %s", powerState)
	}
}

func TestResourceNodeDelete_TurnsOffNode(t *testing.T) {
	server, rec := newPowerMockServer(t)
	defer server.Close()

	r := resourceNode()
	d := r.TestResourceData()

	_ = d.Set("node", 1)
	d.SetId("node-1")

	config := &ProviderConfig{Token: "test-token", Endpoint: server.URL}
	diags := resourceNodeDelete(context.Background(), d, config)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if !rec.contains("node1=0") {
		t.Error("expected delete to power the node off (node1=0)")
	}
}

func TestResourceNodeDelete_DifferentNodes(t *testing.T) {
	server, _ := newPowerMockServer(t)
	defer server.Close()

	r := resourceNode()
	config := &ProviderConfig{Token: "test-token", Endpoint: server.URL}

	nodes := []int{1, 2, 3, 4}

	for _, node := range nodes {
		t.Run("node_"+string(rune('0'+node)), func(t *testing.T) {
			d := r.TestResourceData()
			_ = d.Set("node", node)
			d.SetId("node-" + string(rune('0'+node)))

			diags := resourceNodeDelete(context.Background(), d, config)
			if diags.HasError() {
				t.Fatalf("unexpected error: %s", diags[0].Summary)
			}
		})
	}
}
