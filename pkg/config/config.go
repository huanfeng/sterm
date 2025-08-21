// Package config provides configuration management functionality
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"serial-terminal/pkg/serial"
	"strings"
	"time"
)

// ConfigManager interface defines the contract for configuration operations
type ConfigManager interface {
	SaveConfig(name string, config serial.SerialConfig) error
	LoadConfig(name string) (serial.SerialConfig, error)
	ListConfigs() ([]ConfigInfo, error)
	DeleteConfig(name string) error
	GetDefaultConfig() serial.SerialConfig
	UpdateConfig(name string, config serial.SerialConfig) error
	ConfigExists(name string) bool
}

// ConfigInfo contains metadata about a saved configuration
type ConfigInfo struct {
	Name       string                `json:"name"`
	Config     serial.SerialConfig   `json:"config"`
	CreatedAt  time.Time            `json:"created_at"`
	LastUsedAt time.Time            `json:"last_used_at"`
	Description string               `json:"description,omitempty"`
}

// Validate checks if the configuration info is valid
func (c ConfigInfo) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("configuration name cannot be empty")
	}
	
	if err := c.Config.Validate(); err != nil {
		return fmt.Errorf("invalid serial config: %w", err)
	}
	
	if c.CreatedAt.IsZero() {
		return fmt.Errorf("created_at timestamp cannot be zero")
	}
	
	return nil
}

// ConfigStorage represents the storage format for configurations
type ConfigStorage struct {
	Configs map[string]ConfigInfo `json:"configs"`
	Version string                `json:"version"`
}

// FileConfigManager implements ConfigManager using file storage
type FileConfigManager struct {
	configDir  string
	configFile string
}

// NewFileConfigManager creates a new file-based configuration manager
func NewFileConfigManager(configDir string) *FileConfigManager {
	return &FileConfigManager{
		configDir:  configDir,
		configFile: "configs.json",
	}
}

// Initialize creates the configuration directory and initializes storage if needed
func (fcm *FileConfigManager) Initialize() error {
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(fcm.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Check if config file exists, create empty one if not
	configPath := fcm.getConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		storage := ConfigStorage{
			Configs: make(map[string]ConfigInfo),
			Version: "1.0",
		}
		
		if err := fcm.saveStorage(storage); err != nil {
			return fmt.Errorf("failed to initialize config file: %w", err)
		}
	}
	
	return nil
}

// SaveConfig saves a configuration with the given name
func (fcm *FileConfigManager) SaveConfig(name string, config serial.SerialConfig) error {
	if err := fcm.Initialize(); err != nil {
		return err
	}
	
	if name == "" {
		return fmt.Errorf("configuration name cannot be empty")
	}
	
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	
	storage, err := fcm.loadStorage()
	if err != nil {
		return fmt.Errorf("failed to load existing configurations: %w", err)
	}
	
	now := time.Now()
	configInfo := ConfigInfo{
		Name:       name,
		Config:     config,
		CreatedAt:  now,
		LastUsedAt: now,
	}
	
	// If config already exists, preserve creation time
	if existing, exists := storage.Configs[name]; exists {
		configInfo.CreatedAt = existing.CreatedAt
		configInfo.Description = existing.Description
	}
	
	storage.Configs[name] = configInfo
	
	if err := fcm.saveStorage(storage); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	
	return nil
}

// LoadConfig loads a configuration by name
func (fcm *FileConfigManager) LoadConfig(name string) (serial.SerialConfig, error) {
	if name == "" {
		return serial.SerialConfig{}, fmt.Errorf("configuration name cannot be empty")
	}
	
	storage, err := fcm.loadStorage()
	if err != nil {
		return serial.SerialConfig{}, fmt.Errorf("failed to load configurations: %w", err)
	}
	
	configInfo, exists := storage.Configs[name]
	if !exists {
		return serial.SerialConfig{}, fmt.Errorf("configuration '%s' not found", name)
	}
	
	// Update last used time
	configInfo.LastUsedAt = time.Now()
	storage.Configs[name] = configInfo
	
	// Save updated last used time (ignore errors for this non-critical update)
	fcm.saveStorage(storage)
	
	return configInfo.Config, nil
}

// ListConfigs returns a list of all saved configurations
func (fcm *FileConfigManager) ListConfigs() ([]ConfigInfo, error) {
	storage, err := fcm.loadStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to load configurations: %w", err)
	}
	
	configs := make([]ConfigInfo, 0, len(storage.Configs))
	for _, configInfo := range storage.Configs {
		configs = append(configs, configInfo)
	}
	
	return configs, nil
}

// DeleteConfig deletes a configuration by name
func (fcm *FileConfigManager) DeleteConfig(name string) error {
	if name == "" {
		return fmt.Errorf("configuration name cannot be empty")
	}
	
	storage, err := fcm.loadStorage()
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}
	
	if _, exists := storage.Configs[name]; !exists {
		return fmt.Errorf("configuration '%s' not found", name)
	}
	
	delete(storage.Configs, name)
	
	if err := fcm.saveStorage(storage); err != nil {
		return fmt.Errorf("failed to save configurations after deletion: %w", err)
	}
	
	return nil
}

// GetDefaultConfig returns the default serial configuration
func (fcm *FileConfigManager) GetDefaultConfig() serial.SerialConfig {
	return serial.DefaultConfig()
}

