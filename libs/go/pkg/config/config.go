package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Load reads configuration from environment variables and/or a config file.
// The default path is the current directory, but it can be overridden.
// 'configName' is the name of the config file (without extension), defaults to "config".
// 'configType' is the type of the config file, e.g., "yaml", "json".
func Load(path string, configName string, config interface{}) error {
	v := viper.New()

	if path != "" {
		v.AddConfigPath(path)
	} else {
		v.AddConfigPath(".")
	}

	if configName != "" {
		v.SetConfigName(configName)
	} else {
		v.SetConfigName("config")
	}

	v.SetConfigType("yaml")       // Default to yaml
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// If a config file is found, read it in.
	if err := v.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, we might rely on env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	return v.Unmarshal(config)
}
