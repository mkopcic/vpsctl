package config

import "github.com/spf13/viper"

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

func Load() (*Config, error) {

	viper.SetConfigName("servers")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	var cfg Config

	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
