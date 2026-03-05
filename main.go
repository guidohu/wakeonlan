package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Host struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MACAddress  string `json:"mac_address"`
	BroadcastIP string `json:"broadcast_ip"`
	IP          string `json:"ip"`
	AccessURL   string `json:"access_url,omitempty"`
	PingEnabled bool   `json:"ping_enabled"`
}

var (
	hosts     []Host
	hostsFile = "hosts.json"
	hostsMu   sync.Mutex
)

func loadHosts() {
	hostsMu.Lock()
	defer hostsMu.Unlock()

	log.Printf("Attempting to load hosts from file: %s", hostsFile)

	data, err := os.ReadFile(hostsFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Hosts file does not exist at %s, starting with empty list", hostsFile)
			hosts = []Host{}
			return
		}
		log.Printf("Error reading hosts file: %v", err)
		hosts = []Host{}
		return
	}

	if err := json.Unmarshal(data, &hosts); err != nil {
		log.Printf("Error unmarshaling hosts: %v", err)
		hosts = []Host{}
	}
}

func saveHosts() error {
	data, err := json.MarshalIndent(hosts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(hostsFile, data, 0644)
}

func parseMAC(mac string) ([]byte, error) {
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")
	mac = strings.ReplaceAll(mac, " ", "")

	if len(mac) != 12 {
		return nil, fmt.Errorf("invalid MAC address length")
	}

	var parsed []byte
	for i := 0; i < len(mac); i += 2 {
		var b byte
		_, err := fmt.Sscanf(mac[i:i+2], "%02x", &b)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, b)
	}
	return parsed, nil
}

func validateHost(host *Host) error {
	host.Name = strings.TrimSpace(host.Name)
	host.MACAddress = strings.TrimSpace(host.MACAddress)
	host.BroadcastIP = strings.TrimSpace(host.BroadcastIP)
	host.IP = strings.TrimSpace(host.IP)
	host.AccessURL = strings.TrimSpace(host.AccessURL)

	if host.MACAddress == "" {
		return fmt.Errorf("MAC Address is a mandatory field")
	}
	if _, err := parseMAC(host.MACAddress); err != nil {
		return fmt.Errorf("Invalid MAC Address")
	}

	if host.BroadcastIP == "" {
		host.BroadcastIP = "255.255.255.255"
	} else if net.ParseIP(host.BroadcastIP) == nil {
		return fmt.Errorf("Invalid Broadcast IP")
	}

	if host.IP != "" && net.ParseIP(host.IP) == nil {
		return fmt.Errorf("Invalid Host IP")
	}

	if host.AccessURL != "" {
		u, err := url.ParseRequestURI(host.AccessURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("Invalid Access URL")
		}
	}

	// Name JSON escaping is natively guaranteed by Go's json package during Marshal.
	// We can ensure valid UTF-8.
	host.Name = strings.ToValidUTF8(host.Name, "")

	return nil
}

func sendWOL(mac string, broadcastIP string) error {
	parsedMAC, err := parseMAC(mac)
	if err != nil {
		return err
	}

	if broadcastIP == "" {
		broadcastIP = "255.255.255.255"
	}

	var packet []byte
	for i := 0; i < 6; i++ {
		packet = append(packet, 0xff)
	}
	for i := 0; i < 16; i++ {
		packet = append(packet, parsedMAC...)
	}

	addr, err := net.ResolveUDPAddr("udp", broadcastIP+":9")
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	return err
}

func handleHosts(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		hostsMu.Lock()
		defer hostsMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(hosts)
		return
	}

	if r.Method == http.MethodPost {
		var newHost Host
		if err := json.NewDecoder(r.Body).Decode(&newHost); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if err := validateHost(&newHost); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		newHost.ID = fmt.Sprintf("%d", time.Now().UnixNano())

		hostsMu.Lock()
		hosts = append(hosts, newHost)
		saveHosts()
		hostsMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newHost)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleHostDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/hosts/")
	if id == "" {
		http.Error(w, "Missing host ID", http.StatusBadRequest)
		return
	}

	hostsMu.Lock()
	defer hostsMu.Unlock()

	var newHosts []Host
	for _, h := range hosts {
		if h.ID != id {
			newHosts = append(newHosts, h)
		}
	}
	hosts = newHosts
	saveHosts()

	w.WriteHeader(http.StatusOK)
}

func handleHostWake(w http.ResponseWriter, r *http.Request) {
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

	hostsMu.Lock()
	var target Host
	found := false
	for _, h := range hosts {
		if h.ID == id {
			target = h
			found = true
			break
		}
	}
	hostsMu.Unlock()

	if !found {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	if err := sendWOL(target.MACAddress, target.BroadcastIP); err != nil {
		log.Printf("Failed to wake host %s: %v", target.Name, err)
		http.Error(w, fmt.Sprintf("Failed to send WOL: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "woken"}`))
}

func handleHostPing(w http.ResponseWriter, r *http.Request) {
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

	hostsMu.Lock()
	var target Host
	found := false
	for _, h := range hosts {
		if h.ID == id {
			target = h
			found = true
			break
		}
	}
	hostsMu.Unlock()

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

func handleHostEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/hosts/")
	if id == "" {
		http.Error(w, "Missing host ID", http.StatusBadRequest)
		return
	}

	var updateData Host
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := validateHost(&updateData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hostsMu.Lock()
	defer hostsMu.Unlock()

	found := false
	for i, h := range hosts {
		if h.ID == id {
			hosts[i].Name = updateData.Name
			hosts[i].MACAddress = updateData.MACAddress
			hosts[i].BroadcastIP = updateData.BroadcastIP
			hosts[i].IP = updateData.IP
			hosts[i].AccessURL = updateData.AccessURL
			hosts[i].PingEnabled = updateData.PingEnabled
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	saveHosts()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "updated"}`))
}

func main() {
	if envFile := os.Getenv("HOSTS_FILE"); envFile != "" {
		hostsFile = envFile
	}
	loadHosts()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/api/hosts", handleHosts)
	http.HandleFunc("/api/hosts/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/wake") {
			handleHostWake(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/ping") {
			handleHostPing(w, r)
		} else {
			if r.Method == http.MethodDelete {
				handleHostDelete(w, r)
			} else if r.Method == http.MethodPut {
				handleHostEdit(w, r)
			} else {
				http.Error(w, "Not found", http.StatusNotFound)
			}
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on http://0.0.0.0:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
