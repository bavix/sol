package network

import (
	"errors"
	"net"
)

var (
	ErrNoMACAddress   = errors.New("interface has no MAC address")
	ErrNoSuitableIPv4 = errors.New("no suitable IPv4 address found on interface")
)

type InterfaceResolver struct{}

func NewInterfaceResolver() *InterfaceResolver {
	return &InterfaceResolver{}
}

func (r *InterfaceResolver) Resolve(name string) (net.IP, net.HardwareAddr, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, nil, err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, nil, err
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		ip := ipNet.IP.To4()
		if ip == nil || ip.IsLoopback() {
			continue
		}

		if len(iface.HardwareAddr) == 0 {
			return nil, nil, ErrNoMACAddress
		}

		return ip, iface.HardwareAddr, nil
	}

	return nil, nil, ErrNoSuitableIPv4
}
