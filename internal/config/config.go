package config

type ServerConfig struct {
	Environment string `mapstructure:"SERVER_ENVIRONMENT"`
	Port        string `mapstructure:"SERVER_PORT"`
	BaseURL     string `mapstructure:"APP_BASE_URL"`
}

type DatabaseConfig struct {
	User           string `mapstructure:"DATABASE_USER"`
	Password       string `mapstructure:"DATABASE_PASSWORD"`
	Host           string `mapstructure:"DATABASE_HOST"`
	Port           string `mapstructure:"DATABASE_PORT"`
	Name           string `mapstructure:"DATABASE_NAME"`
	MaxConnections int32  `mapstructure:"DATABASE_MAX_CONNECTIONS"`
}

type CacheConfig struct {
	User     string `mapstructure:"CACHE_USER"`
	Password string `mapstructure:"CACHE_PASSWORD"`
	Host     string `mapstructure:"CACHE_HOST"`
	Port     string `mapstructure:"CACHE_PORT"`
}

type JWTConfig struct {
	SecretKey string `mapstructure:"JWT_SECRET_KEY"`
}

type AWSConfig struct {
	DefaultRegion  string `mapstructure:"AWS_DEFAULT_REGION"`
	SESSenderEmail string `mapstructure:"SES_SENDER_EMAIL"`
	SESReplyTo     string `mapstructure:"SES_REPLY_TO"`
	SESConfigSet   string `mapstructure:"SES_CONFIG_SET"`
}

type EmailConfig struct {
	Provider string `mapstructure:"EMAIL_PROVIDER"`
	Host     string `mapstructure:"EMAIL_HOST"`
	Port     int    `mapstructure:"EMAIL_PORT"`
	From     string `mapstructure:"EMAIL_FROM"`
}

type Config struct {
	Server   *ServerConfig
	Database *DatabaseConfig
	Cache    *CacheConfig
	JWT      *JWTConfig
	AWS      *AWSConfig
	Email    *EmailConfig
}
