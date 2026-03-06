package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"wakeonlan/config"
	"wakeonlan/wol"
)

func HandleHosts(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		config.HostsMu.Lock()
		defer config.HostsMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config.Hosts)
		return
	}

	if r.Method == http.MethodPost {
		var newHost config.Host
		if err := json.NewDecoder(r.Body).Decode(&newHost); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if err := config.ValidateHost(&newHost); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		newHost.ID = fmt.Sprintf("%d", time.Now().UnixNano())

		config.HostsMu.Lock()
		config.Hosts = append(config.Hosts, newHost)
		config.SaveHosts()
		config.HostsMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newHost)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func HandleHostDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/hosts/")
	if id == "" {
		http.Error(w, "Missing host ID", http.StatusBadRequest)
		return
	}

	config.HostsMu.Lock()
	defer config.HostsMu.Unlock()

	var newHosts []config.Host
	for _, h := range config.Hosts {
		if h.ID != id {
			newHosts = append(newHosts, h)
		}
	}
	config.Hosts = newHosts
	config.SaveHosts()

	w.WriteHeader(http.StatusOK)
}

func HandleHostWake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	id := parts[3]

	config.HostsMu.Lock()
	var target config.Host
	found := false
	for _, h := range config.Hosts {
		if h.ID == id {
			target = h
			found = true
			break
		}
	}
	config.HostsMu.Unlock()

	if !found {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	if err := wol.SendWOL(target.MACAddress, target.BroadcastIP); err != nil {
		log.Printf("Failed to wake host %s: %v", target.Name, err)
		http.Error(w, fmt.Sprintf("Failed to send WOL: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "woken"}`))
}

func HandleHostPing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	id := parts[3]

	config.HostsMu.Lock()
	var target config.Host
	found := false
	for _, h := range config.Hosts {
		if h.ID == id {
			target = h
			found = true
			break
		}
	}
	config.HostsMu.Unlock()

	if !found {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if target.IP == "" {
		w.Write([]byte(`{"success": false, "error": "No IP configured"}`))
		return
	}
	if !target.PingEnabled {
		w.Write([]byte(`{"success": false, "error": "Ping disabled"}`))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-c", "1", target.IP)
	err := cmd.Run()

	if err != nil {
		w.Write([]byte(`{"success": false}`))
		return
	}

	w.Write([]byte(`{"success": true}`))
}

func HandleHostEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/hosts/")
	if id == "" {
		http.Error(w, "Missing host ID", http.StatusBadRequest)
		return
	}

	var updateData config.Host
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := config.ValidateHost(&updateData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config.HostsMu.Lock()
	defer config.HostsMu.Unlock()

	found := false
	for i, h := range config.Hosts {
		if h.ID == id {
			config.Hosts[i].Name = updateData.Name
			config.Hosts[i].MACAddress = updateData.MACAddress
			config.Hosts[i].BroadcastIP = updateData.BroadcastIP
			config.Hosts[i].IP = updateData.IP
			config.Hosts[i].AccessURL = updateData.AccessURL
			config.Hosts[i].PingEnabled = updateData.PingEnabled
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	config.SaveHosts()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "updated"}`))
}
