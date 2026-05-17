package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestProvider(t *testing.T) {
	p := Provider()
	if err := p.InternalValidate(); err != nil {
		t.Fatalf("provider internal validation failed: %s", err)
	}
}

func TestProvider_HasRequiredSchema(t *testing.T) {
	p := Provider()

	// Check username field
	if _, ok := p.Schema["username"]; !ok {
		t.Error("provider schema missing 'username' field")
	}

	// Check password field
	if _, ok := p.Schema["password"]; !ok {
		t.Error("provider schema missing 'password' field")
	}

	// Check endpoint field
	if _, ok := p.Schema["endpoint"]; !ok {
		t.Error("provider schema missing 'endpoint' field")
	}
}

func TestProvider_SchemaTypes(t *testing.T) {
	p := Provider()

	tests := []struct {
		field    string
		expected schema.ValueType
	}{
		{"username", schema.TypeString},
		{"password", schema.TypeString},
		{"endpoint", schema.TypeString},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if p.Schema[tt.field].Type != tt.expected {
				t.Errorf("expected %s to be type %v, got %v", tt.field, tt.expected, p.Schema[tt.field].Type)
			}
		})
	}
}

func TestProvider_RequiredFields(t *testing.T) {
	p := Provider()

	if !p.Schema["username"].Required {
		t.Error("username should be required")
	}

	if !p.Schema["password"].Required {
		t.Error("password should be required")
	}

	if p.Schema["endpoint"].Required {
		t.Error("endpoint should not be required")
	}

	if !p.Schema["endpoint"].Optional {
		t.Error("endpoint should be optional")
	}
}

func TestProvider_PasswordIsSensitive(t *testing.T) {
	p := Provider()

	if !p.Schema["password"].Sensitive {
		t.Error("password should be marked as sensitive")
	}
}

func TestProvider_DefaultEndpoint(t *testing.T) {
	expected := "https://turingpi.local"
	if defaultEndpoint != expected {
		t.Errorf("expected default endpoint to be %s, got %s", expected, defaultEndpoint)
	}
}

func TestProvider_EndpointEnvDefault(t *testing.T) {
	p := Provider()

	// Clear any existing env var
	t.Setenv("TURINGPI_ENDPOINT", "")

	// Get the default value
	defaultFunc := p.Schema["endpoint"].DefaultFunc
	if defaultFunc == nil {
		t.Fatal("endpoint should have a DefaultFunc")
	}

	val, err := defaultFunc()
	if err != nil {
		t.Fatalf("DefaultFunc returned error: %s", err)
	}

	if val != defaultEndpoint {
		t.Errorf("expected default value %s, got %v", defaultEndpoint, val)
	}
}

func TestProvider_EndpointEnvOverride(t *testing.T) {
	p := Provider()

	customEndpoint := "https://192.168.1.100"
	t.Setenv("TURINGPI_ENDPOINT", customEndpoint)

	defaultFunc := p.Schema["endpoint"].DefaultFunc
	val, err := defaultFunc()
	if err != nil {
		t.Fatalf("DefaultFunc returned error: %s", err)
	}

	if val != customEndpoint {
		t.Errorf("expected env override value %s, got %v", customEndpoint, val)
	}
}

func TestProvider_UsernameEnvDefault(t *testing.T) {
	p := Provider()

	customUsername := "testuser"
	t.Setenv("TURINGPI_USERNAME", customUsername)

	defaultFunc := p.Schema["username"].DefaultFunc
	if defaultFunc == nil {
		t.Fatal("username should have a DefaultFunc")
	}

	val, err := defaultFunc()
	if err != nil {
		t.Fatalf("DefaultFunc returned error: %s", err)
	}

	if val != customUsername {
		t.Errorf("expected env value %s, got %v", customUsername, val)
	}
}

func TestProvider_PasswordEnvDefault(t *testing.T) {
	p := Provider()

	customPassword := "testpass"
	t.Setenv("TURINGPI_PASSWORD", customPassword)

	defaultFunc := p.Schema["password"].DefaultFunc
	if defaultFunc == nil {
		t.Fatal("password should have a DefaultFunc")
	}

	val, err := defaultFunc()
	if err != nil {
		t.Fatalf("DefaultFunc returned error: %s", err)
	}

	if val != customPassword {
		t.Errorf("expected env value %s, got %v", customPassword, val)
	}
}

func TestProvider_HasResources(t *testing.T) {
	p := Provider()

	expectedResources := []string{
		"turingpi_power",
		"turingpi_flash",
		"turingpi_node",
	}

	for _, resource := range expectedResources {
		if _, ok := p.ResourcesMap[resource]; !ok {
			t.Errorf("provider missing expected resource: %s", resource)
		}
	}
}

func TestProvider_HasConfigureFunc(t *testing.T) {
	p := Provider()

	if p.ConfigureContextFunc == nil {
		t.Error("provider should have a ConfigureContextFunc")
	}
}

func TestProviderConfig_Struct(t *testing.T) {
	config := &ProviderConfig{
		Token:    "test-token",
		Endpoint: "https://test.local",
	}

	if config.Token != "test-token" {
		t.Errorf("expected Token to be 'test-token', got %s", config.Token)
	}

	if config.Endpoint != "https://test.local" {
		t.Errorf("expected Endpoint to be 'https://test.local', got %s", config.Endpoint)
	}
}
