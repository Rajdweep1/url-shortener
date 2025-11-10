// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// AuthIntegrationTestSuite contains integration tests for authentication
type AuthIntegrationTestSuite struct {
	suite.Suite
	baseURL   string
	apiURL    string
	jwtToken  string
	apiKey    string
	adminKey  string
}

// SetupSuite runs before all tests in the suite
func (suite *AuthIntegrationTestSuite) SetupSuite() {
	suite.baseURL = "http://localhost:8081"
	suite.apiURL = suite.baseURL + "/api/v1"
	
	// Wait for server to be ready
	suite.waitForServer()
}

// waitForServer waits for the server to be ready
func (suite *AuthIntegrationTestSuite) waitForServer() {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(suite.apiURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	suite.T().Fatal("Server did not start within timeout")
}

// TestLogin tests the login functionality
func (suite *AuthIntegrationTestSuite) TestLogin() {
	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
		expectToken    bool
	}{
		{
			name:           "valid admin login",
			username:       "admin",
			password:       "admin",
			expectedStatus: http.StatusOK,
			expectToken:    true,
		},
		{
			name:           "invalid credentials",
			username:       "admin",
			password:       "wrong",
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
		{
			name:           "empty username",
			username:       "",
			password:       "admin",
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			loginData := map[string]string{
				"username": tt.username,
				"password": tt.password,
			}
			
			body, _ := json.Marshal(loginData)
			resp, err := http.Post(suite.apiURL+"/auth/login", "application/json", bytes.NewBuffer(body))
			require.NoError(suite.T(), err)
			defer resp.Body.Close()
			
			assert.Equal(suite.T(), tt.expectedStatus, resp.StatusCode)
			
			if tt.expectToken {
				var response map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(suite.T(), err)
				
				token, ok := response["token"].(string)
				assert.True(suite.T(), ok)
				assert.NotEmpty(suite.T(), token)
				
				// Store token for later tests
				if tt.username == "admin" {
					suite.jwtToken = token
				}
			}
		})
	}
}

// TestTokenValidation tests token validation
func (suite *AuthIntegrationTestSuite) TestTokenValidation() {
	// Ensure we have a valid token
	suite.ensureJWTToken()

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		expectValid    bool
	}{
		{
			name:           "valid JWT token",
			token:          suite.jwtToken,
			expectedStatus: http.StatusOK,
			expectValid:    true,
		},
		{
			name:           "invalid token",
			token:          "invalid-token-123",
			expectedStatus: http.StatusUnauthorized,
			expectValid:    false,
		},
		{
			name:           "empty token",
			token:          "",
			expectedStatus: http.StatusBadRequest,
			expectValid:    false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, _ := http.NewRequest("POST", suite.apiURL+"/auth/validate", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(suite.T(), err)
			defer resp.Body.Close()
			
			assert.Equal(suite.T(), tt.expectedStatus, resp.StatusCode)
			
			if tt.expectValid {
				var response map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(suite.T(), err)
				
				valid, ok := response["valid"].(bool)
				assert.True(suite.T(), ok)
				assert.True(suite.T(), valid)
			}
		})
	}
}

// TestAPIKeyCreation tests API key creation
func (suite *AuthIntegrationTestSuite) TestAPIKeyCreation() {
	suite.ensureJWTToken()

	tests := []struct {
		name           string
		useAuth        bool
		keyName        string
		permissions    []string
		expectedStatus int
		expectAPIKey   bool
	}{
		{
			name:    "create API key with admin token",
			useAuth: true,
			keyName: "Integration Test Key",
			permissions: []string{
				"urls:read",
				"urls:write",
			},
			expectedStatus: http.StatusCreated,
			expectAPIKey:   true,
		},
		{
			name:           "create API key without auth",
			useAuth:        false,
			keyName:        "Unauthorized Key",
			expectedStatus: http.StatusUnauthorized,
			expectAPIKey:   false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			keyData := map[string]interface{}{
				"name":        tt.keyName,
				"permissions": tt.permissions,
			}
			
			body, _ := json.Marshal(keyData)
			req, _ := http.NewRequest("POST", suite.apiURL+"/auth/api-keys", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			if tt.useAuth {
				req.Header.Set("Authorization", "Bearer "+suite.jwtToken)
			}
			
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(suite.T(), err)
			defer resp.Body.Close()
			
			assert.Equal(suite.T(), tt.expectedStatus, resp.StatusCode)
			
			if tt.expectAPIKey {
				var response map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(suite.T(), err)
				
				apiKey, ok := response["api_key"].(string)
				assert.True(suite.T(), ok)
				assert.NotEmpty(suite.T(), apiKey)
				assert.Contains(suite.T(), apiKey, "usk_")
				
				// Store API key for later tests
				suite.apiKey = apiKey
			}
		})
	}
}

