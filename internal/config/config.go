package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	App        AppConfig        `mapstructure:"app"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Centrifugo CentrifugoConfig `mapstructure:"centrifugo"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Logging    LoggingConfig    `mapstructure:"logging"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// DSN returns the PostgreSQL connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Ho_Chi_Minh",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

type RedisConfig struct {
	URL string `mapstructure:"url"`
}

type CentrifugoConfig struct {
	URL           string `mapstructure:"url"`
	APIKey        string `mapstructure:"api_key"`
	HMACSecretKey string `mapstructure:"hmac_secret_key"`
}

type JWTConfig struct {
	Secret          string        `mapstructure:"secret"`
	AccessDuration  time.Duration `mapstructure:"access_duration"`
	RefreshDuration time.Duration `mapstructure:"refresh_duration"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// IsProduction checks if app is in production mode
func (c *AppConfig) IsProduction() bool {
	return c.Env == "production"
}

// IsDevelopment checks if app is in development mode
func (c *AppConfig) IsDevelopment() bool {
	return c.Env == "development"
}

// Load reads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	// Load .env file if exists
	_ = godotenv.Load()

	v := viper.New()

	// Set config file
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Bind environment variables - this allows ENV vars to override config
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values from config file with env expansion
	cfg := &Config{
		App: AppConfig{
			Name: getEnvOrDefault("APP_NAME", v.GetString("app.name")),
			Env:  getEnvOrDefault("APP_ENV", v.GetString("app.env")),
			Port: getEnvOrDefaultInt("APP_PORT", v.GetInt("app.port")),
		},
		Database: DatabaseConfig{
			Host:            getEnvOrDefault("DB_HOST", v.GetString("database.host")),
			Port:            getEnvOrDefaultInt("DB_PORT", v.GetInt("database.port")),
			User:            getEnvOrDefault("DB_USER", v.GetString("database.user")),
			Password:        getEnvOrDefault("DB_PASSWORD", v.GetString("database.password")),
			Name:            getEnvOrDefault("DB_NAME", v.GetString("database.name")),
			SSLMode:         getEnvOrDefault("DB_SSL_MODE", v.GetString("database.ssl_mode")),
			MaxOpenConns:    v.GetInt("database.max_open_conns"),
			MaxIdleConns:    v.GetInt("database.max_idle_conns"),
			ConnMaxLifetime: v.GetDuration("database.conn_max_lifetime"),
		},
		Redis: RedisConfig{
			URL: getEnvOrDefault("REDIS_URL", v.GetString("redis.url")),
		},
		Centrifugo: CentrifugoConfig{
			URL:           getEnvOrDefault("CENTRIFUGO_URL", v.GetString("centrifugo.url")),
			APIKey:        getEnvOrDefault("CENTRIFUGO_API_KEY", v.GetString("centrifugo.api_key")),
			HMACSecretKey: getEnvOrDefault("CENTRIFUGO_HMAC_SECRET_KEY", v.GetString("centrifugo.hmac_secret_key")),
		},
		JWT: JWTConfig{
			Secret:          getEnvOrDefault("JWT_SECRET", v.GetString("jwt.secret")),
			AccessDuration:  v.GetDuration("jwt.access_duration"),
			RefreshDuration: v.GetDuration("jwt.refresh_duration"),
		},
		Logging: LoggingConfig{
			Level:  getEnvOrDefault("LOG_LEVEL", v.GetString("logging.level")),
			Format: getEnvOrDefault("LOG_FORMAT", v.GetString("logging.format")),
		},
	}

	// Set defaults
	if cfg.App.Port == 0 {
		cfg.App.Port = 8080
	}
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 25
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 5
	}
	if cfg.Database.ConnMaxLifetime == 0 {
		cfg.Database.ConnMaxLifetime = 5 * time.Minute
	}
	if cfg.JWT.AccessDuration == 0 {
		cfg.JWT.AccessDuration = 15 * time.Minute
	}
	if cfg.JWT.RefreshDuration == 0 {
		cfg.JWT.RefreshDuration = 168 * time.Hour
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

// getEnvOrDefault returns env value or default
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	// Handle ${VAR:default} pattern in defaultVal
	if strings.HasPrefix(defaultVal, "${") && strings.HasSuffix(defaultVal, "}") {
		inner := defaultVal[2 : len(defaultVal)-1]
		parts := strings.SplitN(inner, ":", 2)
		if len(parts) == 2 {
			return parts[1]
		}
		return ""
	}
	return defaultVal
}

// getEnvOrDefaultInt returns env value as int or default
func getEnvOrDefaultInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var intVal int
		fmt.Sscanf(val, "%d", &intVal)
		if intVal > 0 {
			return intVal
		}
	}
	if defaultVal > 0 {
		return defaultVal
	}
	return 0
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.App.Port <= 0 || c.App.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.App.Port)
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	return nil
}

// LoadConfig is a helper function for backward compatibility
func LoadConfig() (*Config, error) {
	return Load("configs/config.yaml")
}