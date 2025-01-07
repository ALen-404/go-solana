package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockQuoteResponse is a mock struct for testing purposes.
type MockQuoteResponse struct {
	Quote string `json:"quote"`
}

// TestGetQuote tests the getQuote method of WalletManager.
func TestGetQuote(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		expectedError  string
		expectedQuote  *QuoteResponse
	}{
		{
			name:           "HTTP Request Failure",
			serverResponse: "",
			statusCode:     http.StatusOK,
			expectedError:  "Get \"\": unsupported protocol scheme \"\"",
		},
		{
			name:           "Non-OK Status Code",
			serverResponse: "Error: Not Found",
			statusCode:     http.StatusNotFound,
			expectedError:  "quote failed: Error: Not Found",
		},
		{
			name:           "JSON Decoding Failure",
			serverResponse: "Invalid JSON",
			statusCode:     http.StatusOK,
			expectedError:  "invalid character 'I' looking for beginning of value",
		},
		{
			name:           "Successful Quote Retrieval",
			serverResponse: `{"quote": "Test Quote"}`,
			statusCode:     http.StatusOK,
			expectedQuote: &QuoteResponse{
				Quote: "Test Quote",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.statusCode)
				w.Write([]byte(test.serverResponse))
			}))
			defer server.Close()

			wm := &WalletManager{}
			quote, err := wm.getQuote(server.URL)

			if test.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedQuote, quote)
			}
		})
	}
}
