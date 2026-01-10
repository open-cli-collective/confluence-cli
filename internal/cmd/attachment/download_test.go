package attachment

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rianjs/confluence-cli/api"
)

// TestSanitizeAttachmentFilename validates that filepath.Base correctly
// sanitizes malicious filenames that could be used for path traversal attacks.
func TestSanitizeAttachmentFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		valid    bool
	}{
		{
			name:     "normal filename",
			input:    "document.pdf",
			expected: "document.pdf",
			valid:    true,
		},
		{
			name:     "path traversal unix",
			input:    "../../../etc/passwd",
			expected: "passwd",
			valid:    true,
		},
		{
			name:     "path traversal with subdirectory",
			input:    "subdir/../../../etc/passwd",
			expected: "passwd",
			valid:    true,
		},
		{
			name:     "absolute path unix",
			input:    "/etc/passwd",
			expected: "passwd",
			valid:    true,
		},
		{
			name:     "filename with spaces",
			input:    "my document.pdf",
			expected: "my document.pdf",
			valid:    true,
		},
		{
			name:     "empty filename",
			input:    "",
			expected: ".",
			valid:    false,
		},
		{
			name:     "single dot",
			input:    ".",
			expected: ".",
			valid:    false,
		},
		{
			name:     "double dot",
			input:    "..",
			expected: "..",
			valid:    false,
		},
		{
			name:     "trailing slash",
			input:    "folder/",
			expected: "folder",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filepath.Base(tt.input)
			if result != tt.expected {
				t.Errorf("filepath.Base(%q) = %q, want %q", tt.input, result, tt.expected)
			}

			// Validate that our invalid filename check works
			isValid := result != "" && result != "." && result != ".."
			if isValid != tt.valid {
				t.Errorf("validity check for %q: got %v, want %v", tt.input, isValid, tt.valid)
			}
		})
	}
}

// mockDownloadServer creates a test server that handles GetAttachment and DownloadAttachment requests
func mockDownloadServer(t *testing.T, attachmentID, filename string, content string, getStatus, downloadStatus int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/attachments/"+attachmentID) && !strings.Contains(r.URL.Path, "/download"):
			// GetAttachment
			w.WriteHeader(getStatus)
			if getStatus == http.StatusOK {
				fmt.Fprintf(w, `{"id": "%s", "title": "%s", "mediaType": "application/octet-stream", "fileSize": %d}`, attachmentID, filename, len(content))
			} else {
				w.Write([]byte(`{"message": "Attachment not found"}`))
			}
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/download"):
			// DownloadAttachment - return redirect first, then content
			if downloadStatus == http.StatusOK {
				w.Header().Set("Content-Type", "application/octet-stream")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(content))
			} else {
				w.WriteHeader(downloadStatus)
				w.Write([]byte(`{"message": "Download failed"}`))
			}
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestRunDownload_Success(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "downloaded.txt")
	fileContent := "test file content"

	server := mockDownloadServer(t, "att123", "test.txt", fileContent, http.StatusOK, http.StatusOK)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &downloadOptions{
		outputFile: outputFile,
		noColor:    true,
	}

	err := runDownload("att123", opts, client)
	require.NoError(t, err)

	// Verify file was written
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Equal(t, fileContent, string(data))
}

func TestRunDownload_UsesAttachmentFilename(t *testing.T) {
	tmpDir := t.TempDir()
	// Change to tmpDir so the file is created there
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(tmpDir)

	fileContent := "test content"

	server := mockDownloadServer(t, "att123", "original.txt", fileContent, http.StatusOK, http.StatusOK)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &downloadOptions{
		outputFile: "", // Use attachment filename
		noColor:    true,
	}

	err := runDownload("att123", opts, client)
	require.NoError(t, err)

	// Verify file was written with attachment filename
	data, err := os.ReadFile("original.txt")
	require.NoError(t, err)
	assert.Equal(t, fileContent, string(data))
}

func TestRunDownload_NotFound(t *testing.T) {
	server := mockDownloadServer(t, "att123", "test.txt", "", http.StatusNotFound, http.StatusOK)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &downloadOptions{
		noColor: true,
	}

	err := runDownload("att123", opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get attachment info")
}

func TestRunDownload_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "existing.txt")

	// Create existing file
	err := os.WriteFile(outputFile, []byte("existing content"), 0644)
	require.NoError(t, err)

	server := mockDownloadServer(t, "att123", "test.txt", "new content", http.StatusOK, http.StatusOK)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &downloadOptions{
		outputFile: outputFile,
		force:      false,
		noColor:    true,
	}

	err = runDownload("att123", opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file already exists")
	assert.Contains(t, err.Error(), "use --force to overwrite")
}

func TestRunDownload_ForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "existing.txt")

	// Create existing file
	err := os.WriteFile(outputFile, []byte("old content"), 0644)
	require.NoError(t, err)

	newContent := "new content"
	server := mockDownloadServer(t, "att123", "test.txt", newContent, http.StatusOK, http.StatusOK)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &downloadOptions{
		outputFile: outputFile,
		force:      true,
		noColor:    true,
	}

	err = runDownload("att123", opts, client)
	require.NoError(t, err)

	// Verify file was overwritten
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Equal(t, newContent, string(data))
}

func TestRunDownload_DownloadFailed(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	server := mockDownloadServer(t, "att123", "test.txt", "", http.StatusOK, http.StatusInternalServerError)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &downloadOptions{
		outputFile: outputFile,
		noColor:    true,
	}

	err := runDownload("att123", opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download attachment")
}
