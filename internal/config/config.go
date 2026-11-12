package config

import (
	"os"

	"github.com/spf13/viper"
)

// defaultPath vrijedi kad VPSCTL_CONFIG nije postavljen — relativno na
// working directory, kao i dosad.
const defaultPath = "configs/servers.yaml"

type Server struct {
	Name     string `mapstructure:"name"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Key      string `mapstructure:"key"`
	Password string `mapstructure:"password"`
	// Domains se koriste za `vpsctl certs` (provjera TLS certifikata)
	Domains []string `mapstructure:"domains"`
}

type Config struct {
	Servers []Server `mapstructure:"servers"`
}

// Find vraca server po imenu. ok=false ako ne postoji.
func (c *Config) Find(name string) (Server, bool) {
	for _, s := range c.Servers {
		if s.Name == name {
			return s, true
		}
	}
	return Server{}, false
}

// Path vraca put do servers.yaml.
// VPSCTL_CONFIG ima prednost — u containeru config dolazi iz volumea ili
// ConfigMapa, ne iz ./configs.
func Path() string {
	if p := os.Getenv("VPSCTL_CONFIG"); p != "" {
		return p
	}
	return defaultPath
}

func Load() (*Config, error) {
	// viper.New() umjesto globalnog vipera — svaki Load cita ispocetka,
	// bez stanja koje bi ostalo od prethodnog poziva
	v := viper.New()
	v.SetConfigFile(Path())

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
