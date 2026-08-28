package config

import (
	"strings"

	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.SetDefault("DATABASE_MAX_CONNECTIONS", 5)

	config := Config{
		Server: &ServerConfig{
			Environment: viper.GetString("SERVER_ENVIRONMENT"),
			Port:        viper.GetString("SERVER_PORT"),
			BaseURL:     viper.GetString("APP_BASE_URL"),
		},
		Database: &DatabaseConfig{
			User:           viper.GetString("DATABASE_USER"),
			Password:       viper.GetString("DATABASE_PASSWORD"),
			Host:           viper.GetString("DATABASE_HOST"),
			Port:           viper.GetString("DATABASE_PORT"),
			Name:           viper.GetString("DATABASE_NAME"),
			MaxConnections: int32(viper.GetInt("DATABASE_MAX_CONNECTIONS")),
		},
		Cache: &CacheConfig{
			User:     viper.GetString("CACHE_USER"),
			Password: viper.GetString("CACHE_PASSWORD"),
			Host:     viper.GetString("CACHE_HOST"),
			Port:     viper.GetString("CACHE_PORT"),
		},
		JWT: &JWTConfig{
			SecretKey: viper.GetString("JWT_SECRET_KEY"),
		},
		AWS: &AWSConfig{
			DefaultRegion:  viper.GetString("AWS_DEFAULT_REGION"),
			SESSenderEmail: viper.GetString("SES_SENDER_EMAIL"),
			SESReplyTo:     viper.GetString("SES_REPLY_TO"),
			SESConfigSet:   viper.GetString("SES_CONFIG_SET"),
		},
		Email: &EmailConfig{
			Provider: viper.GetString("EMAIL_PROVIDER"),
			Host:     viper.GetString("EMAIL_HOST"),
			Port:     viper.GetInt("EMAIL_PORT"),
			From:     viper.GetString("EMAIL_FROM"),
		},
	}

	return &config, nil
}
