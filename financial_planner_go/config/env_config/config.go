package envconfig

import (
	"os"
	"strconv"
)

type Config struct {
	Env       string
	App       AppSettings
	Database  DatabaseSettings
	Security  SecuritySettings
	RateLimit RateLimitSettings
	Network   NetworkSettings
	Mail      MailSettings
	Brand     BrandSettings
	S3        S3Settings
}

type AppSettings struct {
	Port     string
	LogLevel string
}

type DatabaseSettings struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
	TimeZone string
}

type SecuritySettings struct {
	Enabled              bool
	APIKey               string
	IPWhitelistEnabled   bool
	IPWhitelist          string
	ReportSigningSecret  string
	ReportLinkTTLSeconds int
}

type RateLimitSettings struct {
	Enabled           bool
	RequestsPerMinute int
	Burst             int
}

type NetworkSettings struct {
	TrustedProxies string
}

type MailSettings struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
	UseTLS    bool
	UseSSL    bool
}

type BrandSettings struct {
	LogoPath string
}

type S3Settings struct {
	Endpoint     string
	Region       string
	AccessKey    string
	SecretKey    string
	BucketName   string
	PublicFolder string
}

var AppConfig *Config

func Load() {
	reportLinkTTLSeconds, err := StringToInt(
		os.Getenv("REPORT_LINK_TTL_SECONDS"),
	)
	if err != nil {
		panic("invalid REPORT_LINK_TTL_SECONDS: " + err.Error())
	}

	requestsPerMinute, err := StringToInt(
		os.Getenv("REPORT_RATE_LIMIT_PER_MINUTE"),
	)
	if err != nil {
		panic("invalid REPORT_RATE_LIMIT_PER_MINUTE: " + err.Error())
	}

	rateLimitBurst, err := StringToInt(
		os.Getenv("REPORT_RATE_LIMIT_BURST"),
	)
	if err != nil {
		panic("invalid REPORT_RATE_LIMIT_BURST: " + err.Error())
	}

	securityEnabled, err := StringToBool(
		os.Getenv("SECURITY_ENABLED"),
	)
	if err != nil {
		panic("invalid SECURITY_ENABLED: " + err.Error())
	}

	ipWhitelistEnabled, err := StringToBool(
		os.Getenv("IP_WHITELIST_ENABLED"),
	)
	if err != nil {
		panic("invalid IP_WHITELIST_ENABLED: " + err.Error())
	}

	mailPort, err := StringToInt(os.Getenv("BOLTPE_SMTP_PORT"))
	if err != nil {
		panic("invalid BOLTPE_SMTP_PORT: " + err.Error())
	}
	mailTLS, err := StringToBool(os.Getenv("BOLTPE_SMTP_USE_TLS"))
	if err != nil {
		panic("invalid BOLTPE_SMTP_USE_TLS: " + err.Error())
	}
	mailSSL, err := StringToBool(os.Getenv("BOLTPE_SMTP_USE_SSL"))
	if err != nil {
		panic("invalid BOLTPE_SMTP_USE_SSL: " + err.Error())
	}

	rateLimitEnabled, err := StringToBool(
		os.Getenv("RATE_LIMIT_ENABLED"),
	)
	if err != nil {
		panic("invalid RATE_LIMIT_ENABLED: " + err.Error())
	}

	AppConfig = &Config{
		Env: os.Getenv("ENV"),

		App: AppSettings{
			Port:     os.Getenv("APP_PORT"),
			LogLevel: os.Getenv("LOG_LEVEL"),
		},

		Database: DatabaseSettings{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			Name:     os.Getenv("DB_NAME"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
			TimeZone: os.Getenv("DB_TIMEZONE"),
		},

		Security: SecuritySettings{
			Enabled:              securityEnabled,
			APIKey:               os.Getenv("API_KEY"),
			IPWhitelistEnabled:   ipWhitelistEnabled,
			IPWhitelist:          os.Getenv("IP_WHITELIST"),
			ReportSigningSecret:  os.Getenv("REPORT_SIGNING_SECRET"),
			ReportLinkTTLSeconds: reportLinkTTLSeconds,
		},

		RateLimit: RateLimitSettings{
			Enabled:           rateLimitEnabled,
			RequestsPerMinute: requestsPerMinute,
			Burst:             rateLimitBurst,
		},

		Network: NetworkSettings{
			TrustedProxies: os.Getenv("TRUSTED_PROXIES"),
		},
		Mail: MailSettings{
			Host:      os.Getenv("BOLTPE_SMTP_HOST"),
			Port:      mailPort,
			Username:  os.Getenv("BOLTPE_SMTP_USERNAME"),
			Password:  os.Getenv("BOLTPE_SMTP_PASSWORD"),
			FromEmail: os.Getenv("BOLTPE_SMTP_FROM_EMAIL"),
			FromName:  os.Getenv("BOLTPE_SMTP_FROM_NAME"),
			UseTLS:    mailTLS,
			UseSSL:    mailSSL,
		},
		Brand: BrandSettings{LogoPath: os.Getenv("BOLTPE_LOGO_PATH")},
		S3:    S3Settings{Endpoint: os.Getenv("S3_ENDPOINT"), Region: os.Getenv("S3_REGION"), AccessKey: os.Getenv("S3_ACCESS_KEY"), SecretKey: os.Getenv("S3_SECRET_KEY"), BucketName: os.Getenv("S3_BUCKET_NAME"), PublicFolder: os.Getenv("S3_PUBLIC_FOLDER")},
	}
}

func StringToInt(value string) (int, error) {
	return strconv.Atoi(value)
}

func StringToBool(value string) (bool, error) {
	return strconv.ParseBool(value)
}
