package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLiveHealth(t *testing.T) {
	router := NewRouter(func(context.Context) error { return nil }, nil, nil, nil, nil, nil, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}

func TestReadyHealthWhenDatabaseIsUnavailable(t *testing.T) {
	router := NewRouter(func(context.Context) error { return errors.New("database unavailable") }, nil, nil, nil, nil, nil, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, response.Code)
	}
}
