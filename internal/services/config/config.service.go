package configsvc

import (
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/mcuadros/go-defaults"
	"github.com/samber/do"
	"github.com/spf13/viper"
)

// Config represents the configuration options for the service.
type Config struct {
	HTTPPort   int    `mapstructure:"HTTP_PORT"  default:"3000" validate:"required"`
	APIPrefix  string `mapstructure:"API_PREFIX" default:"/api" validate:"required"`
	KeytabPath string `mapstructure:"KEYTAB_PATH" default:"/etc/krb5.keytab" validate:"required"`

	TokenDuration int `mapstructure:"TOKEN_DURATION" default:"600" validate:"required"`
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
