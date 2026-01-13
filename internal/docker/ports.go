package docker

import (
	"fmt"
	"net"
)

// PortConflict represents a potential port conflict
type PortConflict struct {
	Port       int
	Service    string
	Reason     string
	Suggestion int
}

// CheckPortConflicts checks for potential port conflicts
func CheckPortConflicts(ports []PortMapping, offset int) []PortConflict {
	var conflicts []PortConflict

	for _, pm := range ports {
		newPort := pm.HostPort + offset

		// Check if port is in use
		if IsPortInUse(newPort) {
			conflicts = append(conflicts, PortConflict{
				Port:       newPort,
				Service:    pm.ServiceName,
				Reason:     "Port is currently in use",
				Suggestion: SuggestAlternativePort(pm.HostPort, offset),
			})
			continue
		}

		// Check for common port conflicts
		if desc, ok := commonPorts[newPort]; ok {
			conflicts = append(conflicts, PortConflict{
				Port:       newPort,
				Service:    pm.ServiceName,
				Reason:     fmt.Sprintf("Commonly used by %s", desc),
				Suggestion: newPort + 1,
			})
		}
	}

	return conflicts
}

// IsPortInUse checks if a port is currently in use
func IsPortInUse(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return true
	}
	listener.Close()
	return false
}

// SuggestAlternativePort suggests an available port
func SuggestAlternativePort(basePort, offset int) int {
	// Try incrementing from the offset port
	startPort := basePort + offset + 1
	for port := startPort; port < startPort+100; port++ {
		if !IsPortInUse(port) {
			return port
		}
	}
	return startPort
}
