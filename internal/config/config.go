package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Port      int
	DBPath    string
	DataDir   string
	MasterKey string
	MITMPort  int
}

func Load() (*Config, error) {
	dataDir := getEnv("LINTASAN_DATA_DIR", "./data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	return &Config{
		Port:      getEnvInt("PORT", 20180),
		DBPath:    filepath.Join(dataDir, "lintasan.db"),
		DataDir:   dataDir,
		MasterKey: getEnv("LINTASAN_MASTER_KEY", ""),
		MITMPort:  getEnvInt("MITM_PORT", 8443),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
