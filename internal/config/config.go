package config

import "github.com/spf13/viper"

type Server struct {
	Name     string `mapstructure:"name"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Key      string `mapstructure:"key"`
	Password string `mapstructure:"password"`
}

type Config struct {
	Servers []Server `mapstructure:"servers"`
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
