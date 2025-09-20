package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"

	"github.com/bavix/sol/internal/config"
)

const (
	// WoLPortDefault is the default UDP port for Wake-on-LAN.
	WoLPortDefault = 9
	// BufferSize is the size of the UDP receive buffer.
	BufferSize = 2048
	// MagicPacketHeaderSize is the size of the magic packet header (6 bytes of 0xFF).
	MagicPacketHeaderSize = 6
	// MagicPacketRepeatCount is the number of times the MAC address is repeated.
	MagicPacketRepeatCount = 16
	// MacAddressSize is the size of a MAC address in bytes.
	MacAddressSize = 6
)

var (
	// ErrNoMACAddress indicates that the interface has no MAC address.
	ErrNoMACAddress = errors.New("interface has no MAC address")
	// ErrNoSuitableIP indicates that no suitable IPv4 address was found.
	ErrNoSuitableIP = errors.New("no suitable IPv4 address found on interface")
)

// App represents the main application.
type App struct {
	cfg *config.Config
}

// New creates a new application instance.
func New(cfg *config.Config) *App {
	return &App{
		cfg: cfg,
	}
}

// Run starts the application.
func (a *App) Run() error {
	ip, mac, err := getIPv4AndMAC(a.cfg.InterfaceName)
	if err != nil {
		return fmt.Errorf("failed to get IP/MAC for interface %q: %w", a.cfg.InterfaceName, err)
	}

	log.Printf("Using interface %q: IP=%s, MAC=%s", a.cfg.InterfaceName, ip, mac)

	expected := buildMagicPacket(mac)

	// Bind to 0.0.0.0 to reliably receive broadcast packets (255.255.255.255)
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: a.cfg.Port}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind UDP %s:%d: %w", addr.IP.String(), a.cfg.Port, err)
	}
	defer conn.Close()

	log.Printf("Listening on %s:%d (Shutdown-on-LAN)", addr.IP.String(), a.cfg.Port)

	buf := make([]byte, BufferSize)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("read error: %v", err)

			continue
		}

		pkt := buf[:n]
		if isMagicPacket(pkt, expected) {
			log.Printf("Magic packet match from %s — triggering %s", src, ternary(a.cfg.DryRun, "DRY-RUN", "shutdown"))

			if a.cfg.DryRun {
				continue
			}

			if err := shutdown(); err != nil {
				log.Printf("shutdown failed: %v", err)
			}
		} else {
			// Optionally log once in a while or at debug level.
			log.Printf("Non-matching packet from %s, len=%d", src, n)
		}
	}
}

func getIPv4AndMAC(name string) (net.IP, net.HardwareAddr, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, nil, err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, nil, err
	}

	for _, a := range addrs {
		if ipNet, ok := a.(*net.IPNet); ok {
			ip := ipNet.IP.To4()
			if ip != nil && !ip.IsLoopback() {
				if len(iface.HardwareAddr) == 0 {
					return nil, nil, ErrNoMACAddress
				}

				return ip, iface.HardwareAddr, nil
			}
		}
	}

	return nil, nil, ErrNoSuitableIP
}

// buildMagicPacket constructs 6x 0xFF followed by 16x MAC (6 bytes each).
func buildMagicPacket(mac net.HardwareAddr) []byte {
	pkt := make([]byte, MagicPacketHeaderSize+MagicPacketRepeatCount*MacAddressSize)

	for i := range MagicPacketHeaderSize {
		pkt[i] = 0xFF
	}

	o := MagicPacketHeaderSize
	for range MagicPacketRepeatCount {
		copy(pkt[o:o+MacAddressSize], mac)
		o += MacAddressSize
	}

	return pkt
}

// isMagicPacket checks if expected sequence appears contiguously anywhere in the payload.
// This tolerates extra bytes (e.g., padding, SecureOn passwords, or vendor headers).
func isMagicPacket(got []byte, expected []byte) bool {
	return bytes.Contains(got, expected)
}

// ErrUnsupportedOS indicates that the operating system is not supported.
var ErrUnsupportedOS = errors.New("unsupported operating system")

func shutdown() error {
	switch runtime.GOOS {
	case "windows":
		return execCmd("shutdown", "-s", "-t", "0", "-f")
	case "linux", "darwin":
		// On macOS, 'shutdown -h now' also works; may require sudo.
		return execCmd("shutdown", "-h", "now")
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedOS, runtime.GOOS)
	}
}

func execCmd(name string, args ...string) error {
	cmd := exec.CommandContext(context.Background(), name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}

	return b
}
