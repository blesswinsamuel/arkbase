package config

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Server        ServerConfig                  `yaml:"server"`
	Destinations  map[string]DestinationConfig  `yaml:"destinations"`
	Databases     map[string]DatabaseConfig     `yaml:"databases"`
	Notifications []NotificationConfig          `yaml:"notifications"`
}

type ServerConfig struct {
	Port    int        `yaml:"port"`
	DataDir string     `yaml:"data_dir"`
	Auth    AuthConfig `yaml:"auth"`
}

type AuthConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type DestinationType string

const (
	DestinationTypeFilesystem DestinationType = "filesystem"
	DestinationTypeS3         DestinationType = "s3"
)

type DestinationConfig struct {
	Type        DestinationType  `yaml:"type"`
	Path        string           `yaml:"path"`
	Retention   *RetentionPolicy `yaml:"retention"`
	Encryption  *EncryptionConfig `yaml:"encryption"`

	// S3 specific options
	Endpoint    string `yaml:"endpoint"`
	Bucket      string `yaml:"bucket"`
	Region      string `yaml:"region"`
	AccessKey   string `yaml:"access_key"`
	SecretKey   string `yaml:"secret_key"`
	Insecure    bool   `yaml:"insecure"`
}

type EncryptionConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Passphrase string `yaml:"passphrase"`
}

type RetentionPolicy struct {
	KeepLast int `yaml:"keep_last" json:"keep_last"` // Keep the last N backups
	Hourly   int `yaml:"hourly" json:"hourly"`       // Keep hourly backups for N hours
	Daily    int `yaml:"daily" json:"daily"`         // Keep daily backups for N days
	Weekly   int `yaml:"weekly" json:"weekly"`       // Keep weekly backups for N weeks
	Monthly  int `yaml:"monthly" json:"monthly"`     // Keep monthly backups for N months
	Yearly   int `yaml:"yearly" json:"yearly"`       // Keep yearly backups for N years
}

type DatabaseConfig struct {
	Engine       string        `yaml:"engine"` // "postgres", etc.
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	Username     string        `yaml:"username"`
	Password     string        `yaml:"password"`
	Database     string        `yaml:"database"`
	URI          string        `yaml:"uri"`
	SSLMode      string        `yaml:"sslmode"`
	Schedule     string        `yaml:"schedule"` // Cron expression
	Destinations []string      `yaml:"destinations"`
	Options      []string      `yaml:"options"`
	Timeout      time.Duration `yaml:"timeout"`
}

type NotificationConfig struct {
	Name       string   `yaml:"name"`
	Type       string   `yaml:"type"` // "webhook", "discord", "telegram"
	URL        string   `yaml:"url"`
	WebhookURL string   `yaml:"webhook_url"`
	BotToken   string   `yaml:"bot_token"`
	ChatID     string   `yaml:"chat_id"`
	OnEvents   []string `yaml:"on_events"` // "success", "failure"
}

var envVarPattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)(?::-(.*?))?\}`)

// ExpandEnv replaces ${VAR} or ${VAR:-default} with environment variable values.
func ExpandEnv(input string) string {
	return envVarPattern.ReplaceAllStringFunc(input, func(m string) string {
		submatch := envVarPattern.FindStringSubmatch(m)
		if len(submatch) >= 2 {
			varName := submatch[1]
			defaultVal := ""
			if len(submatch) >= 3 {
				defaultVal = submatch[2]
			}
			if val, ok := os.LookupEnv(varName); ok {
				return val
			}
			return defaultVal
		}
		return m
	})
}

func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	expanded := ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml config: %w", err)
	}

	// Apply defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.DataDir == "" {
		cfg.Server.DataDir = "/var/lib/arkbase"
	}

	for name, db := range cfg.Databases {
		if db.Engine == "" {
			db.Engine = "postgres"
		}
		if db.Port == 0 && db.Engine == "postgres" {
			db.Port = 5432
		}
		if db.Timeout == 0 {
			db.Timeout = 30 * time.Minute
		}
		cfg.Databases[name] = db
	}

	return &cfg, nil
}
