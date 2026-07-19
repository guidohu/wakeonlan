package wol

import (
	"encoding/hex"
	"fmt"
	"net"
)

// ParseMAC parses a MAC address string into a 6-byte array.
func ParseMAC(mac string) ([]byte, error) {
	var clean [12]byte
	var count int
	for i := 0; i < len(mac); i++ {
		c := mac[i]
		if c == ':' || c == '-' || c == '.' || c == ' ' {
			continue
		}
		if count >= 12 {
			return nil, fmt.Errorf("invalid MAC address length")
		}
		clean[count] = c
		count++
	}
	if count != 12 {
		return nil, fmt.Errorf("invalid MAC address length")
	}

	parsed := make([]byte, 6)
	_, err := hex.Decode(parsed, clean[:])
	if err != nil {
		return nil, err
	}

	return parsed, nil
}

func SendWOL(mac string, broadcastIP string) error {
	parsedMAC, err := ParseMAC(mac)
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

	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.WriteTo(packet, addr)
	return err
}
