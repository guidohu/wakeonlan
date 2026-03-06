package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"wakeonlan/config"
)

func TestValidateHost(t *testing.T) {
	tests := []struct {
		name      string
		host      config.Host
		wantError bool
	}{
		{
			name: "Valid Host",
			host: config.Host{
				Name:       "Test Host",
				MACAddress: "aa:bb:cc:dd:ee:ff",
				IP:         "192.168.1.100",
				AccessURL:  "http://example.com",
			},
			wantError: false,
		},
		{
			name: "Missing MAC",
			host: config.Host{
				Name: "Missing MAC Host",
			},
			wantError: true,
		},
		{
			name: "Invalid MAC",
			host: config.Host{
				Name:       "Invalid MAC Host",
				MACAddress: "invalid-mac",
			},
			wantError: true,
		},
		{
			name: "Invalid Broadcast IP",
			host: config.Host{
				Name:        "Invalid Bcast",
				MACAddress:  "aa:bb:cc:dd:ee:ff",
				BroadcastIP: "256.256.256.256",
			},
			wantError: true,
		},
		{
			name: "Invalid IP",
			host: config.Host{
				Name:       "Invalid IP",
				MACAddress: "aa:bb:cc:dd:ee:ff",
				IP:         "999.999.999.999",
			},
			wantError: true,
		},
		{
			name: "Invalid URL (Not a URL)",
			host: config.Host{
				Name:       "Invalid URL",
				MACAddress: "aa:bb:cc:dd:ee:ff",
				AccessURL:  "not a url",
			},
			wantError: true,
		},
		{
			name: "Invalid URL (XSS javascript:)",
			host: config.Host{
				Name:       "XSS URL",
				MACAddress: "aa:bb:cc:dd:ee:ff",
				AccessURL:  "javascript:alert(1)",
			},
			wantError: true,
		},
		{
			name: "Only MAC (Minimum Required)",
			host: config.Host{
				MACAddress: "aa:bb:cc:dd:ee:ff",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validation might modify the struct (e.g. setting default broadcast IP),
			// so we pass a copy.
			h := tt.host
			err := config.ValidateHost(&h)

			if (err != nil) != tt.wantError {
				t.Fatalf("ValidateHost() error = %v, wantError %v", err, tt.wantError)
			}

			// Verify default BroadcastIP behavior
			if !tt.wantError && tt.host.BroadcastIP == "" {
				if h.BroadcastIP != "255.255.255.255" {
					t.Errorf("Expected BroadcastIP to be set to 255.255.255.255, got %v", h.BroadcastIP)
				}
			}
		})
	}
}

func TestLoadAndSaveHosts(t *testing.T) {
	// Create a temporary directory to avoid messing with the real hosts.json
	tempDir, err := os.MkdirTemp("", "wakeonlan-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // clean up

	tempFile := filepath.Join(tempDir, "hosts.json")

	// Set the package global variable to point to our temp file
	originalFile := config.HostsFile
	config.HostsFile = tempFile
	defer func() { config.HostsFile = originalFile }()

	// Start with a clean slate
	config.Hosts = []config.Host{}

	// Test loading when file doesn't exist
	config.LoadHosts()
	if len(config.Hosts) != 0 {
		t.Errorf("Expected 0 hosts on load without file, got %d", len(config.Hosts))
	}

	// Test saving some hosts
	testHosts := []config.Host{
		{ID: "1", Name: "Host 1", MACAddress: "aa:ll:cc:dd:ee:ff"},
		{ID: "2", Name: "Host 2", MACAddress: "11:22:33:44:55:66"},
	}
	config.Hosts = testHosts

	if err := config.SaveHosts(); err != nil {
		t.Fatalf("Failed to save hosts: %v", err)
	}

	// Test loading the saved hosts
	config.Hosts = []config.Host{} // Clear it
	config.LoadHosts()

	if len(config.Hosts) != 2 {
		t.Fatalf("Expected 2 hosts, got %d", len(config.Hosts))
	}
	if config.Hosts[0].ID != "1" || config.Hosts[1].ID != "2" {
		t.Errorf("Loaded hosts do not match saved hosts")
	}

	// Verify the file directly as well
	data, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read temp file directly: %v", err)
	}

	var parsed []config.Host
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal temp file data: %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("Expected 2 hosts in JSON file, got %d", len(parsed))
	}
}
