package application

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"regexp"
	"strconv"
)

// Wake On LAN Controller
// Taken from sabrhiram's go-wol package

var (
	header        = [6]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	regMAC        = regexp.MustCompile("^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$")
	regIface      = regexp.MustCompile(`^[a-zA-Z0-9:\-]+$`)
	bcastAddr     = "255.255.255.255"
	ErrInvalidMAC = fmt.Errorf("invalid MAC address")
)

type WOL struct {
	PBCID     string `json:"id"`
	Alias     string `json:"alias"`
	Interface string `json:"iface"`
	MAC       string `json:"mac"`
	Port      int    `json:"port"`
	Enabled   bool   `json:"enabled"`
}

// NewWOL creates a new WOL struct.
func NewWOL(pbcid, alias, iface, mac string, port int) (WOL, error) {
	w := WOL{
		PBCID:     pbcid,
		Alias:     alias,
		Interface: iface,
		MAC:       mac,
		Port:      port,
		Enabled:   true,
	}

	if err := w.Validate(); err != nil {
		return WOL{}, err
	}

	return w, nil
}

// Validate checks if the WOL struct is valid.
func (w WOL) Validate() error {
	if !regPBCID.MatchString(w.PBCID) {
		return fmt.Errorf("invalid PBC ID")
	}

	if !regPBCName.MatchString(w.Alias) {
		return fmt.Errorf("invalid alias: allowed characters (min 2, max 32) a-z, A-Z, 0-9, _, -")
	}

	if !regIface.MatchString(w.Interface) {
		return fmt.Errorf("invalid interface name")
	}

	if !regMAC.MatchString(w.MAC) {
		return ErrInvalidMAC
	}

	if w.Port < 0 || w.Port > 65535 {
		return fmt.Errorf("invalid port %d", w.Port)
	}

	return nil
}

// MagicPacket is a struct that represents a Wake On LAN Magic Packet.
type MagicPacket struct {
	Header  [6]byte
	Payload [16][6]byte // Same MACAddr 16 times
}

// NewMagicPacket creates a new MagicPacket.
func NewMagicPacket(mac string) ([]byte, error) {
	// Make sure the MAC address is valid.
	if !regMAC.MatchString(mac) {
		return nil, ErrInvalidMAC
	}

	// Parse the MAC address.
	m, err := net.ParseMAC(mac)
	if err != nil {
		return nil, err
	}

	mp := MagicPacket{
		Header: header,
	}

	// Copy the MAC address 16 times to the payload.
	for i := range mp.Payload {
		copy(mp.Payload[i][:], m[:])
	}

	b, err := mp.Marshal()
	if err != nil {
		return nil, err
	}

	return b, nil
}

// Bytes returns the byte representation of the MagicPacket.
func (mp *MagicPacket) Marshal() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, mp); err != nil {
		return nil, fmt.Errorf("MagicPacket.Marshal() error: %v", err)
	}

	return buf.Bytes(), nil
}

func getAddr(iface string) (*net.UDPAddr, error) {
	if iface == "" {
		return nil, fmt.Errorf("MagicPacket.getAddr() no interface provided")
	}

	i, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, fmt.Errorf("MagicPacket.getAddr() error: %v", err)
	}

	addrs, err := i.Addrs()
	if err != nil {
		return nil, fmt.Errorf("MagicPacket.getAddr() error: %v", err)
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("MagicPacket.getAddr() no addresses found for interface %s", iface)
	}

	for _, addr := range addrs {
		switch ip := addr.(type) {
		case *net.IPNet:
			if !ip.IP.IsLoopback() && ip.IP.To4() != nil {
				return &net.UDPAddr{
					IP:   net.IPv4bcast,
					Port: 9,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("MagicPacket.getAddr() no suitable address found for interface %s", iface)
}

func Wake(iface, dstMAC, port string) error {
	srcAddr, err := getAddr(iface)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	p, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	if p < 0 || p > 65535 {
		return fmt.Errorf("error: invalid port %d", p)
	}

	dstAddr, err := net.ResolveUDPAddr("udp", bcastAddr+":"+port)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	bs, err := NewMagicPacket(dstMAC)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	conn, err := net.DialUDP("udp", srcAddr, dstAddr)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	defer conn.Close()

	n, err := conn.Write(bs)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	if n != len(bs) {
		return fmt.Errorf("error: %v", err)
	}

	return nil
}
