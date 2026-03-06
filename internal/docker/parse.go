package docker

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeConfig represents a parsed docker-compose file
type ComposeConfig struct {
	Version  string                  `yaml:"version,omitempty"`
	Services map[string]Service      `yaml:"services"`
	Volumes  map[string]VolumeConfig `yaml:"volumes,omitempty"`
	Networks map[string]interface{}  `yaml:"networks,omitempty"`
}

// Service represents a docker-compose service
type Service struct {
	Image       string                 `yaml:"image,omitempty"`
	Build       interface{}            `yaml:"build,omitempty"`
	Volumes     []string               `yaml:"volumes,omitempty"`
	Ports       []string               `yaml:"ports,omitempty"`
	Environment map[string]interface{} `yaml:"environment,omitempty"`
	DependsOn   interface{}            `yaml:"depends_on,omitempty"`
}

// VolumeConfig represents a named volume configuration
type VolumeConfig struct {
	Name     string            `yaml:"name,omitempty"`
	Driver   string            `yaml:"driver,omitempty"`
	External bool              `yaml:"external,omitempty"`
	Labels   map[string]string `yaml:"labels,omitempty"`
}

// VolumeMount represents a parsed volume mount
type VolumeMount struct {
	Source      string // Volume name or host path
	Target      string // Container path
	IsNamed     bool   // True if named volume, false if bind mount
	IsReadOnly  bool   // True if read-only mount
	ServiceName string // Service that uses this volume
}

// PortMapping represents a parsed port mapping
type PortMapping struct {
	HostPort      int    // Port on host
	ContainerPort int    // Port in container
	Protocol      string // "tcp" or "udp"
	ServiceName   string // Service that exposes this port
}

// ParseComposeFile parses a docker-compose file
func ParseComposeFile(path string) (*ComposeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &DockerError{
			Op:   "read",
			File: path,
			Err:  err,
		}
	}

	var config ComposeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, &DockerError{
			Op:   "parse",
			File: path,
			Err:  ErrParseError,
		}
	}

	return &config, nil
}

// ParseComposeFiles parses multiple compose files and merges them
// Later files override earlier ones (standard docker-compose behavior)
func ParseComposeFiles(paths []string) (*ComposeConfig, error) {
	if len(paths) == 0 {
		return nil, ErrNoComposeFile
	}

	// Parse first file as base
	config, err := ParseComposeFile(paths[0])
	if err != nil {
		return nil, err
	}

	// Merge remaining files
	for _, path := range paths[1:] {
		override, err := ParseComposeFile(path)
		if err != nil {
			return nil, err
		}
		config = mergeConfigs(config, override)
	}

	return config, nil
}

// mergeConfigs merges two compose configs (override takes precedence)
func mergeConfigs(base, override *ComposeConfig) *ComposeConfig {
	merged := &ComposeConfig{
		Version:  base.Version,
		Services: make(map[string]Service),
		Volumes:  make(map[string]VolumeConfig),
		Networks: make(map[string]interface{}),
	}

	// Copy base services
	for name, svc := range base.Services {
		merged.Services[name] = svc
	}

	// Merge override services
	for name, overrideSvc := range override.Services {
		if baseSvc, exists := merged.Services[name]; exists {
			merged.Services[name] = mergeServices(baseSvc, overrideSvc)
		} else {
			merged.Services[name] = overrideSvc
		}
	}

	// Copy base volumes
	for name, vol := range base.Volumes {
		merged.Volumes[name] = vol
	}

	// Override volumes
	for name, vol := range override.Volumes {
		merged.Volumes[name] = vol
	}

	// Copy base networks
	for name, net := range base.Networks {
		merged.Networks[name] = net
	}

	// Override networks
	for name, net := range override.Networks {
		merged.Networks[name] = net
	}

	return merged
}

// mergeServices merges two service definitions
func mergeServices(base, override Service) Service {
	merged := base

	if override.Image != "" {
		merged.Image = override.Image
	}
	if override.Build != nil {
		merged.Build = override.Build
	}
	if len(override.Volumes) > 0 {
		merged.Volumes = override.Volumes
	}
	if len(override.Ports) > 0 {
		merged.Ports = override.Ports
	}
	if override.Environment != nil {
		if merged.Environment == nil {
			merged.Environment = make(map[string]interface{})
		}
		for k, v := range override.Environment {
			merged.Environment[k] = v
		}
	}
	if override.DependsOn != nil {
		merged.DependsOn = override.DependsOn
	}

	return merged
}

// ExtractVolumes extracts all volume mounts from a compose config
func ExtractVolumes(config *ComposeConfig) []VolumeMount {
	var mounts []VolumeMount

	for serviceName, service := range config.Services {
		for _, volStr := range service.Volumes {
			mount := parseVolumeMount(volStr, serviceName)
			mounts = append(mounts, mount)
		}
	}

	return mounts
}

