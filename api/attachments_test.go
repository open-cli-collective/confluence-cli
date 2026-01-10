package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ListAttachments(t *testing.T) {
	testData := loadTestData(t, "attachments.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/pages/98765/attachments", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	result, err := client.ListAttachments(context.Background(), "98765", nil)

	require.NoError(t, err)
	assert.Len(t, result.Results, 2)

	// Check first attachment
	att := result.Results[0]
	assert.Equal(t, "att111", att.ID)
	assert.Equal(t, "screenshot.png", att.Title)
	assert.Equal(t, "image/png", att.MediaType)
	assert.Equal(t, int64(245678), att.FileSize)
}

func TestClient_ListAttachments_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		assert.Equal(t, "image/png", r.URL.Query().Get("mediaType"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	opts := &ListAttachmentsOptions{
		Limit:     50,
		MediaType: "image/png",
	}
	_, err := client.ListAttachments(context.Background(), "98765", opts)
	require.NoError(t, err)
}

func TestClient_GetAttachment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/attachments/att111", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "att111",
			"title": "screenshot.png",
			"mediaType": "image/png",
			"fileSize": 245678,
			"downloadLink": "/download/attachments/98765/screenshot.png"
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	att, err := client.GetAttachment(context.Background(), "att111")

	require.NoError(t, err)
	assert.Equal(t, "att111", att.ID)
	assert.Equal(t, "screenshot.png", att.Title)
	assert.Equal(t, int64(245678), att.FileSize)
}

func TestClient_DownloadAttachment(t *testing.T) {
	fileContent := []byte("fake image content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/attachments/att111/download" {
			// Return redirect
			w.Header().Set("Location", "/download/attachments/98765/screenshot.png")
			w.WriteHeader(http.StatusFound)
			return
		}

		if r.URL.Path == "/download/attachments/98765/screenshot.png" {
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fileContent)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	reader, err := client.DownloadAttachment(context.Background(), "att111")
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	// Read and verify content
	buf := make([]byte, 100)
	n, _ := reader.Read(buf)
	assert.Equal(t, fileContent, buf[:n])
}

func TestClient_DeleteAttachment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/attachments/att111", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	err := client.DeleteAttachment(context.Background(), "att111")
	require.NoError(t, err)
}

func TestClient_DeleteAttachment_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Attachment not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	err := client.DeleteAttachment(context.Background(), "invalid")
	require.Error(t, err)
}

func TestClient_UploadAttachment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/content/12345/child/attachment", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "nocheck", r.Header.Get("X-Atlassian-Token"))
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)

		// Verify file is present
		file, header, err := r.FormFile("file")
		require.NoError(t, err)
		defer func() { _ = file.Close() }()
		assert.Equal(t, "test.txt", header.Filename)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [{
				"id": "att123",
				"title": "test.txt",
				"mediaType": "text/plain",
				"fileSize": 12
			}]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	content := strings.NewReader("test content")
	att, err := client.UploadAttachment(context.Background(), "12345", "test.txt", content, "")

	require.NoError(t, err)
	assert.Equal(t, "att123", att.ID)
	assert.Equal(t, "test.txt", att.Title)
	assert.Equal(t, int64(12), att.FileSize)
}

func TestClient_UploadAttachment_WithComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)

		// Verify comment field is present
		comment := r.FormValue("comment")
		assert.Equal(t, "This is a test attachment", comment)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [{
				"id": "att123",
				"title": "test.txt",
				"mediaType": "text/plain",
				"fileSize": 12
			}]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	content := strings.NewReader("test content")
	att, err := client.UploadAttachment(context.Background(), "12345", "test.txt", content, "This is a test attachment")

	require.NoError(t, err)
	assert.Equal(t, "att123", att.ID)
}

func TestClient_UploadAttachment_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message": "Permission denied"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	content := strings.NewReader("test content")
	att, err := client.UploadAttachment(context.Background(), "12345", "test.txt", content, "")

	require.Error(t, err)
	assert.Nil(t, att)
	assert.Contains(t, err.Error(), "Permission denied")
}

func TestClient_UploadAttachment_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	content := strings.NewReader("test content")
	att, err := client.UploadAttachment(context.Background(), "12345", "test.txt", content, "")

	require.Error(t, err)
	assert.Nil(t, att)
	assert.Contains(t, err.Error(), "no attachment returned from upload")
}
