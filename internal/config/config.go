package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPPort           string
	DB                 DBConfig
	RedisHost          string
	RedisPort          string
	JWTSecret          string
	JWTTTL             time.Duration
	AssetsDir          string
	WebDir             string
	KafkaBrokers       string
	KafkaTopicMessages string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (db DBConfig) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		db.User, db.Password, db.Host, db.Port, db.Name, db.SSLMode,
	)
}

func getEnv(key, defaultVal string) string {
	env := os.Getenv(key)
	if env == "" {
		return defaultVal
	}
	return env
}

func Load() *Config {
	DB := DBConfig{
		Host:     getEnv("POSTGRES_HOST", "localhost"),
		Port:     getEnv("POSTGRES_PORT", "5432"),
		User:     getEnv("POSTGRES_USER", "didok362"),
		Password: getEnv("POSTGRES_PASSWORD", "qwerty67"),
		Name:     getEnv("POSTGRES_DB", "messenger_db"),
		SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
	}

	ttl, err := time.ParseDuration(getEnv("JWT_TTL", "24h"))
	if err != nil {
		ttl = 24 * time.Hour
	}

	return &Config{
		HTTPPort:           getEnv("HTTP_PORT", "8080"),
		DB:                 DB,
		JWTSecret:          getEnv("JWT_SECRET", "super-secret-jwt-key"),
		JWTTTL:             ttl,
		AssetsDir:          getEnv("ASSETS_DIR", "assets/avatars"),
		WebDir:             getEnv("WEB_DIR", "web"),
		RedisHost:          getEnv("REDIS_HOST", "localhost"),
		RedisPort:          getEnv("REDIS_PORT", "6379"),
		KafkaBrokers:       getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopicMessages: getEnv("KAFKA_TOPIC_MESSAGES", "chat.messages"),
	}
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func (c *Config) KafkaBrokersList() []string {
	return strings.Split(c.KafkaBrokers, ",")
}
