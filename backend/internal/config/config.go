package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App       AppConfig
	HTTP      HTTPConfig
	Postgres  PostgresConfig
	Redis     RedisConfig
	Search    SearchConfig
	Analytics AnalyticsConfig
	Auth      AuthConfig
	Assistant AssistantConfig
	Storage   StorageConfig
	GeoIP     GeoIPConfig
	SMTP      SMTPConfig
	OAuth     OAuthConfig
}

type OAuthConfig struct {
	PublicURL          string
	WebURL             string
	GitHubClientID     string
	GitHubClientSecret string
	GoogleClientID     string
	GoogleClientSecret string
}

type SMTPConfig struct {
	Host     string
	Port     int
	From     string
	TLSMode  string
	Username string
	Password string
}

type GeoIPConfig struct{ DBPath string }

type AuthConfig struct{ JWTSecret string }

type RedisConfig struct {
	Addr     string
	Password string
}

type SearchConfig struct{ URL string }

type AnalyticsConfig struct {
	MongoAddr      string
	MongoUser      string
	MongoPassword  string
	RabbitAddr     string
	RabbitUser     string
	RabbitPassword string
}

type AssistantConfig struct {
	APIBaseURL string
	APIKey     string
	Model      string
}

type StorageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type AppConfig struct {
	Env string
}

type HTTPConfig struct {
	Host           string
	Port           int
	TrustedProxies string
}
type PostgresConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	DBName       string
	SSLMode      string
	TimeZone     string
	MaxOpenConns int
	MaxIdleConns int
}

func (c PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName,
		c.SSLMode,
		c.TimeZone,
	)
}

