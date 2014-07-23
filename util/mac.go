package util

import (
	"crypto/rand"
	"net"
)

// Generate a MAC address.
func GenerateMAC() (net.HardwareAddr, error) {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, err
	}
	buf[0] &= 0xFE // Unicast
	buf[0] |= 0x02 // Locally administered
	return net.HardwareAddr(buf), nil
}
