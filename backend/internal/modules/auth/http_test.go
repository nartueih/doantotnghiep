package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLoginThenGetCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, tokens := newTestService(t, StatusActive)
	handler := NewHTTPHandler(service, tokens)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))

	loginResponse := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		bytes.NewBufferString(`{"email":"admin@example.com","password":"correct-password"}`),
	)
	loginRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d: %s", loginResponse.Code, loginResponse.Body.String())
	}

	var loginBody struct {
		Tokens TokenPair `json:"tokens"`
	}
	if err := json.Unmarshal(loginResponse.Body.Bytes(), &loginBody); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	meResponse := httptest.NewRecorder()
	meRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meRequest.Header.Set("Authorization", "Bearer "+loginBody.Tokens.AccessToken)
	router.ServeHTTP(meResponse, meRequest)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("expected me status 200, got %d: %s", meResponse.Code, meResponse.Body.String())
	}
}

func TestCurrentUserRequiresAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, tokens := newTestService(t, StatusActive)
	handler := NewHTTPHandler(service, tokens)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
}
