package ldapsvc

import (
	"fmt"

	configsvc "github.com/froz42/kerbernetes/internal/services/config"
	"github.com/go-ldap/ldap/v3"
	"github.com/samber/do"
)

type LDAPSvc interface {
	// GetUser retrieves a user from LDAP by username
	GetUser(username string) (*ldap.Entry, error)
	
	// GetUserGroups retrieves groups for a user from LDAP
	GetUserGroups(dn string) ([]string, error)
}

type ldapSvc struct {
	config configsvc.Config
}

// Dependency injection provider
func NewProvider() func(i *do.Injector) (LDAPSvc, error) {
	return func(i *do.Injector) (LDAPSvc, error) {
		config := do.MustInvoke[configsvc.ConfigService](i).GetConfig()
		return New(config)
	}
}

// Create the service with just config
func New(config configsvc.Config) (LDAPSvc, error) {
	return &ldapSvc{
		config: config,
	}, nil
}

// GetUser retrieves a user from LDAP by username
func (s *ldapSvc) GetUser(username string) (*ldap.Entry, error) {
	var user *ldap.Entry
	err := s.withConnection(func(conn *ldap.Conn) error {
		searchRequest := ldap.NewSearchRequest(
			s.config.LDAPUserBaseDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			fmt.Sprintf(s.config.LDAPUserFilter, username),
			[]string{"dn", "cn"},
			nil,
		)

		result, err := conn.Search(searchRequest)
		if err != nil {
			return err
		}

		if len(result.Entries) == 0 {
			return ldap.NewError(ldap.LDAPResultNoSuchObject, nil)
		}

		if len(result.Entries) > 1 {
			return ldap.NewError(ldap.LDAPResultAmbiguousResponse, nil)
		}

		user = result.Entries[0]
		return nil
	})

	return user, err
}

// GetUserGroups retrieves groups for a user from LDAP
func (s *ldapSvc) GetUserGroups(dn string) ([]string, error) {
	var groups []string
	err := s.withConnection(func(conn *ldap.Conn) error {
		fmt.Printf(s.config.LDAPGroupFilter, dn)
		searchRequest := ldap.NewSearchRequest(
			s.config.LDAPGroupBaseDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			fmt.Sprintf(s.config.LDAPGroupFilter, dn),
			[]string{"dn", s.config.LDAPGroupNameAttribute},
			nil,
		)

		result, err := conn.Search(searchRequest)
		if err != nil {
			return err
		}

		for _, entry := range result.Entries {
			groups = append(groups, entry.GetAttributeValue(s.config.LDAPGroupNameAttribute))
		}

		return nil
	})

	return groups, err
}

// WithConnection handles connection setup, bind, and cleanup per operation
func (s *ldapSvc) withConnection(fn func(conn *ldap.Conn) error) error {
	conn, err := ldap.DialURL(s.config.LDAPURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.Bind(s.config.LDAPBindDN, s.config.LDAPBindPassword); err != nil {
		return err
	}

	return fn(conn)
}
