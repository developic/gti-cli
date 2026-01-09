package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

var globalConfig *Config

func InitConfig(configFile string) {
	if configFile != "" {
		ConfigFile = ExpandPath(configFile)
		ConfigDir = filepath.Dir(ConfigFile)
		CacheDir = filepath.Join(ConfigDir, "cache")
	}

	if _, err := os.Stat(ConfigFile); os.IsNotExist(err) {
		if err := GenerateConfig(); err != nil {
			panic("Failed to generate config: " + err.Error())
		}
	}

	if err := LoadConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config file %s: %v\nUsing default configuration.\n", ConfigFile, err)
		globalConfig = DefaultConfig()
	}
}

func LoadConfig() error {
	file, err := os.Open(ConfigFile)
	if err != nil {
		return err
	}
	defer file.Close()

	cfg := DefaultConfig()
	_, err = toml.DecodeReader(file, cfg)
	if err != nil {
		return err
	}

	globalConfig = cfg
	return nil
}

func GetConfig() *Config {
	if globalConfig == nil {
		InitConfig("")
	}
	return globalConfig
}

func SaveConfig() error {
	return SaveTOMLConfig(ConfigFile, globalConfig)
}

// SaveTOMLConfig provides unified TOML config saving
func SaveTOMLConfig(filePath string, config interface{}) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	return encoder.Encode(config)
}

// SaveJSONData provides unified JSON data saving
func SaveJSONData(filePath string, data interface{}) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// LoadJSONData provides unified JSON data loading
func LoadJSONData(filePath string, data interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(data)
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		return strings.Replace(path, "~", home, 1)
	}
	return path
}
