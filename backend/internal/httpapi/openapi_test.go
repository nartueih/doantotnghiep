package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAPISpecIsValidJSONAndContainsEveryAPIArea(t *testing.T) {
	if err := rejectDuplicateJSONKeys(json.NewDecoder(bytes.NewReader(openAPISpec))); err != nil {
		t.Fatalf("openapi document contains duplicate keys: %v", err)
	}
	var document struct {
		OpenAPI string `json:"openapi"`
		Servers []struct {
			URL string `json:"url"`
		} `json:"servers"`
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(openAPISpec, &document); err != nil {
		t.Fatalf("openapi document is not valid JSON: %v", err)
	}
	if document.OpenAPI != "3.0.3" {
		t.Fatalf("expected OpenAPI 3.0.3, got %q", document.OpenAPI)
	}
	if len(document.Servers) != 1 || document.Servers[0].URL != "/" {
		t.Fatalf("openapi server URL must follow the current host")
	}
	for _, path := range []string{
		"/health/live",
		"/health/ready",
		"/api/v1/",
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
		"/api/v1/auth/logout",
		"/api/v1/auth/me",
		"/api/v1/users",
		"/api/v1/users/{id}/status",
		"/api/v1/departments",
		"/api/v1/departments/{id}",
		"/api/v1/software",
		"/api/v1/software/{id}",
		"/api/v1/licenses",
		"/api/v1/licenses/{id}",
		"/api/v1/licenses/{id}/key",
		"/api/v1/licenses/{id}/archive",
		"/api/v1/devices",
		"/api/v1/devices/{id}",
		"/api/v1/devices/{id}/status",
		"/api/v1/devices/{id}/assignment",
		"/api/v1/license-assignments",
		"/api/v1/license-assignments/{id}/revoke",
		"/api/v1/audit-logs",
		"/api/v1/dashboard/summary",
		"/api/v1/dashboard/license-alerts",
		"/api/v1/me/devices",
		"/api/v1/me/licenses",
	} {
		if _, exists := document.Paths[path]; !exists {
			t.Fatalf("openapi document is missing path %s", path)
		}
	}
}

func rejectDuplicateJSONKeys(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicated mapping key %q", key)
			}
			seen[key] = struct{}{}
			if err := rejectDuplicateJSONKeys(decoder); err != nil {
				return fmt.Errorf("at key %q: %w", key, err)
			}
		}
	case '[':
		for decoder.More() {
			if err := rejectDuplicateJSONKeys(decoder); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected delimiter %q", delimiter)
	}
	_, err = decoder.Token()
	return err
}

func TestOpenAPISpecHasUniqueOperationIDsAndResolvableReferences(t *testing.T) {
	var document map[string]any
	if err := json.Unmarshal(openAPISpec, &document); err != nil {
		t.Fatalf("decode openapi document: %v", err)
	}
	operationIDs := make(map[string]string)
	paths := document["paths"].(map[string]any)
	for path, rawPathItem := range paths {
		pathItem := rawPathItem.(map[string]any)
		for method, rawOperation := range pathItem {
			operation, ok := rawOperation.(map[string]any)
			if !ok {
				continue
			}
			operationID, _ := operation["operationId"].(string)
			if operationID == "" {
				t.Fatalf("%s %s is missing operationId", method, path)
			}
			if previous, exists := operationIDs[operationID]; exists {
				t.Fatalf("operationId %q is duplicated by %s and %s %s", operationID, previous, method, path)
			}
			operationIDs[operationID] = method + " " + path
		}
	}
	checkOpenAPIReferences(t, document, document)
}

func checkOpenAPIReferences(t *testing.T, root map[string]any, value any) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]any:
		if reference, ok := typed["$ref"].(string); ok {
			if !strings.HasPrefix(reference, "#/") {
				t.Fatalf("external reference is not supported: %s", reference)
			}
			var current any = root
			for _, segment := range strings.Split(strings.TrimPrefix(reference, "#/"), "/") {
				object, ok := current.(map[string]any)
				if !ok {
					t.Fatalf("reference %s traverses a non-object", reference)
				}
				current, ok = object[segment]
				if !ok {
					t.Fatalf("reference does not resolve: %s", reference)
				}
			}
		}
		for _, child := range typed {
			checkOpenAPIReferences(t, root, child)
		}
	case []any:
		for _, child := range typed {
			checkOpenAPIReferences(t, root, child)
		}
	}
}

func TestDocumentationRoutesArePublic(t *testing.T) {
	router := NewRouter(func(context.Context) error { return nil }, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	specResponse := httptest.NewRecorder()
	router.ServeHTTP(specResponse, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if specResponse.Code != http.StatusOK || !strings.Contains(specResponse.Body.String(), `"openapi": "3.0.3"`) {
		t.Fatalf("unexpected spec response %d: %s", specResponse.Code, specResponse.Body.String())
	}
	if specResponse.Header().Get("Cache-Control") != "no-store, max-age=0" {
		t.Fatalf("openapi response must disable caching")
	}

	docsResponse := httptest.NewRecorder()
	router.ServeHTTP(docsResponse, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if docsResponse.Code != http.StatusOK || !strings.Contains(docsResponse.Body.String(), "/openapi.json") {
		t.Fatalf("unexpected docs response %d: %s", docsResponse.Code, docsResponse.Body.String())
	}
	if docsResponse.Header().Get("Cache-Control") != "no-store, max-age=0" {
		t.Fatalf("docs response must disable caching")
	}
}
