package config

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"

	"wakeonlan/wol"
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
	Hosts     []Host
	HostsFile = "hosts.json"
	HostsMu   sync.Mutex

	AdminUser     string
	AdminPassword string
	JWTSecret     []byte
)

func init() {
	AdminUser = os.Getenv("ADMIN_USER")
	if AdminUser == "" {
		AdminUser = "admin"
	}
	AdminPassword = os.Getenv("ADMIN_PASSWORD")
	if AdminPassword == "" {
		AdminPassword = "admin"
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			log.Fatalf("failed to generate random JWT secret: %v", err)
		}
		JWTSecret = b
	} else {
		JWTSecret = []byte(secret)
	}
}

func LoadHosts() {
	HostsMu.Lock()
	defer HostsMu.Unlock()

	log.Printf("Attempting to load hosts from file: %s", HostsFile)

	data, err := os.ReadFile(HostsFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Hosts file does not exist at %s, creating an empty one", HostsFile)
			Hosts = []Host{}
			if err := os.WriteFile(HostsFile, []byte("[]\n"), 0644); err != nil {
				log.Printf("Error creating empty hosts file: %v", err)
			}
			return
		}
		log.Printf("Error reading hosts file: %v", err)
		Hosts = []Host{}
		return
	}

	if err := json.Unmarshal(data, &Hosts); err != nil {
		log.Printf("Error unmarshaling hosts: %v", err)
		Hosts = []Host{}
	}
}

func SaveHosts() error {
	data, err := json.MarshalIndent(Hosts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(HostsFile, data, 0644)
}

func ValidateHost(host *Host) error {
	host.Name = strings.TrimSpace(host.Name)
	host.MACAddress = strings.TrimSpace(host.MACAddress)
	host.BroadcastIP = strings.TrimSpace(host.BroadcastIP)
	host.IP = strings.TrimSpace(host.IP)
	host.AccessURL = strings.TrimSpace(host.AccessURL)

	if host.MACAddress == "" {
		return fmt.Errorf("mac address is a mandatory field")
	}
	if _, err := wol.ParseMAC(host.MACAddress); err != nil {
		return fmt.Errorf("invalid mac address")
	}

	if host.BroadcastIP == "" {
		host.BroadcastIP = "255.255.255.255"
	} else if net.ParseIP(host.BroadcastIP) == nil {
		return fmt.Errorf("invalid broadcast IP")
	}

	if host.IP != "" && net.ParseIP(host.IP) == nil {
		return fmt.Errorf("invalid host IP")
	}

	if host.AccessURL != "" {
		u, err := url.ParseRequestURI(host.AccessURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("invalid access URL")
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("invalid access URL scheme: must be http or https")
		}
	}

	host.Name = strings.ToValidUTF8(host.Name, "")

	return nil
}