func Load() (*Config, error) {
	// 本地开发环境加载 .env。
	// 生产环境以后会直接使用系统环境变量。
	if os.Getenv("APP_ENV") == "" {
		if err := loadDotEnv(); err != nil {
			return nil, err
		}
	}

	v := viper.New()

	v.AutomaticEnv()

	// 显式绑定环境变量。
	//
	// 这样比单纯依赖 AutomaticEnv 更明确，
	// 以后也方便知道项目到底有哪些配置项。
	envKeys := []string{
		"APP_ENV",

		"HTTP_HOST",
		"HTTP_PORT",
		"HTTP_TRUSTED_PROXIES",
		"IP_REGION_DB_PATH",
		"SMTP_HOST", "SMTP_PORT", "SMTP_FROM", "SMTP_TLS_MODE", "SMTP_USERNAME", "SMTP_PASSWORD",
		"AUTH_PUBLIC_URL", "WEB_PUBLIC_URL", "GITHUB_CLIENT_ID", "GITHUB_CLIENT_SECRET", "GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET",

		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
		"POSTGRES_SSLMODE",
		"POSTGRES_TIMEZONE",
		"POSTGRES_MAX_OPEN_CONNS",
		"POSTGRES_MAX_IDLE_CONNS",
		"REDIS_ADDR",
		"REDIS_PASSWORD",
		"ELASTICSEARCH_URL",
		"MONGO_ADDR",
		"MONGO_ROOT_USERNAME",
		"MONGO_ROOT_PASSWORD",
		"RABBITMQ_ADDR",
		"RABBITMQ_USERNAME",
		"RABBITMQ_PASSWORD",
		"AUTH_JWT_SECRET",
		"ASSISTANT_API_BASE_URL",
		"ASSISTANT_API_KEY",
		"ASSISTANT_MODEL",
		"MINIO_ENDPOINT",
		"MINIO_ROOT_USER",
		"MINIO_ROOT_PASSWORD",
		"MINIO_BUCKET",
		"MINIO_USE_SSL",
	}

	for _, key := range envKeys {
		if err := v.BindEnv(key); err != nil {
			return nil, fmt.Errorf(
				"bind env %s: %w",
				key,
				err,
			)
		}
	}

	v.SetDefault("APP_ENV", "local")

	v.SetDefault("HTTP_HOST", "0.0.0.0")
	v.SetDefault("HTTP_PORT", 8080)

	v.SetDefault("POSTGRES_HOST", "127.0.0.1")
	v.SetDefault("POSTGRES_PORT", 5432)
	v.SetDefault("POSTGRES_SSLMODE", "disable")
	v.SetDefault("POSTGRES_TIMEZONE", "UTC")

	v.SetDefault("POSTGRES_MAX_OPEN_CONNS", 50)
	v.SetDefault("POSTGRES_MAX_IDLE_CONNS", 10)
	v.SetDefault("REDIS_ADDR", "127.0.0.1:6379")
	v.SetDefault("ELASTICSEARCH_URL", "http://127.0.0.1:9200")
	v.SetDefault("MONGO_ADDR", "127.0.0.1:27017")
	v.SetDefault("RABBITMQ_ADDR", "127.0.0.1:5672")
	v.SetDefault("MINIO_ENDPOINT", "127.0.0.1:9000")
	v.SetDefault("MINIO_BUCKET", "devhub-projects")
	v.SetDefault("SMTP_HOST", "127.0.0.1")
	v.SetDefault("SMTP_PORT", 1025)
	v.SetDefault("SMTP_FROM", "DevHub <no-reply@devhub.local>")
	v.SetDefault("SMTP_TLS_MODE", "none")
	v.SetDefault("AUTH_PUBLIC_URL", "http://localhost:5173")
	v.SetDefault("WEB_PUBLIC_URL", "http://localhost:5173")

	cfg := &Config{
		App: AppConfig{
			Env: v.GetString("APP_ENV"),
		},

		HTTP: HTTPConfig{
			Host:           v.GetString("HTTP_HOST"),
			Port:           v.GetInt("HTTP_PORT"),
			TrustedProxies: v.GetString("HTTP_TRUSTED_PROXIES"),
		},

		Postgres: PostgresConfig{
			Host: v.GetString("POSTGRES_HOST"),
			Port: v.GetInt("POSTGRES_PORT"),

			User:     v.GetString("POSTGRES_USER"),
			Password: v.GetString("POSTGRES_PASSWORD"),
			DBName:   v.GetString("POSTGRES_DB"),

			SSLMode:  v.GetString("POSTGRES_SSLMODE"),
			TimeZone: v.GetString("POSTGRES_TIMEZONE"),

			MaxOpenConns: v.GetInt("POSTGRES_MAX_OPEN_CONNS"),
			MaxIdleConns: v.GetInt("POSTGRES_MAX_IDLE_CONNS"),
		},
		Auth:   AuthConfig{JWTSecret: v.GetString("AUTH_JWT_SECRET")},
		Redis:  RedisConfig{Addr: v.GetString("REDIS_ADDR"), Password: v.GetString("REDIS_PASSWORD")},
		Search: SearchConfig{URL: v.GetString("ELASTICSEARCH_URL")},
		Analytics: AnalyticsConfig{
			MongoAddr: v.GetString("MONGO_ADDR"), MongoUser: v.GetString("MONGO_ROOT_USERNAME"), MongoPassword: v.GetString("MONGO_ROOT_PASSWORD"),
			RabbitAddr: v.GetString("RABBITMQ_ADDR"), RabbitUser: v.GetString("RABBITMQ_USERNAME"), RabbitPassword: v.GetString("RABBITMQ_PASSWORD"),
		},
		Assistant: AssistantConfig{
			APIBaseURL: v.GetString("ASSISTANT_API_BASE_URL"),
			APIKey:     v.GetString("ASSISTANT_API_KEY"),
			Model:      v.GetString("ASSISTANT_MODEL"),
		},
		Storage: StorageConfig{
			Endpoint:  v.GetString("MINIO_ENDPOINT"),
			AccessKey: v.GetString("MINIO_ROOT_USER"),
			SecretKey: v.GetString("MINIO_ROOT_PASSWORD"),
			Bucket:    v.GetString("MINIO_BUCKET"),
			UseSSL:    v.GetBool("MINIO_USE_SSL"),
		},
		GeoIP: GeoIPConfig{DBPath: v.GetString("IP_REGION_DB_PATH")},
		SMTP:  SMTPConfig{Host: v.GetString("SMTP_HOST"), Port: v.GetInt("SMTP_PORT"), From: v.GetString("SMTP_FROM"), TLSMode: v.GetString("SMTP_TLS_MODE"), Username: v.GetString("SMTP_USERNAME"), Password: v.GetString("SMTP_PASSWORD")},
		OAuth: OAuthConfig{PublicURL: v.GetString("AUTH_PUBLIC_URL"), WebURL: v.GetString("WEB_PUBLIC_URL"), GitHubClientID: v.GetString("GITHUB_CLIENT_ID"), GitHubClientSecret: v.GetString("GITHUB_CLIENT_SECRET"), GoogleClientID: v.GetString("GOOGLE_CLIENT_ID"), GoogleClientSecret: v.GetString("GOOGLE_CLIENT_SECRET")},
	}

	if cfg.Postgres.User == "" {
		return nil, fmt.Errorf("POSTGRES_USER is required")
	}

	if cfg.Postgres.Password == "" {
		return nil, fmt.Errorf("POSTGRES_PASSWORD is required")
	}

	if cfg.Postgres.DBName == "" {
		return nil, fmt.Errorf("POSTGRES_DB is required")
	}
	if len(cfg.Auth.JWTSecret) < 32 {
		return nil, fmt.Errorf("AUTH_JWT_SECRET must contain at least 32 characters")
	}
	if cfg.App.Env != "local" && cfg.App.Env != "test" && cfg.SMTP.TLSMode == "none" {
		return nil, fmt.Errorf("SMTP_TLS_MODE must be tls or starttls outside local development")
	}

	return cfg, nil
}

func loadDotEnv() error {
	paths := []string{".env", "../.env"}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		// 找到 .env 后加载。
		if err := godotenv.Load(path); err != nil {
			return fmt.Errorf("load env file %s: %w", path, err)
		}

		return nil
	}
	return fmt.Errorf(".env file not found")
}
