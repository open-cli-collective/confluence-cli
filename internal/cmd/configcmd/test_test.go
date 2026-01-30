package configcmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/confluence-cli/internal/config"
)

func testConfig(serverURL string) *config.Config {
	return &config.Config{
		URL:      serverURL,
		Email:    "test@example.com",
		APIToken: "test-token",
	}
}

func TestRunTest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	err := runTest(true, nil, testConfig(server.URL))
	require.NoError(t, err)
}

func TestRunTest_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message": "Unauthorized"}`))
	}))
	defer server.Close()

	err := runTest(true, nil, testConfig(server.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestRunTest_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	err := runTest(true, nil, testConfig(server.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

func TestRunTest_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := runTest(true, nil, testConfig(server.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 500")
}
