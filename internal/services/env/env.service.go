package envsvc

import (
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/mcuadros/go-defaults"
	"github.com/samber/do"
	"github.com/spf13/viper"
)

// Config represents the configuration options for the service.
type Env struct {
	HTTPPort   int    `mapstructure:"HTTP_PORT"  default:"3000" validate:"required"`
	APIPrefix  string `mapstructure:"API_PREFIX" default:"/api" validate:"required"`
	KeytabPath string `mapstructure:"KEYTAB_PATH" default:"/etc/krb5.keytab" validate:"required"`
	Namespace  string `mapstructure:"NAMESPACE" default:"default" validate:"required"`

	TokenDuration int    `mapstructure:"TOKEN_DURATION" default:"600" validate:"required"`
	TokenAudience string `mapstructure:"TOKEN_AUDIENCE" default:"https://kubernetes.default.svc.cluster.local"`

	LDAPEnabled bool   `mapstructure:"LDAP_ENABLED" default:"false"`
	LDAPURL     string `mapstructure:"LDAP_URL"`

	LDAPBindDN       string `mapstructure:"LDAP_BIND_DN"`
	LDAPBindPassword string `mapstructure:"LDAP_BIND_PASSWORD"`

	LDAPUserBaseDN string `mapstructure:"LDAP_USER_BASE_DN" default:"ou=users"`
	LDAPUserFilter string `mapstructure:"LDAP_USER_FILTER" default:"(uid=%s)"`

	LDAPGroupBaseDN string `mapstructure:"LDAP_GROUP_BASE_DN" default:"ou=groups"`
	LDAPGroupFilter string `mapstructure:"LDAP_GROUP_FILTER" default:"((member=%s)"`
}

// ConfigService is the interface for the config service.
type EnvSvc interface {
	GetEnv() Env
}

type configService struct {
	env Env
}

func automaticBindEnv() {
	v := reflect.ValueOf(&Env{})
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

func NewProvider() func(i *do.Injector) (EnvSvc, error) {
	return func(i *do.Injector) (EnvSvc, error) {
		return New()
	}
}

// New creates a new instance of the config service.
// It reads the configuration from the environment variables.
// If the environment variables are not set, it uses the default values.
// It returns an error if the configuration is invalid.
func New() (EnvSvc, error) {
	viper.AddConfigPath(".")
	viper.SetConfigType("env")
	viper.SetConfigName(".env")

	_ = viper.ReadInConfig()

	viper.AutomaticEnv()
	automaticBindEnv()

	env := &Env{}
	err := viper.Unmarshal(env)
	if err != nil {
		return nil, err
	}

	defaults.SetDefaults(env)

	err = validator.New().Struct(env)
	if err != nil {
		return nil, err
	}
	return &configService{
		env: *env,
	}, nil
}

// GetConfig returns the configuration options for the service.
func (c *configService) GetEnv() Env {
	return c.env
}
