package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseComposeFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test compose file
	composeContent := `version: '3.8'
services:
  db:
    image: postgres:14
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"
    environment:
      POSTGRES_PASSWORD: secret

  redis:
    image: redis:7
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"

volumes:
  postgres_data:
    name: postgres_data
  redis_data:
    name: redis_data
`

	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := ParseComposeFile(composePath)
	if err != nil {
		t.Fatalf("Failed to parse compose file: %v", err)
	}

	// Check version
	if config.Version != "3.8" {
		t.Errorf("Expected version 3.8, got %s", config.Version)
	}

	// Check services
	if len(config.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(config.Services))
	}

	// Check db service
	db, ok := config.Services["db"]
	if !ok {
		t.Fatal("db service not found")
	}
	if db.Image != "postgres:14" {
		t.Errorf("Expected image postgres:14, got %s", db.Image)
	}
	if len(db.Volumes) != 2 {
		t.Errorf("Expected 2 volumes, got %d", len(db.Volumes))
	}
	if len(db.Ports) != 1 {
		t.Errorf("Expected 1 port, got %d", len(db.Ports))
	}

	// Check volumes
	if len(config.Volumes) != 2 {
		t.Errorf("Expected 2 volumes, got %d", len(config.Volumes))
	}
}

func TestParseVolumeMount(t *testing.T) {
	tests := []struct {
		name           string
		volumeStr      string
		expectSource   string
		expectTarget   string
		expectNamed    bool
		expectReadOnly bool
	}{
		{
			name:           "Named volume",
			volumeStr:      "postgres_data:/var/lib/postgresql/data",
			expectSource:   "postgres_data",
			expectTarget:   "/var/lib/postgresql/data",
			expectNamed:    true,
			expectReadOnly: false,
		},
		{
			name:           "Named volume read-only",
			volumeStr:      "postgres_data:/var/lib/postgresql/data:ro",
			expectSource:   "postgres_data",
			expectTarget:   "/var/lib/postgresql/data",
			expectNamed:    true,
			expectReadOnly: true,
		},
		{
			name:           "Bind mount relative",
			volumeStr:      "./data:/app/data",
			expectSource:   "./data",
			expectTarget:   "/app/data",
			expectNamed:    false,
			expectReadOnly: false,
		},
		{
			name:           "Bind mount absolute",
			volumeStr:      "/host/data:/container/data",
			expectSource:   "/host/data",
			expectTarget:   "/container/data",
			expectNamed:    false,
			expectReadOnly: false,
		},
		{
			name:           "Variable expansion",
			volumeStr:      "${DATA_DIR}:/app/data",
			expectSource:   "${DATA_DIR}",
			expectTarget:   "/app/data",
			expectNamed:    false,
			expectReadOnly: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mount := parseVolumeMount(tt.volumeStr, "test-service")

			if mount.Source != tt.expectSource {
				t.Errorf("Expected source %s, got %s", tt.expectSource, mount.Source)
			}
			if mount.Target != tt.expectTarget {
				t.Errorf("Expected target %s, got %s", tt.expectTarget, mount.Target)
			}
			if mount.IsNamed != tt.expectNamed {
				t.Errorf("Expected IsNamed %v, got %v", tt.expectNamed, mount.IsNamed)
			}
			if mount.IsReadOnly != tt.expectReadOnly {
				t.Errorf("Expected IsReadOnly %v, got %v", tt.expectReadOnly, mount.IsReadOnly)
			}
		})
	}
}

func TestParsePortMapping(t *testing.T) {
	tests := []struct {
		name            string
		portStr         string
		expectHost      int
		expectContainer int
		expectProtocol  string
	}{
		{
			name:            "Simple mapping",
			portStr:         "8080:80",
			expectHost:      8080,
			expectContainer: 80,
			expectProtocol:  "tcp",
		},
		{
			name:            "With IP",
			portStr:         "127.0.0.1:8080:80",
			expectHost:      8080,
			expectContainer: 80,
			expectProtocol:  "tcp",
		},
		{
			name:            "With protocol",
			portStr:         "53:53/udp",
			expectHost:      53,
			expectContainer: 53,
			expectProtocol:  "udp",
		},
		{
			name:            "Container only",
			portStr:         "80",
			expectHost:      0,
			expectContainer: 80,
			expectProtocol:  "tcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := parsePortMapping(tt.portStr, "test-service")

			if mapping.HostPort != tt.expectHost {
				t.Errorf("Expected host port %d, got %d", tt.expectHost, mapping.HostPort)
			}
			if mapping.ContainerPort != tt.expectContainer {
				t.Errorf("Expected container port %d, got %d", tt.expectContainer, mapping.ContainerPort)
			}
			if mapping.Protocol != tt.expectProtocol {
				t.Errorf("Expected protocol %s, got %s", tt.expectProtocol, mapping.Protocol)
			}
		})
	}
}

func TestExtractNamedVolumes(t *testing.T) {
	config := &ComposeConfig{
		Services: map[string]Service{
			"db": {
				Volumes: []string{
					"postgres_data:/var/lib/postgresql/data",
					"./init:/docker-entrypoint-initdb.d",
				},
			},
			"redis": {
				Volumes: []string{
					"redis_data:/data",
				},
			},
		},
	}

	namedVolumes := ExtractNamedVolumes(config)

	if len(namedVolumes) != 2 {
		t.Errorf("Expected 2 named volumes, got %d", len(namedVolumes))
	}

	// Check that we got the right volumes (order doesn't matter)
	volumeMap := make(map[string]bool)
	for _, v := range namedVolumes {
		volumeMap[v] = true
	}

	if !volumeMap["postgres_data"] {
		t.Error("Expected postgres_data in named volumes")
	}
	if !volumeMap["redis_data"] {
		t.Error("Expected redis_data in named volumes")
	}
}

func TestExtractDataDirectories(t *testing.T) {
	config := &ComposeConfig{
		Services: map[string]Service{
			"db": {
				Volumes: []string{
					"postgres_data:/var/lib/postgresql/data",
					"./db-data:/data",
					"./config:/config",
				},
			},
			"app": {
				Volumes: []string{
					"./storage:/app/storage",
					"./logs:/app/logs",
				},
			},
		},
	}

	dataDirs := ExtractDataDirectories(config)

	// Should find ./db-data and ./storage (contain keywords)
	// May or may not find ./logs depending on implementation

	foundDbData := false
	foundStorage := false

	for _, dir := range dataDirs {
		if dir == "./db-data" {
			foundDbData = true
		}
		if dir == "./storage" {
			foundStorage = true
		}
	}

	if !foundDbData {
		t.Error("Expected to find ./db-data")
	}
	if !foundStorage {
		t.Error("Expected to find ./storage")
	}
}

func TestMergeConfigs(t *testing.T) {
	base := &ComposeConfig{
		Version: "3.8",
		Services: map[string]Service{
			"db": {
				Image: "postgres:14",
				Ports: []string{"5432:5432"},
			},
		},
	}

	override := &ComposeConfig{
		Services: map[string]Service{
			"db": {
				Ports: []string{"5433:5432"},
			},
		},
	}

	merged := mergeConfigs(base, override)

	// Check that override took effect
	db := merged.Services["db"]
	if len(db.Ports) != 1 || db.Ports[0] != "5433:5432" {
		t.Errorf("Expected port 5433:5432, got %v", db.Ports)
	}

	// Check that base properties remain
	if db.Image != "postgres:14" {
		t.Errorf("Expected image postgres:14, got %s", db.Image)
	}
}
