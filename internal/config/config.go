package config

import (
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	WriteDBURL   string   `mapstructure:"write_db_url"`
	ReadDBURL    string   `mapstructure:"read_db_url"`
	RedisURL     string   `mapstructure:"redis_url"`
	KafkaBrokers []string `mapstructure:"kafka_brokers"`
	KafkaTopic   string   `mapstructure:"kafka_topic"`
	Environment  string   `mapstructure:"environment"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/root/")  // for Docker

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Override with environment variables if they exist
	if envWriteDBURL := viper.GetString("WRITE_DB_URL"); envWriteDBURL != "" {
		config.WriteDBURL = envWriteDBURL
	}
	if envReadDBURL := viper.GetString("READ_DB_URL"); envReadDBURL != "" {
		config.ReadDBURL = envReadDBURL
	}
	if envRedisURL := viper.GetString("REDIS_URL"); envRedisURL != "" {
		config.RedisURL = envRedisURL
	}
	if envKafkaBrokers := viper.GetString("KAFKA_BROKERS"); envKafkaBrokers != "" {
		config.KafkaBrokers = strings.Split(envKafkaBrokers, ",")
	}
	if envKafkaTopic := viper.GetString("KAFKA_TOPIC"); envKafkaTopic != "" {
		config.KafkaTopic = envKafkaTopic
	}
	if envEnvironment := viper.GetString("ENVIRONMENT"); envEnvironment != "" {
		config.Environment = envEnvironment
	}

	return &config, nil
}