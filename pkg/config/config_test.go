package config

import (
	"fmt"
	"os"
	"path/filepath"
	"serial-terminal/pkg/serial"
	"testing"
	"time"
)

func TestConfigInfo_Validate(t *testing.T) {
	validConfig := serial.SerialConfig{
		Port:     "COM1",
		BaudRate: 115200,
		DataBits: 8,
		StopBits: 1,
		Parity:   "none",
		Timeout:  time.Second * 5,
	}

	tests := []struct {
		name    string
		config  ConfigInfo
		wantErr bool
	}{
		{
			name: "valid config info",
			config: ConfigInfo{
				Name:       "test-config",
				Config:     validConfig,
				CreatedAt:  time.Now(),
				LastUsedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "empty name",
			config: ConfigInfo{
				Name:       "",
				Config:     validConfig,
				CreatedAt:  time.Now(),
				LastUsedAt: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "invalid serial config",
			config: ConfigInfo{
				Name: "test-config",
				Config: serial.SerialConfig{
					Port:     "",
					BaudRate: 115200,
					DataBits: 8,
					StopBits: 1,
					Parity:   "none",
					Timeout:  time.Second * 5,
				},
				CreatedAt:  time.Now(),
				LastUsedAt: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "zero created at",
			config: ConfigInfo{
				Name:       "test-config",
				Config:     validConfig,
				CreatedAt:  time.Time{},
				LastUsedAt: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigInfo.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewFileConfigManager(t *testing.T) {
	configDir := "/test/config"
	manager := NewFileConfigManager(configDir)

	if manager == nil {
		t.Error("NewFileConfigManager() returned nil")
	}

	if manager.configDir != configDir {
		t.Errorf("NewFileConfigManager() configDir = %s, want %s", manager.configDir, configDir)
	}

	if manager.configFile != "configs.json" {
		t.Errorf("NewFileConfigManager() configFile = %s, want configs.json", manager.configFile)
	}
}

func TestConfigInfo_WithDescription(t *testing.T) {
	config := ConfigInfo{
		Name: "test-config",
		Config: serial.SerialConfig{
			Port:     "COM1",
			BaudRate: 115200,
			DataBits: 8,
			StopBits: 1,
			Parity:   "none",
			Timeout:  time.Second * 5,
		},
		CreatedAt:   time.Now(),
		LastUsedAt:  time.Now(),
		Description: "Test configuration for COM1",
	}

	if err := config.Validate(); err != nil {
		t.Errorf("ConfigInfo with description should be valid: %v", err)
	}

	if config.Description != "Test configuration for COM1" {
		t.Errorf("ConfigInfo.Description = %s, want 'Test configuration for COM1'", config.Description)
	}
}

func TestConfigStorage_Structure(t *testing.T) {
	storage := ConfigStorage{
		Configs: make(map[string]ConfigInfo),
		Version: "1.0",
	}

	if storage.Configs == nil {
		t.Error("ConfigStorage.Configs should not be nil")
	}

	if storage.Version != "1.0" {
		t.Errorf("ConfigStorage.Version = %s, want '1.0'", storage.Version)
	}

	// Test adding a config
	testConfig := ConfigInfo{
		Name: "test",
		Config: serial.SerialConfig{
			Port:     "COM1",
			BaudRate: 115200,
			DataBits: 8,
			StopBits: 1,
			Parity:   "none",
			Timeout:  time.Second * 5,
		},
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	storage.Configs["test"] = testConfig

	if len(storage.Configs) != 1 {
		t.Errorf("ConfigStorage should contain 1 config, got %d", len(storage.Configs))
	}

	retrieved, exists := storage.Configs["test"]
	if !exists {
		t.Error("Config 'test' should exist in storage")
	}

	if retrieved.Name != "test" {
		t.Errorf("Retrieved config name = %s, want 'test'", retrieved.Name)
	}
}

func TestFileConfigManager_Initialize(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	err := manager.Initialize()
	if err != nil {
		t.Errorf("Initialize() failed: %v", err)
	}

	// Check if config file was created
	configPath := manager.GetConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should be created after Initialize()")
	}
}

func TestFileConfigManager_SaveAndLoadConfig(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	config := serial.SerialConfig{
		Port:     "COM1",
		BaudRate: 115200,
		DataBits: 8,
		StopBits: 1,
		Parity:   "none",
		Timeout:  time.Second * 5,
	}

	// Save config
	err := manager.SaveConfig("test-config", config)
	if err != nil {
		t.Errorf("SaveConfig() failed: %v", err)
	}

	// Load config
	loadedConfig, err := manager.LoadConfig("test-config")
	if err != nil {
		t.Errorf("LoadConfig() failed: %v", err)
	}

	if loadedConfig.Port != config.Port {
		t.Errorf("Loaded config Port = %s, want %s", loadedConfig.Port, config.Port)
	}

	if loadedConfig.BaudRate != config.BaudRate {
		t.Errorf("Loaded config BaudRate = %d, want %d", loadedConfig.BaudRate, config.BaudRate)
	}
}

func TestFileConfigManager_SaveConfigEmptyName(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	config := serial.DefaultConfig()

	err := manager.SaveConfig("", config)
	if err == nil {
		t.Error("SaveConfig() with empty name should return error")
	}
}

func TestFileConfigManager_LoadConfigNotFound(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	_, err := manager.LoadConfig("non-existent")
	if err == nil {
		t.Error("LoadConfig() for non-existent config should return error")
	}
}

func TestFileConfigManager_ListConfigs(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	config1 := serial.DefaultConfig()
	config1.Port = "COM1"

	config2 := serial.DefaultConfig()
	config2.Port = "COM2"

	// Save two configs
	err := manager.SaveConfig("config1", config1)
	if err != nil {
		t.Errorf("SaveConfig() failed: %v", err)
	}

	err = manager.SaveConfig("config2", config2)
	if err != nil {
		t.Errorf("SaveConfig() failed: %v", err)
	}

	// List configs
	configs, err := manager.ListConfigs()
	if err != nil {
		t.Errorf("ListConfigs() failed: %v", err)
	}

	if len(configs) != 2 {
		t.Errorf("ListConfigs() returned %d configs, want 2", len(configs))
	}
}

func TestFileConfigManager_DeleteConfig(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	config := serial.DefaultConfig()

	// Save config
	err := manager.SaveConfig("test-config", config)
	if err != nil {
		t.Errorf("SaveConfig() failed: %v", err)
	}

	// Verify it exists
	if !manager.ConfigExists("test-config") {
		t.Error("Config should exist before deletion")
	}

	// Delete config
	err = manager.DeleteConfig("test-config")
	if err != nil {
		t.Errorf("DeleteConfig() failed: %v", err)
	}

	// Verify it's gone
	if manager.ConfigExists("test-config") {
		t.Error("Config should not exist after deletion")
	}
}

func TestFileConfigManager_DeleteConfigNotFound(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	err := manager.DeleteConfig("non-existent")
	if err == nil {
		t.Error("DeleteConfig() for non-existent config should return error")
	}
}

func TestFileConfigManager_UpdateConfig(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	originalConfig := serial.DefaultConfig()
	originalConfig.Port = "COM1"

	// Save original config
	err := manager.SaveConfig("test-config", originalConfig)
	if err != nil {
		t.Errorf("SaveConfig() failed: %v", err)
	}

	// Update config
	updatedConfig := originalConfig
	updatedConfig.Port = "COM2"

	err = manager.UpdateConfig("test-config", updatedConfig)
	if err != nil {
		t.Errorf("UpdateConfig() failed: %v", err)
	}

	// Load and verify update
	loadedConfig, err := manager.LoadConfig("test-config")
	if err != nil {
		t.Errorf("LoadConfig() failed: %v", err)
	}

	if loadedConfig.Port != "COM2" {
		t.Errorf("Updated config Port = %s, want COM2", loadedConfig.Port)
	}
}

func TestFileConfigManager_UpdateConfigNotFound(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	config := serial.DefaultConfig()

	err := manager.UpdateConfig("non-existent", config)
	if err == nil {
		t.Error("UpdateConfig() for non-existent config should return error")
	}
}

func TestFileConfigManager_GetDefaultConfig(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	defaultConfig := manager.GetDefaultConfig()

	if err := defaultConfig.Validate(); err != nil {
		t.Errorf("GetDefaultConfig() should return valid config: %v", err)
	}
}

func TestFileConfigManager_ConfigExists(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	// Should not exist initially
	if manager.ConfigExists("test-config") {
		t.Error("Config should not exist initially")
	}

	// Save config
	config := serial.DefaultConfig()
	err := manager.SaveConfig("test-config", config)
	if err != nil {
		t.Errorf("SaveConfig() failed: %v", err)
	}

	// Should exist now
	if !manager.ConfigExists("test-config") {
		t.Error("Config should exist after saving")
	}

	// Empty name should return false
	if manager.ConfigExists("") {
		t.Error("ConfigExists() with empty name should return false")
	}
}

func TestFileConfigManager_SetConfigDescription(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	config := serial.DefaultConfig()

	// Save config
	err := manager.SaveConfig("test-config", config)
	if err != nil {
		t.Errorf("SaveConfig() failed: %v", err)
	}

	// Set description
	description := "Test configuration for COM1"
	err = manager.SetConfigDescription("test-config", description)
	if err != nil {
		t.Errorf("SetConfigDescription() failed: %v", err)
	}

	// Verify description was set
	configs, err := manager.ListConfigs()
	if err != nil {
		t.Errorf("ListConfigs() failed: %v", err)
	}

	found := false
	for _, configInfo := range configs {
		if configInfo.Name == "test-config" {
			if configInfo.Description != description {
				t.Errorf("Config description = %s, want %s", configInfo.Description, description)
			}
			found = true
			break
		}
	}

	if !found {
		t.Error("Config not found in list")
	}
}

func TestFileConfigManager_SearchConfigs(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	// Save configs with different names and descriptions
	config1 := serial.DefaultConfig()
	config1.Port = "COM1"
	manager.SaveConfig("arduino-config", config1)
	manager.SetConfigDescription("arduino-config", "Configuration for Arduino board")

	config2 := serial.DefaultConfig()
	config2.Port = "COM2"
	manager.SaveConfig("sensor-config", config2)
	manager.SetConfigDescription("sensor-config", "Configuration for sensor module")

	// Search by name
	results, err := manager.SearchConfigs("arduino")
	if err != nil {
		t.Errorf("SearchConfigs() failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("SearchConfigs('arduino') returned %d results, want 1", len(results))
	}

	if results[0].Name != "arduino-config" {
		t.Errorf("Search result name = %s, want arduino-config", results[0].Name)
	}

	// Search by description
	results, err = manager.SearchConfigs("sensor")
	if err != nil {
		t.Errorf("SearchConfigs() failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("SearchConfigs('sensor') returned %d results, want 1", len(results))
	}

	// Empty search should return all
	results, err = manager.SearchConfigs("")
	if err != nil {
		t.Errorf("SearchConfigs('') failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("SearchConfigs('') returned %d results, want 2", len(results))
	}
}

func TestFileConfigManager_ExportImportConfig(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	config := serial.DefaultConfig()
	config.Port = "COM1"

	// Save config
	err := manager.SaveConfig("test-config", config)
	if err != nil {
		t.Errorf("SaveConfig() failed: %v", err)
	}

	// Set description
	err = manager.SetConfigDescription("test-config", "Test configuration")
	if err != nil {
		t.Errorf("SetConfigDescription() failed: %v", err)
	}

	// Export config
	exportPath := filepath.Join(tempDir, "exported-config.json")
	err = manager.ExportConfig("test-config", exportPath)
	if err != nil {
		t.Errorf("ExportConfig() failed: %v", err)
	}

	// Verify export file exists
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Error("Export file should exist")
	}

	// Delete original config
	err = manager.DeleteConfig("test-config")
	if err != nil {
		t.Errorf("DeleteConfig() failed: %v", err)
	}

	// Import config back
	err = manager.ImportConfig(exportPath)
	if err != nil {
		t.Errorf("ImportConfig() failed: %v", err)
	}

	// Verify imported config
	if !manager.ConfigExists("test-config") {
		t.Error("Imported config should exist")
	}

	loadedConfig, err := manager.LoadConfig("test-config")
	if err != nil {
		t.Errorf("LoadConfig() failed: %v", err)
	}

	if loadedConfig.Port != "COM1" {
		t.Errorf("Imported config Port = %s, want COM1", loadedConfig.Port)
	}
}

func TestFileConfigManager_BackupRestore(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	// Save some configs
	config1 := serial.DefaultConfig()
	config1.Port = "COM1"
	manager.SaveConfig("config1", config1)

	config2 := serial.DefaultConfig()
	config2.Port = "COM2"
	manager.SaveConfig("config2", config2)

	// Create backup
	backupPath := filepath.Join(tempDir, "backup.json")
	err := manager.BackupConfigs(backupPath)
	if err != nil {
		t.Errorf("BackupConfigs() failed: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup file should exist")
	}

	// Delete all configs
	manager.DeleteConfig("config1")
	manager.DeleteConfig("config2")

	// Verify configs are gone
	configs, _ := manager.ListConfigs()
	if len(configs) != 0 {
		t.Errorf("Should have 0 configs after deletion, got %d", len(configs))
	}

	// Restore from backup
	err = manager.RestoreConfigs(backupPath)
	if err != nil {
		t.Errorf("RestoreConfigs() failed: %v", err)
	}

	// Verify configs are restored
	configs, err = manager.ListConfigs()
	if err != nil {
		t.Errorf("ListConfigs() failed: %v", err)
	}

	if len(configs) != 2 {
		t.Errorf("Should have 2 configs after restore, got %d", len(configs))
	}
}

func TestFileConfigManager_ErrorCases(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	// Test export non-existent config
	err := manager.ExportConfig("non-existent", filepath.Join(tempDir, "export.json"))
	if err == nil {
		t.Error("ExportConfig() for non-existent config should return error")
	}

	// Test export with empty file path
	config := serial.DefaultConfig()
	manager.SaveConfig("test-config", config)

	err = manager.ExportConfig("test-config", "")
	if err == nil {
		t.Error("ExportConfig() with empty file path should return error")
	}

	// Test import non-existent file
	err = manager.ImportConfig("non-existent.json")
	if err == nil {
		t.Error("ImportConfig() for non-existent file should return error")
	}

	// Test backup with empty path
	err = manager.BackupConfigs("")
	if err == nil {
		t.Error("BackupConfigs() with empty path should return error")
	}

	// Test restore with empty path
	err = manager.RestoreConfigs("")
	if err == nil {
		t.Error("RestoreConfigs() with empty path should return error")
	}
}

func TestFileConfigManager_InvalidConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	// Create invalid config file
	configPath := manager.GetConfigPath()
	os.MkdirAll(filepath.Dir(configPath), 0755)
	os.WriteFile(configPath, []byte("invalid json"), 0644)

	// Try to load configs - should handle gracefully
	_, err := manager.ListConfigs()
	if err == nil {
		t.Error("ListConfigs() with invalid config file should return error")
	}
}

func TestFileConfigManager_SequentialOperations(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewFileConfigManager(tempDir)

	config := serial.DefaultConfig()

	// Test sequential saves to ensure basic functionality works
	for i := 0; i < 10; i++ {
		config.Port = fmt.Sprintf("COM%d", i+1)
		err := manager.SaveConfig(fmt.Sprintf("config_%d", i), config)
		if err != nil {
			t.Errorf("SaveConfig() failed for config_%d: %v", i, err)
		}
	}

	// Verify all configs were saved
	configs, err := manager.ListConfigs()
	if err != nil {
		t.Errorf("ListConfigs() failed: %v", err)
	}

	if len(configs) != 10 {
		t.Errorf("Should have 10 configs, got %d", len(configs))
	}

	// Test sequential operations (load, update, delete)
	for i := 0; i < 5; i++ {
		configName := fmt.Sprintf("config_%d", i)

		// Load config
		loadedConfig, err := manager.LoadConfig(configName)
		if err != nil {
			t.Errorf("LoadConfig() failed for %s: %v", configName, err)
		}

		// Update config
		loadedConfig.BaudRate = 9600
		err = manager.UpdateConfig(configName, loadedConfig)
		if err != nil {
			t.Errorf("UpdateConfig() failed for %s: %v", configName, err)
		}

		// Verify update
		updatedConfig, err := manager.LoadConfig(configName)
		if err != nil {
			t.Errorf("LoadConfig() after update failed for %s: %v", configName, err)
		}

		if updatedConfig.BaudRate != 9600 {
			t.Errorf("Config %s baud rate = %d, want 9600", configName, updatedConfig.BaudRate)
		}
	}

	// Delete some configs
	for i := 0; i < 3; i++ {
		configName := fmt.Sprintf("config_%d", i)
		err := manager.DeleteConfig(configName)
		if err != nil {
			t.Errorf("DeleteConfig() failed for %s: %v", configName, err)
		}
	}

	// Verify final count
	finalConfigs, err := manager.ListConfigs()
	if err != nil {
		t.Errorf("Final ListConfigs() failed: %v", err)
	}

	if len(finalConfigs) != 7 {
		t.Errorf("Should have 7 configs after deletions, got %d", len(finalConfigs))
	}
}
