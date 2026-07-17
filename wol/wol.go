package wol

import (
	"encoding/hex"
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

	// hex.DecodeString is significantly faster than using fmt.Sscanf in a loop
	return hex.DecodeString(mac)
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
