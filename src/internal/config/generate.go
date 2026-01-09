package config

import (
	"os"
)

func GenerateConfig() error {
	dirs := []string{ConfigDir, CacheDir}
	for _, dir := range dirs {
		if err := EnsureDir(dir); err != nil {
			return err
		}
	}

	cfg := DefaultConfig()
	return SaveTOMLConfig(ConfigFile, cfg)
}

// EnsureDir provides unified directory creation with standard permissions
func EnsureDir(dirPath string) error {
	return os.MkdirAll(dirPath, 0755)
}
