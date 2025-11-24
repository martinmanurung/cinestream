package config

import (
	"log"

	"github.com/spf13/viper"
)

var AppConfig Config

func LoadConfig() (*Config, error) {
	viper.SetConfigName("app-config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("Unable to decode config into struct: %s", err)
	}

	log.Println("Configuration loaded successfully.")
	return &AppConfig, nil
}
