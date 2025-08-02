package configservice

import (
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/mcuadros/go-defaults"
	"github.com/samber/do"
	"github.com/spf13/viper"
)

// Config represents the configuration options for the service.
type Config struct {
	LogLevel string `mapstructure:"LOG_LEVEL" default:"INFO" validate:"required"`

	HTTPPort  int    `mapstructure:"HTTP_PORT"  default:"3000" validate:"required"`
	APIPrefix string `mapstructure:"API_PREFIX" default:"/api" validate:"required"`

	CookieSecure bool `mapstructure:"COOKIE_SECURE" default:"false"`

	JWTSecret      string `mapstructure:"JWT_SECRET"                validate:"required"`
	JWTExp         int    `mapstructure:"JWT_EXPIRATION" default:"3600" validate:"required"`
	RBACConfigPath string `mapstructure:"RBAC_CONFIG_PATH" default:"./configs/rbac.yaml" validate:"required"`

	DBHost string `mapstructure:"DB_HOST" validate:"required"`
	DBPort string `mapstructure:"DB_PORT" validate:"required"`
	DBUser string `mapstructure:"DB_USER" validate:"required"`
	DBPass string `mapstructure:"DB_PASS" validate:"required"`
	DBName string `mapstructure:"DB_NAME" validate:"required"`

	ValkeyAddress string `mapstructure:"VALKEY_ADDRESS" validate:"required"`

	OpenIDIssuer         string `mapstructure:"OPENID_ISSUER" validate:"required"`
	ConsentDayExpiration int    `mapstructure:"CONSENT_DAY_EXPIRATION" default:"30" validate:"required"`
	LoginURL             string `mapstructure:"LOGIN_URL" validate:"required"`

	IntraTokenURL     string `mapstructure:"INTRA_TOKEN_URL" validate:"required"`
	IntraAPIURL       string `mapstructure:"INTRA_API_URL" validate:"required"`
	IntraClientID     string `mapstructure:"INTRA_CLIENT_ID" validate:"required"`
	IntraClientSecret string `mapstructure:"INTRA_CLIENT_SECRET" validate:"required"`
}

// ConfigService is the interface for the config service.
type ConfigService interface {
	GetConfig() Config
}

type configService struct {
	config Config
}

func automaticBindEnv() {
	v := reflect.ValueOf(&Config{})
	t := v.Elem().Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		env := field.Tag.Get("mapstructure")
		if env == "" {
			continue
		}
		_ = viper.BindEnv(env)
	}
}

func NewProvider() func(i *do.Injector) (ConfigService, error) {
	return func(i *do.Injector) (ConfigService, error) {
		return New()
	}
}

// New creates a new instance of the config service.
// It reads the configuration from the environment variables.
// If the environment variables are not set, it uses the default values.
// It returns an error if the configuration is invalid.
func New() (ConfigService, error) {
	viper.AddConfigPath(".")
	viper.SetConfigType("env")
	viper.SetConfigName(".env")

	_ = viper.ReadInConfig()

	viper.AutomaticEnv()
	automaticBindEnv()

	config := &Config{}
	err := viper.Unmarshal(config)
	if err != nil {
		return nil, err
	}

	defaults.SetDefaults(config)

	err = validator.New().Struct(config)
	if err != nil {
		return nil, err
	}
	return &configService{
		config: *config,
	}, nil
}

// GetConfig returns the configuration options for the service.
func (c *configService) GetConfig() Config {
	return c.config
}