// TestProtectedEndpoints tests access to protected endpoints
func (suite *AuthIntegrationTestSuite) TestProtectedEndpoints() {
	suite.ensureJWTToken()
	suite.ensureAPIKey()

	tests := []struct {
		name           string
		endpoint       string
		method         string
		authType       string
		expectedStatus int
	}{
		{
			name:           "access profile with JWT",
			endpoint:       "/auth/profile",
			method:         "GET",
			authType:       "jwt",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "access profile with API key",
			endpoint:       "/auth/profile",
			method:         "GET",
			authType:       "apikey",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "access profile without auth",
			endpoint:       "/auth/profile",
			method:         "GET",
			authType:       "none",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "create URL with JWT",
			endpoint:       "/urls",
			method:         "POST",
			authType:       "jwt",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "create URL with API key",
			endpoint:       "/urls",
			method:         "POST",
			authType:       "apikey",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "create URL without auth",
			endpoint:       "/urls",
			method:         "POST",
			authType:       "none",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			var body []byte
			if tt.method == "POST" && tt.endpoint == "/urls" {
				urlData := map[string]string{
					"original_url": "https://example.com/test-" + fmt.Sprintf("%d", time.Now().UnixNano()),
				}
				body, _ = json.Marshal(urlData)
			}
			
			req, _ := http.NewRequest(tt.method, suite.apiURL+tt.endpoint, bytes.NewBuffer(body))
			if len(body) > 0 {
				req.Header.Set("Content-Type", "application/json")
			}
			
			switch tt.authType {
			case "jwt":
				req.Header.Set("Authorization", "Bearer "+suite.jwtToken)
			case "apikey":
				req.Header.Set("X-API-Key", suite.apiKey)
			}
			
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(suite.T(), err)
			defer resp.Body.Close()
			
			assert.Equal(suite.T(), tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestPublicEndpoints tests that public endpoints are accessible without auth
func (suite *AuthIntegrationTestSuite) TestPublicEndpoints() {
	publicEndpoints := []struct {
		path   string
		method string
	}{
		{"/health", "GET"},
		{"/api/v1/health", "GET"},
		{"/api/v1/auth/login", "POST"},
		{"/api/v1/auth/validate", "POST"},
	}

	for _, endpoint := range publicEndpoints {
		suite.Run(fmt.Sprintf("%s %s", endpoint.method, endpoint.path), func() {
			var body []byte
			if endpoint.method == "POST" {
				body = []byte(`{"test": "data"}`)
			}
			
			req, _ := http.NewRequest(endpoint.method, suite.baseURL+endpoint.path, bytes.NewBuffer(body))
			if len(body) > 0 {
				req.Header.Set("Content-Type", "application/json")
			}
			
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(suite.T(), err)
			defer resp.Body.Close()
			
			// Public endpoints should not return 401 Unauthorized
			assert.NotEqual(suite.T(), http.StatusUnauthorized, resp.StatusCode)
		})
	}
}

// ensureJWTToken ensures we have a valid JWT token
func (suite *AuthIntegrationTestSuite) ensureJWTToken() {
	if suite.jwtToken != "" {
		return
	}
	
	loginData := map[string]string{
		"username": "admin",
		"password": "admin",
	}
	
	body, _ := json.Marshal(loginData)
	resp, err := http.Post(suite.apiURL+"/auth/login", "application/json", bytes.NewBuffer(body))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)
	
	token, ok := response["token"].(string)
	require.True(suite.T(), ok)
	suite.jwtToken = token
}

// ensureAPIKey ensures we have a valid API key
func (suite *AuthIntegrationTestSuite) ensureAPIKey() {
	if suite.apiKey != "" {
		return
	}
	
	suite.ensureJWTToken()
	
	keyData := map[string]interface{}{
		"name": "Integration Test API Key",
		"permissions": []string{
			"urls:read",
			"urls:write",
		},
	}
	
	body, _ := json.Marshal(keyData)
	req, _ := http.NewRequest("POST", suite.apiURL+"/auth/api-keys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.jwtToken)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)
	
	apiKey, ok := response["api_key"].(string)
	require.True(suite.T(), ok)
	suite.apiKey = apiKey
}

// TestAuthIntegrationTestSuite runs the integration test suite
func TestAuthIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AuthIntegrationTestSuite))
}
