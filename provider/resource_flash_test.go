package provider

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestResourceFlash(t *testing.T) {
	r := resourceFlash()
	if err := r.InternalValidate(nil, true); err != nil {
		t.Fatalf("resource internal validation failed: %s", err)
	}
}

func TestResourceFlash_Schema(t *testing.T) {
	r := resourceFlash()

	for _, f := range []string{"node", "firmware_file", "firmware_url"} {
		if _, ok := r.Schema[f]; !ok {
			t.Errorf("schema missing %q field", f)
		}
	}
}

func TestResourceFlash_SchemaTypes(t *testing.T) {
	r := resourceFlash()

	tests := []struct {
		field    string
		expected schema.ValueType
	}{
		{"node", schema.TypeInt},
		{"firmware_file", schema.TypeString},
		{"firmware_url", schema.TypeString},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if r.Schema[tt.field].Type != tt.expected {
				t.Errorf("expected %s to be type %v, got %v", tt.field, tt.expected, r.Schema[tt.field].Type)
			}
		})
	}
}

// firmware_file and firmware_url are mutually exclusive — exactly one of the
// two must be set on every turingpi_flash resource (issue #66: URL path is the
// only reliable code path; the file path is kept for back-compat with a
// deprecation note).
func TestResourceFlash_RequiredFields(t *testing.T) {
	r := resourceFlash()

	if !r.Schema["node"].Required {
		t.Error("node should be Required")
	}
	if r.Schema["firmware_file"].Required {
		t.Error("firmware_file should now be Optional (use ExactlyOneOf with firmware_url)")
	}
	if r.Schema["firmware_url"].Required {
		t.Error("firmware_url should be Optional (use ExactlyOneOf with firmware_file)")
	}

	for _, f := range []string{"firmware_file", "firmware_url"} {
		exo := r.Schema[f].ExactlyOneOf
		if len(exo) != 2 {
			t.Errorf("%s should declare ExactlyOneOf with 2 entries, got %v", f, exo)
			continue
		}
		got := map[string]bool{}
		for _, s := range exo {
			got[s] = true
		}
		if !got["firmware_file"] || !got["firmware_url"] {
			t.Errorf("%s ExactlyOneOf should contain firmware_file + firmware_url, got %v", f, exo)
		}
	}
}

func TestResourceFlash_ForceNewFields(t *testing.T) {
	r := resourceFlash()

	for _, f := range []string{"node", "firmware_file", "firmware_url"} {
		if !r.Schema[f].ForceNew {
			t.Errorf("%s should have ForceNew=true", f)
		}
	}
}

// firmware_url must look like an http(s) URL — the BMC only accepts that.
func TestResourceFlash_FirmwareURLValidation(t *testing.T) {
	r := resourceFlash()
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
		{"", true},
	}
	for _, c := range cases {
		t.Run(c.v, func(t *testing.T) {
			diags := validate(c.v, nil)
			if c.wantErr && !diags.HasError() {
				t.Errorf("expected error for %q, got none", c.v)
			}
			if !c.wantErr && diags.HasError() {
				t.Errorf("expected no error for %q, got %v", c.v, diags)
			}
		})
	}
}

func TestResourceFlash_HasCRUDFunctions(t *testing.T) {
	r := resourceFlash()

	if r.CreateContext == nil {
		t.Error("resource should have CreateContext function")
	}

	if r.ReadContext == nil {
		t.Error("resource should have ReadContext function")
	}

	// Flash resource should NOT have Update (uses ForceNew instead)
	if r.UpdateContext != nil {
		t.Error("resource should NOT have UpdateContext function (uses ForceNew)")
	}

	if r.DeleteContext == nil {
		t.Error("resource should have DeleteContext function")
	}
}

func TestResourceFlashCreate_FileNotFound(t *testing.T) {
	r := resourceFlash()
	d := r.TestResourceData()

	_ = d.Set("node", 1)
	_ = d.Set("firmware_file", "/nonexistent/firmware.img")

	config := &ProviderConfig{
		Endpoint: "https://example.com",
		Token:    "test-token",
	}

	diags := resourceFlashCreate(context.Background(), d, config)
	if !diags.HasError() {
		t.Fatal("expected error for non-existent file")
	}
	if !strings.Contains(diags[0].Summary, "failed to open firmware file") {
		t.Errorf("expected file open error, got: %s", diags[0].Summary)
	}
}

