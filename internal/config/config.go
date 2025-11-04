package config

import (
	"flag"
)

type Config struct {
	ServerAddress string
	BaseURL       string
}

func NewConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "адрес запуска HTTP-сервера")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "базовый адрес результирующего сокращённого URL")
	
	flag.Parse()

	return cfg
}