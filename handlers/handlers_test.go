package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"wakeonlan/config"
	"wakeonlan/handlers"
)

func resetConfigHosts() {
	config.HostsMu.Lock()
	defer config.HostsMu.Unlock()
	config.Hosts = []config.Host{}
}

func TestHandleHosts_GET(t *testing.T) {
	resetConfigHosts()

	// Add some dummy hosts
	expectedHosts := []config.Host{
		{ID: "1", Name: "Host 1", MACAddress: "aa:bb:cc:dd:ee:11", IP: "10.0.0.1"},
		{ID: "2", Name: "Host 2", MACAddress: "aa:bb:cc:dd:ee:22", IP: "10.0.0.2"},
	}
	config.Hosts = expectedHosts

	req, err := http.NewRequest("GET", "/api/hosts", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleHosts)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var returnedHosts []config.Host
	if err := json.Unmarshal(rr.Body.Bytes(), &returnedHosts); err != nil {
		t.Fatalf("Failed to parse response body as JSON: %v", err)
	}

	if len(returnedHosts) != 2 {
		t.Errorf("Expected 2 hosts, got %d", len(returnedHosts))
	}
}

func TestHandleHosts_POST(t *testing.T) {
	resetConfigHosts()

	newHost := config.Host{
		Name:       "New Test Host",
		MACAddress: "11:22:33:44:55:66",
		IP:         "192.168.1.50",
	}
	body, _ := json.Marshal(newHost)

	req, err := http.NewRequest("POST", "/api/hosts", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleHosts)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v | body: %s",
			status, http.StatusCreated, rr.Body.String())
	}

	var returnedHost config.Host
	if err := json.Unmarshal(rr.Body.Bytes(), &returnedHost); err != nil {
		t.Fatalf("Failed to parse response body as JSON: %v", err)
	}

	if returnedHost.Name != newHost.Name {
		t.Errorf("Expected name %s, got %s", newHost.Name, returnedHost.Name)
	}
	if returnedHost.ID == "" {
		t.Errorf("Expected ID to be set, but it was empty")
	}

	// Verify it was added to the config (length should be 1)
	if len(config.Hosts) != 1 {
		t.Errorf("Expected 1 host in config, got %d", len(config.Hosts))
	}
}

func TestHandleHostDelete(t *testing.T) {
	resetConfigHosts()
	config.Hosts = []config.Host{
		{ID: "keep-me", Name: "Keep"},
		{ID: "delete-me", Name: "Delete"},
	}

	req, err := http.NewRequest("DELETE", "/api/hosts/delete-me", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleHostDelete)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if len(config.Hosts) != 1 {
		t.Errorf("Expected 1 host remaining, got %d", len(config.Hosts))
	}
	if config.Hosts[0].ID != "keep-me" {
		t.Errorf("Wrong host deleted")
	}
}

func TestHandleHostEdit(t *testing.T) {
	resetConfigHosts()
	config.Hosts = []config.Host{
		{ID: "edit-me", Name: "Old Name", MACAddress: "aa:bb:cc:dd:ee:ff"},
	}

	updateData := config.Host{
		Name:       "New Name",
		MACAddress: "11:22:33:44:55:66", // required for validation
	}
	body, _ := json.Marshal(updateData)

	req, err := http.NewRequest("PUT", "/api/hosts/edit-me", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleHostEdit)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v | body: %s",
			status, http.StatusOK, rr.Body.String())
	}

	if len(config.Hosts) != 1 {
		t.Errorf("Expected 1 host, got %d", len(config.Hosts))
	}
	if config.Hosts[0].Name != "New Name" {
		t.Errorf("Expected name to be updated, got %s", config.Hosts[0].Name)
	}
}

func TestHandleHostWake_NotFound(t *testing.T) {
	resetConfigHosts()

	req, err := http.NewRequest("POST", "/api/hosts/nonexistent/wake", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleHostWake)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}
}

// Ensure the method restriction works for all routes
func TestMethodRestrictions(t *testing.T) {
	tests := []struct {
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{"DELETE", "/api/hosts", handlers.HandleHosts},
		{"GET", "/api/hosts/123", handlers.HandleHostDelete},
		{"GET", "/api/hosts/123/wake", handlers.HandleHostWake},
		{"POST", "/api/hosts/123/ping", handlers.HandleHostPing},
		{"POST", "/api/hosts/123", handlers.HandleHostEdit},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			tt.handler.ServeHTTP(rr, req)

			if !strings.Contains(rr.Body.String(), "Method not allowed") && rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected MethodNotAllowed, got %v", rr.Code)
			}
		})
	}
}
