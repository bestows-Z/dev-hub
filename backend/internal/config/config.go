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
	Auth      AuthConfig
	Assistant AssistantConfig
	Storage   StorageConfig
}

type AuthConfig struct{ JWTSecret string }

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
	Host string
	Port int
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

		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
		"POSTGRES_SSLMODE",
		"POSTGRES_TIMEZONE",
		"POSTGRES_MAX_OPEN_CONNS",
		"POSTGRES_MAX_IDLE_CONNS",
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
	v.SetDefault("MINIO_ENDPOINT", "127.0.0.1:9000")
	v.SetDefault("MINIO_BUCKET", "devhub-projects")

	cfg := &Config{
		App: AppConfig{
			Env: v.GetString("APP_ENV"),
		},

		HTTP: HTTPConfig{
			Host: v.GetString("HTTP_HOST"),
			Port: v.GetInt("HTTP_PORT"),
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
		Auth: AuthConfig{JWTSecret: v.GetString("AUTH_JWT_SECRET")},
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