// ExtractPorts extracts all port mappings from a compose config
func ExtractPorts(config *ComposeConfig) []PortMapping {
	var mappings []PortMapping

	for serviceName, service := range config.Services {
		for _, portStr := range service.Ports {
			mapping := parsePortMapping(portStr, serviceName)
			mappings = append(mappings, mapping)
		}
	}

	return mappings
}

// ExtractNamedVolumes returns only named volumes (not bind mounts)
func ExtractNamedVolumes(config *ComposeConfig) []string {
	volumeSet := make(map[string]bool)

	for _, mount := range ExtractVolumes(config) {
		if mount.IsNamed {
			volumeSet[mount.Source] = true
		}
	}

	volumes := make([]string, 0, len(volumeSet))
	for vol := range volumeSet {
		volumes = append(volumes, vol)
	}

	return volumes
}

// ExtractDataDirectories returns bind mount paths that look like data directories
// Heuristics: contains "data", "db", "storage", "volumes", etc.
func ExtractDataDirectories(config *ComposeConfig) []string {
	dirSet := make(map[string]bool)
	dataKeywords := []string{"data", "db", "storage", "volumes", "postgres", "mysql", "redis", "mongo"}

	for _, mount := range ExtractVolumes(config) {
		if mount.IsNamed {
			continue // Skip named volumes
		}

		// Check if path contains data keywords
		lowerPath := strings.ToLower(mount.Source)
		for _, keyword := range dataKeywords {
			if strings.Contains(lowerPath, keyword) {
				dirSet[mount.Source] = true
				break
			}
		}
	}

	dirs := make([]string, 0, len(dirSet))
	for dir := range dirSet {
		dirs = append(dirs, dir)
	}

	return dirs
}

// parseVolumeMount parses a volume string like "postgres_data:/var/lib/postgresql/data:ro"
func parseVolumeMount(volumeStr string, serviceName string) VolumeMount {
	mount := VolumeMount{
		ServiceName: serviceName,
	}

	parts := strings.Split(volumeStr, ":")
	if len(parts) < 2 {
		// Invalid format, treat as anonymous volume
		mount.Target = volumeStr
		mount.IsNamed = false
		return mount
	}

	mount.Source = parts[0]
	mount.Target = parts[1]

	// Check if read-only
	if len(parts) >= 3 && strings.Contains(parts[2], "ro") {
		mount.IsReadOnly = true
	}

	// Determine if named volume or bind mount
	// Named volumes don't start with . or / and don't contain path separators
	if strings.HasPrefix(mount.Source, ".") ||
		strings.HasPrefix(mount.Source, "/") ||
		strings.HasPrefix(mount.Source, "~") ||
		strings.Contains(mount.Source, "\\") ||
		strings.Contains(mount.Source, "$") {
		mount.IsNamed = false
	} else {
		mount.IsNamed = true
	}

	return mount
}

// parsePortMapping parses a port string like "5432:5432" or "127.0.0.1:5432:5432/tcp"
func parsePortMapping(portStr string, serviceName string) PortMapping {
	mapping := PortMapping{
		ServiceName: serviceName,
		Protocol:    "tcp", // Default protocol
	}

	// Check for protocol suffix
	if strings.Contains(portStr, "/") {
		parts := strings.Split(portStr, "/")
		portStr = parts[0]
		if len(parts) > 1 {
			mapping.Protocol = parts[1]
		}
	}

	// Split by colon
	parts := strings.Split(portStr, ":")

	switch len(parts) {
	case 1:
		// Just container port (e.g., "8080")
		if port, err := strconv.Atoi(parts[0]); err == nil {
			mapping.ContainerPort = port
		}

	case 2:
		// host:container (e.g., "8080:80")
		if hostPort, err := strconv.Atoi(parts[0]); err == nil {
			mapping.HostPort = hostPort
		}
		if containerPort, err := strconv.Atoi(parts[1]); err == nil {
			mapping.ContainerPort = containerPort
		}

	case 3:
		// ip:host:container (e.g., "127.0.0.1:8080:80")
		// Ignore IP, just parse ports
		if hostPort, err := strconv.Atoi(parts[1]); err == nil {
			mapping.HostPort = hostPort
		}
		if containerPort, err := strconv.Atoi(parts[2]); err == nil {
			mapping.ContainerPort = containerPort
		}
	}

	return mapping
}

// GetComposePaths converts ComposeFile slice to path slice
func GetComposePaths(files []ComposeFile) []string {
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.FullPath
	}
	return paths
}

// WriteComposeFile writes a ComposeConfig to a YAML file
func WriteComposeFile(config *ComposeConfig, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return &DockerError{
			Op:   "marshal",
			File: path,
			Err:  err,
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return &DockerError{
			Op:   "write",
			File: path,
			Err:  err,
		}
	}

	return nil
}

// CleanRelativePath cleans a relative path for consistency
func CleanRelativePath(path string) string {
	// Remove leading ./
	path = strings.TrimPrefix(path, "./")
	// Convert backslashes to forward slashes
	path = filepath.ToSlash(path)
	return path
}