// writeTempFirmware writes payload to a temp file and returns an open *os.File
// positioned at offset 0 along with its size. The caller is responsible for
// closing the file.
func writeTempFirmware(t *testing.T, payload []byte) (*os.File, int64) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "firmware.img")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("write temp firmware: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open temp firmware: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f, int64(len(payload))
}

// TestUploadFlashStream_NoChunkedEncoding ensures the upload request carries
// an explicit Content-Length and is not sent with Transfer-Encoding: chunked.
// The BMC accepts multipart/form-data with a single "file" part, but older
// firmware rejects chunked bodies (root cause of issue #46). Also round-trips
// the multipart body to confirm the BMC sees the expected structure.
func TestUploadFlashStream_NoChunkedEncoding(t *testing.T) {
	payload := []byte("firmware-bytes-payload")

	var gotContentLength int64
	var gotTransferEncoding []string
	var gotContentType string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentLength = r.ContentLength
		gotTransferEncoding = r.TransferEncoding
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	file, size := writeTempFirmware(t, payload)

	if err := uploadFlashStream(context.Background(), server.URL, "test-token", "handle-123", file, size); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(gotTransferEncoding) != 0 {
		t.Errorf("expected no Transfer-Encoding, got %v", gotTransferEncoding)
	}
	if gotContentLength <= 0 {
		t.Errorf("expected positive Content-Length, got %d", gotContentLength)
	}
	if gotContentLength != int64(len(gotBody)) {
		t.Errorf("Content-Length=%d does not match received body length=%d", gotContentLength, len(gotBody))
	}
	if !strings.HasPrefix(gotContentType, "multipart/form-data") {
		t.Errorf("expected multipart/form-data Content-Type, got %q", gotContentType)
	}

	_, params, err := mime.ParseMediaType(gotContentType)
	if err != nil {
		t.Fatalf("parse Content-Type: %v", err)
	}
	mr := multipart.NewReader(bytes.NewReader(gotBody), params["boundary"])
	part, err := mr.NextPart()
	if err != nil {
		t.Fatalf("read multipart part: %v", err)
	}
	if part.FormName() != "file" {
		t.Errorf("expected form field name=\"file\", got %q", part.FormName())
	}
	gotFile, _ := io.ReadAll(part)
	if string(gotFile) != string(payload) {
		t.Errorf("file part content mismatch: got %q want %q", gotFile, payload)
	}
	if _, err := mr.NextPart(); err != io.EOF {
		t.Errorf("expected exactly one part, got more (err=%v)", err)
	}
}

// TestUploadFlashStream_SurfacesBMCError verifies that when the BMC returns
// a non-2xx status mid-upload, the error message contains the status code and
// response body rather than the previous "io: read/write on closed pipe"
// symptom that masked the real cause in issue #46.
func TestUploadFlashStream_SurfacesBMCError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = w.Write([]byte("payload too large"))
	}))
	defer server.Close()

	file, size := writeTempFirmware(t, []byte("firmware"))

	err := uploadFlashStream(context.Background(), server.URL, "test-token", "handle-123", file, size)
	if err == nil {
		t.Fatal("expected error from non-2xx BMC response")
	}
	if !strings.Contains(err.Error(), "413") {
		t.Errorf("expected status code in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "payload too large") {
		t.Errorf("expected BMC response body in error, got: %v", err)
	}
	if strings.Contains(err.Error(), "closed pipe") {
		t.Errorf("error should not mention closed pipe, got: %v", err)
	}
}

// buildURLFlashInit must percent-encode the firmware URL so that BMC's query
// parser sees a single `file=...` value rather than treating the URL's own
// `&`/`=` characters as additional params.
func TestBuildURLFlashInit(t *testing.T) {
	got := buildURLFlashInit("https://bmc.example", 1, "https://files.example/img.xz?a=1&b=2")
	want := "https://bmc.example/api/bmc?opt=set&type=flash&node=1&file=https%3A%2F%2Ffiles.example%2Fimg.xz%3Fa%3D1%26b%3D2"
	if got != want {
		t.Errorf("buildURLFlashInit:\n got  %q\n want %q", got, want)
	}
}

func TestGetFlashStatus_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := `{"Done": [{"secs": 100, "nanos": 123456}, 12345678]}`
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	status, err := getFlashStatus(context.Background(), server.URL, "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Done == nil {
		t.Error("expected Done status")
	}
}

