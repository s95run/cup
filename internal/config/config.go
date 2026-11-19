package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var schemaNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type Config struct {
	Server       ServerConfig
	DB           DBConfig
	S95          S95Config
	Auth         AuthConfig
	Registration RegistrationConfig
	Sync         SyncConfig
}

type ServerConfig struct {
	Addr         string
	PublicURL    string
	CookieSecure bool
	CookieDomain string
}

type DBConfig struct {
	Host     string
	Port     int
	DBName   string
	User     string
	Password string
	SSLMode  string
	Schema   string
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&search_path=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode, d.Schema,
	)
}

type S95Config struct {
	BaseURL      string
	APIKey       string
	SSOSecret    string
	RateLimitRPS float64
	Mock         bool
}

type AuthConfig struct {
	JWTSecret         string
	JWTTTL            time.Duration
	DevAuthEnabled    bool
	AdminAthleteIDs []int64
}

type RegistrationConfig struct {
	DeadlineBeforeStage time.Duration
}

type SyncConfig struct {
	LocationsInterval    time.Duration
	ProtocolPollInterval time.Duration
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix("CUP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Addr:         v.GetString("server.addr"),
			PublicURL:    v.GetString("server.public_url"),
			CookieSecure: v.GetBool("server.cookie_secure"),
			CookieDomain: v.GetString("server.cookie_domain"),
		},
		DB: DBConfig{
			Host:     v.GetString("db.host"),
			Port:     v.GetInt("db.port"),
			DBName:   v.GetString("db.dbname"),
			User:     v.GetString("db.user"),
			Password: v.GetString("db.password"),
			SSLMode:  v.GetString("db.sslmode"),
			Schema:   v.GetString("db.schema"),
		},
		S95: S95Config{
			BaseURL:      v.GetString("s95.base_url"),
			APIKey:       v.GetString("s95.api_key"),
			SSOSecret:    v.GetString("s95.sso_secret"),
			RateLimitRPS: v.GetFloat64("s95.rate_limit_rps"),
			Mock:         v.GetBool("s95.mock"),
		},
		Auth: AuthConfig{
			JWTSecret:         v.GetString("auth.jwt_secret"),
			JWTTTL:            time.Duration(v.GetInt("auth.jwt_ttl_hours")) * time.Hour,
			DevAuthEnabled:    v.GetBool("auth.dev_auth_enabled"),
			AdminAthleteIDs: adminAthleteIDsFromViper(v),
		},
		Registration: RegistrationConfig{
			DeadlineBeforeStage: time.Duration(v.GetInt("registration.deadline_hours_before_stage")) * time.Hour,
		},
		Sync: SyncConfig{
			LocationsInterval:    time.Duration(v.GetInt("sync.locations_interval_hours")) * time.Hour,
			ProtocolPollInterval: time.Duration(v.GetInt("sync.protocol_poll_interval_minutes")) * time.Minute,
		},
	}

	if raw := os.Getenv("CUP_AUTH_ADMIN_ATHLETE_IDS"); raw != "" {
		cfg.Auth.AdminAthleteIDs = parseAthleteIDs(raw)
	} else if raw := os.Getenv("CUP_AUTH_ADMIN_ATHLETE_CODES"); raw != "" {
		cfg.Auth.AdminAthleteIDs = parseAthleteIDs(raw)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func adminAthleteIDsFromViper(v *viper.Viper) []int64 {
	if ids := intSliceToInt64(v.GetIntSlice("auth.admin_athlete_ids")); len(ids) > 0 {
		return ids
	}
	return intSliceToInt64(v.GetIntSlice("auth.admin_athlete_codes"))
}

func parseAthleteIDs(raw string) []int64 {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	})
	out := make([]int64, 0, len(parts))
	for _, part := range parts {
		code, err := strconv.ParseInt(part, 10, 64)
		if err != nil || code == 0 {
			continue
		}
		out = append(out, code)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func intSliceToInt64(in []int) []int64 {
	out := make([]int64, len(in))
	for i, v := range in {
		out[i] = int64(v)
	}
	return out
}

func (c *Config) Validate() error {
	if c.Server.Addr == "" {
		return fmt.Errorf("server.addr is required")
	}
	if c.DB.Schema == "" || !schemaNameRE.MatchString(c.DB.Schema) {
		return fmt.Errorf("db.schema must be a valid PostgreSQL identifier")
	}
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("auth.jwt_secret is required")
	}
	if len(c.Auth.JWTSecret) < 32 {
		return fmt.Errorf("auth.jwt_secret must be at least 32 bytes")
	}
	if c.S95.SSOSecret == "" {
		return fmt.Errorf("s95.sso_secret is required")
	}
	if len(c.S95.SSOSecret) < 32 {
		return fmt.Errorf("s95.sso_secret must be at least 32 bytes")
	}
	if c.S95.RateLimitRPS <= 0 {
		c.S95.RateLimitRPS = 5
	}
	return nil
}
