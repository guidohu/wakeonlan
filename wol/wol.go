package wol

import (
	"fmt"
	"net"
	"strings"
)

func ParseMAC(mac string) ([]byte, error) {
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

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	return err
}