func TestGetFlashStatus_Flashing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := `{"Flashing": {"bytes_written": 5000, "total_bytes": 10000}}`
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	status, err := getFlashStatus(context.Background(), server.URL, "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Flashing == nil {
		t.Fatal("expected Flashing status")
	}
	if status.Flashing.BytesWritten != 5000 {
		t.Errorf("expected 5000 bytes written, got %d", status.Flashing.BytesWritten)
	}
}

func TestGetFlashStatus_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
	}))
	defer server.Close()

	_, err := getFlashStatus(context.Background(), server.URL, "test-token")
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestResourceFlashRead_DoesNotError(t *testing.T) {
	r := resourceFlash()
	d := r.TestResourceData()
	d.SetId("node-1")

	diags := resourceFlashRead(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
}

func TestResourceFlashDelete_DoesNotError(t *testing.T) {
	r := resourceFlash()
	d := r.TestResourceData()
	d.SetId("node-1")

	diags := resourceFlashDelete(context.Background(), d, nil)
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
}

func TestFlashStatusResponse_isTransferring_NewFormat(t *testing.T) {
	// BMC firmware 2.0.5+ returns an object format
	status := &flashStatusResponse{
		Transferring: []byte(`{"id":395533509,"process_name":"Node 1 upgrade service","size":1103253504,"cancelled":false,"bytes_written":550000000}`),
	}

	inProgress, bytesWritten, totalBytes := status.isTransferring()
	if !inProgress {
		t.Error("expected transfer to be in progress")
	}
	if bytesWritten != 550000000 {
		t.Errorf("expected bytesWritten=550000000, got %d", bytesWritten)
	}
	if totalBytes != 1103253504 {
		t.Errorf("expected totalBytes=1103253504, got %d", totalBytes)
	}
}

func TestFlashStatusResponse_isTransferring_OldFormat(t *testing.T) {
	// Legacy BMC firmware returns an array format [bytes_written, total_bytes]
	status := &flashStatusResponse{
		Transferring: []byte(`[550000000, 1103253504]`),
	}

	inProgress, bytesWritten, totalBytes := status.isTransferring()
	if !inProgress {
		t.Error("expected transfer to be in progress")
	}
	if bytesWritten != 550000000 {
		t.Errorf("expected bytesWritten=550000000, got %d", bytesWritten)
	}
	if totalBytes != 1103253504 {
		t.Errorf("expected totalBytes=1103253504, got %d", totalBytes)
	}
}

func TestFlashStatusResponse_isTransferring_Nil(t *testing.T) {
	status := &flashStatusResponse{
		Transferring: nil,
	}

	inProgress, bytesWritten, totalBytes := status.isTransferring()
	if inProgress {
		t.Error("expected no transfer in progress")
	}
	if bytesWritten != 0 || totalBytes != 0 {
		t.Error("expected zero values for nil Transferring")
	}
}

func TestFlashStatusResponse_isTransferring_Empty(t *testing.T) {
	status := &flashStatusResponse{
		Transferring: []byte{},
	}

	inProgress, bytesWritten, totalBytes := status.isTransferring()
	if inProgress {
		t.Error("expected no transfer in progress for empty data")
	}
	if bytesWritten != 0 || totalBytes != 0 {
		t.Error("expected zero values for empty Transferring")
	}
}

func TestGetFlashStatus_TransferringNewFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// BMC firmware 2.0.5+ response format
		response := `{"Transferring":{"id":395533509,"process_name":"Node 1 upgrade service","size":1103253504,"cancelled":false,"bytes_written":550000000}}`
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	status, err := getFlashStatus(context.Background(), server.URL, "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inProgress, bytesWritten, totalBytes := status.isTransferring()
	if !inProgress {
		t.Error("expected transfer in progress")
	}
	if bytesWritten != 550000000 {
		t.Errorf("expected 550000000 bytes written, got %d", bytesWritten)
	}
	if totalBytes != 1103253504 {
		t.Errorf("expected 1103253504 total bytes, got %d", totalBytes)
	}
}

func TestGetFlashStatus_TransferringOldFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Legacy BMC firmware response format
		response := `{"Transferring":[550000000, 1103253504]}`
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	status, err := getFlashStatus(context.Background(), server.URL, "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inProgress, bytesWritten, totalBytes := status.isTransferring()
	if !inProgress {
		t.Error("expected transfer in progress")
	}
	if bytesWritten != 550000000 {
		t.Errorf("expected 550000000 bytes written, got %d", bytesWritten)
	}
	if totalBytes != 1103253504 {
		t.Errorf("expected 1103253504 total bytes, got %d", totalBytes)
	}
}
