package config

import "time"

type Config struct {
	Env     string        `yaml:"env" env-default:"local"`
	Server  ServerConfig  `yaml:"server"`
	Storage StorageConfig `yaml:"storage"`
}

type ServerConfig struct {
	Address string `yaml:"addr" env-default:":8080"`
}

type StorageConfig struct {
	Username       string        `yaml:"username"`
	Password       string        `yaml:"password" env:"DB_PASSWORD"`
	Host           string        `yaml:"host"`
	Database       string        `yaml:"database"`
	Port           int           `yaml:"port"`
	SSLMode        string        `yaml:"ssl_mode"`
	MaxConns       int32         `yaml:"max_conns" env-default:"10"`
	ConnectTimeout time.Duration `yaml:"connect_timeout" env-default:"5s"`
}
