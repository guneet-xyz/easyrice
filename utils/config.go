package utils

import (
	"log/slog"

	"github.com/BurntSushi/toml"
)

type Config struct {
	RepositoryDir string
}

func GetConfig() (Config, error) {
	slog.Debug("Getting config")
	bytes, err := ReadBytes(ConfigFile)

	if err != nil {
		slog.Error("Error while reading config")
		return Config{}, err
	}

	config := Config{}
	err = toml.Unmarshal(bytes, &config)
	if err != nil {
		slog.Error("Error while parsing config file", "path", ConfigFile, "error", err)
		return Config{}, err
	}

	if IsEmpty(config.RepositoryDir) {
		slog.Error("Config is not valid.")
		return Config{}, err
	}

	return config, nil
}

func UpdateConfig(config Config) error {
	slog.Debug("Updating config", "config", config)
	bytes, err := toml.Marshal(config)
	if err != nil {
		slog.Warn("Could not marshal config", "error", err)
		return err
	}

	err = WriteBytes(ConfigFile, bytes)
	if err != nil {
		slog.Warn("Error while updating config", "error", err)
		return err
	}

	slog.Debug("Updated config")
	return nil
}