// UpdateConfig updates an existing configuration
func (fcm *FileConfigManager) UpdateConfig(name string, config serial.SerialConfig) error {
	if name == "" {
		return fmt.Errorf("configuration name cannot be empty")
	}
	
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	
	storage, err := fcm.loadStorage()
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}
	
	existing, exists := storage.Configs[name]
	if !exists {
		return fmt.Errorf("configuration '%s' not found", name)
	}
	
	// Update the configuration while preserving metadata
	existing.Config = config
	existing.LastUsedAt = time.Now()
	storage.Configs[name] = existing
	
	if err := fcm.saveStorage(storage); err != nil {
		return fmt.Errorf("failed to save updated configuration: %w", err)
	}
	
	return nil
}

// ConfigExists checks if a configuration with the given name exists
func (fcm *FileConfigManager) ConfigExists(name string) bool {
	if name == "" {
		return false
	}
	
	storage, err := fcm.loadStorage()
	if err != nil {
		return false
	}
	
	_, exists := storage.Configs[name]
	return exists
}

// SetConfigDescription sets the description for a configuration
func (fcm *FileConfigManager) SetConfigDescription(name, description string) error {
	if name == "" {
		return fmt.Errorf("configuration name cannot be empty")
	}
	
	storage, err := fcm.loadStorage()
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}
	
	configInfo, exists := storage.Configs[name]
	if !exists {
		return fmt.Errorf("configuration '%s' not found", name)
	}
	
	configInfo.Description = description
	storage.Configs[name] = configInfo
	
	if err := fcm.saveStorage(storage); err != nil {
		return fmt.Errorf("failed to save configuration description: %w", err)
	}
	
	return nil
}

// ExportConfig exports a configuration to a JSON file
func (fcm *FileConfigManager) ExportConfig(name, filePath string) error {
	if name == "" {
		return fmt.Errorf("configuration name cannot be empty")
	}
	
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	
	storage, err := fcm.loadStorage()
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}
	
	configInfo, exists := storage.Configs[name]
	if !exists {
		return fmt.Errorf("configuration '%s' not found", name)
	}
	
	data, err := json.MarshalIndent(configInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}
	
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}
	
	return nil
}

// ImportConfig imports a configuration from a JSON file
func (fcm *FileConfigManager) ImportConfig(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}
	
	var configInfo ConfigInfo
	if err := json.Unmarshal(data, &configInfo); err != nil {
		return fmt.Errorf("failed to parse configuration file: %w", err)
	}
	
	if err := configInfo.Validate(); err != nil {
		return fmt.Errorf("invalid configuration in file: %w", err)
	}
	
	// Save the imported configuration
	return fcm.SaveConfig(configInfo.Name, configInfo.Config)
}

// SearchConfigs searches for configurations by name or description
func (fcm *FileConfigManager) SearchConfigs(query string) ([]ConfigInfo, error) {
	if query == "" {
		return fcm.ListConfigs()
	}
	
	storage, err := fcm.loadStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to load configurations: %w", err)
	}
	
	query = strings.ToLower(query)
	var results []ConfigInfo
	
	for _, configInfo := range storage.Configs {
		if strings.Contains(strings.ToLower(configInfo.Name), query) ||
		   strings.Contains(strings.ToLower(configInfo.Description), query) {
			results = append(results, configInfo)
		}
	}
	
	return results, nil
}

// GetConfigPath returns the full path to the configuration file
func (fcm *FileConfigManager) GetConfigPath() string {
	return fcm.getConfigPath()
}

// BackupConfigs creates a backup of all configurations
func (fcm *FileConfigManager) BackupConfigs(backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("backup path cannot be empty")
	}
	
	configPath := fcm.getConfigPath()
	
	// Read the current config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}
	
	// Write to backup location
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}
	
	return nil
}

// RestoreConfigs restores configurations from a backup
func (fcm *FileConfigManager) RestoreConfigs(backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("backup path cannot be empty")
	}
	
	// Validate the backup file by loading it
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}
	
	var storage ConfigStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		return fmt.Errorf("invalid backup file format: %w", err)
	}
	
	// Validate all configurations in the backup
	for name, configInfo := range storage.Configs {
		if err := configInfo.Validate(); err != nil {
			return fmt.Errorf("invalid configuration '%s' in backup: %w", name, err)
		}
	}
	
	// If validation passes, restore the backup
	configPath := fcm.getConfigPath()
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore configuration file: %w", err)
	}
	
	return nil
}

// Private helper methods

// getConfigPath returns the full path to the configuration file
func (fcm *FileConfigManager) getConfigPath() string {
	return filepath.Join(fcm.configDir, fcm.configFile)
}

// loadStorage loads the configuration storage from file
func (fcm *FileConfigManager) loadStorage() (ConfigStorage, error) {
	configPath := fcm.getConfigPath()
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty storage if file doesn't exist
			return ConfigStorage{
				Configs: make(map[string]ConfigInfo),
				Version: "1.0",
			}, nil
		}
		return ConfigStorage{}, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var storage ConfigStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		return ConfigStorage{}, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Initialize configs map if nil
	if storage.Configs == nil {
		storage.Configs = make(map[string]ConfigInfo)
	}
	
	return storage, nil
}

// saveStorage saves the configuration storage to file
func (fcm *FileConfigManager) saveStorage(storage ConfigStorage) error {
	configPath := fcm.getConfigPath()
	
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config data: %w", err)
	}
	
	// Write to temporary file first, then rename for atomic operation
	tempPath := configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary config file: %w", err)
	}
	
	if err := os.Rename(tempPath, configPath); err != nil {
		// Clean up temporary file on failure
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temporary config file: %w", err)
	}
	
	return nil
}