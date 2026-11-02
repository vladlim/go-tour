package config

import (
	"flag"
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

func MustLoad() *Config {
	cfgPath := flag.String("config_path", "./configs/config.yaml", "config for service")
	flag.Parse()

	if *cfgPath == "" {
		panic("config_path is empty")
	}

	var cfg Config

	if err := cleanenv.ReadConfig(*cfgPath, &cfg); err != nil {
		panic(fmt.Errorf("error reading config: %w", err))
	}

	if cfg.Storage.Host == "" || cfg.Storage.Database == "" || cfg.Storage.Username == "" {
		panic("storage host, database and username are required")
	}
	if cfg.Storage.Port < 1 || cfg.Storage.Port > 65535 {
		panic("storage port must be between 1 and 65535")
	}
	if cfg.Storage.MaxConns < 1 || cfg.Storage.ConnectTimeout <= 0 {
		panic("storage max_conns and connect_timeout must be positive")
	}
	switch cfg.Storage.SSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		panic("invalid storage ssl_mode")
	}

	return &cfg
}
