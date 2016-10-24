package fingerprint

import (
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/nomad/structs"
)

// NetworkFingerprint is used to fingerprint the Network capabilities of a node
type NetworkFingerprint struct {
	StaticFingerprinter
	logger            *log.Logger
	interfaceDetector NetworkInterfaceDetector
}

// An interface to isolate calls to various api in net package
// This facilitates testing where we can implement
// fake interfaces and addresses to test varios code paths
type NetworkInterfaceDetector interface {
	Interfaces() ([]net.Interface, error)
	InterfaceByName(name string) (*net.Interface, error)
	Addrs(intf *net.Interface) ([]net.Addr, error)
}

// Implements the interface detector which calls net directly
type DefaultNetworkInterfaceDetector struct {
}

func (b *DefaultNetworkInterfaceDetector) Interfaces() ([]net.Interface, error) {
	return net.Interfaces()
}

func (b *DefaultNetworkInterfaceDetector) InterfaceByName(name string) (*net.Interface, error) {
	return net.InterfaceByName(name)
}

func (b *DefaultNetworkInterfaceDetector) Addrs(intf *net.Interface) ([]net.Addr, error) {
	return intf.Addrs()
}

// NewNetworkFingerprint returns a new NetworkFingerprinter with the given
// logger
func NewNetworkFingerprint(logger *log.Logger) Fingerprint {
	f := &NetworkFingerprint{logger: logger, interfaceDetector: &DefaultNetworkInterfaceDetector{}}
	return f
}

func (f *NetworkFingerprint) Fingerprint(cfg *config.Config, node *structs.Node) (bool, error) {
	// newNetwork is populated and addded to the Nodes resources
	newNetwork := &structs.NetworkResource{}
	var ip string

	intf, err := f.findInterface(cfg.NetworkInterface)
	switch {
	case err != nil:
		return false, fmt.Errorf("Error while detecting network interface during fingerprinting: %v", err)
	case intf == nil:
		// No interface could be found
		return false, nil
	}

	if ip, err = f.ipAddress(intf); err != nil {
		return false, fmt.Errorf("Unable to find IP address of interface: %s, err: %v", intf.Name, err)
	}

	newNetwork.Device = intf.Name
	node.Attributes["unique.network.ip-address"] = ip
	newNetwork.IP = ip
	newNetwork.CIDR = newNetwork.IP + "/32"

	f.logger.Printf("[DEBUG] fingerprint.network: Detected interface %v with IP %v during fingerprinting", intf.Name, ip)

	if throughput := f.linkSpeed(intf.Name); throughput > 0 {
		newNetwork.MBits = throughput
		f.logger.Printf("[DEBUG] fingerprint.network: link speed for %v set to %v", intf.Name, newNetwork.MBits)
	} else {
		f.logger.Printf("[DEBUG] fingerprint.network: Unable to read link speed; setting to default %v", cfg.NetworkSpeed)
		newNetwork.MBits = cfg.NetworkSpeed
	}

	if node.Resources == nil {
		node.Resources = &structs.Resources{}
	}

	node.Resources.Networks = append(node.Resources.Networks, newNetwork)

	// return true, because we have a network connection
	return true, nil
}

// Gets the ipv4 addr for a network interface
func (f *NetworkFingerprint) ipAddress(intf *net.Interface) (string, error) {
	var addrs []net.Addr
	var err error

	if addrs, err = f.interfaceDetector.Addrs(intf); err != nil {
		return "", err
	}

	if len(addrs) == 0 {
		return "", errors.New(fmt.Sprintf("Interface %s has no IP address", intf.Name))
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := (addr).(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip.To4() != nil {
			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("Couldn't parse IP address for interface %s", intf.Name)

}

// Checks if the device is marked UP by the operator
func (f *NetworkFingerprint) isDeviceEnabled(intf *net.Interface) bool {
	return intf.Flags&net.FlagUp != 0
}

// Checks if the device has any IP address configured
func (f *NetworkFingerprint) deviceHasIpAddress(intf *net.Interface) bool {
	_, err := f.ipAddress(intf)
	return err == nil
}

func (n *NetworkFingerprint) isDeviceLoopBackOrPointToPoint(intf *net.Interface) bool {
	return intf.Flags&(net.FlagLoopback|net.FlagPointToPoint) != 0
}

// Returns the interface with the name passed by user
// If the name is blank then it iterates through all the devices
// and finds one which is routable and marked as UP
// It excludes PPP and lo devices unless they are specifically asked
func (f *NetworkFingerprint) findInterface(deviceName string) (*net.Interface, error) {
	var interfaces []net.Interface
	var err error

	if deviceName != "" {
		return f.interfaceDetector.InterfaceByName(deviceName)
	}

	var intfs []net.Interface

	if intfs, err = f.interfaceDetector.Interfaces(); err != nil {
		return nil, err
	}

	for _, intf := range intfs {
		if f.isDeviceEnabled(&intf) && !f.isDeviceLoopBackOrPointToPoint(&intf) && f.deviceHasIpAddress(&intf) {
			interfaces = append(interfaces, intf)
		}
	}

	if len(interfaces) == 0 {
		return nil, nil
	}
	return &interfaces[0], nil
}
