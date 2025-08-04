package ldapsvc

import (
	"fmt"

	envsvc "github.com/froz42/kerbernetes/internal/services/env"
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
	env envsvc.Env
}

func NewProvider() func(i *do.Injector) (LDAPSvc, error) {
	return func(i *do.Injector) (LDAPSvc, error) {
		config := do.MustInvoke[envsvc.EnvSvc](i).GetEnv()
		return New(config)
	}
}

func New(env envsvc.Env) (LDAPSvc, error) {
	return &ldapSvc{
		env: env,
	}, nil
}

// GetUser retrieves a user from LDAP by username
func (s *ldapSvc) GetUser(username string) (*ldap.Entry, error) {
	var user *ldap.Entry
	err := s.withConnection(func(conn *ldap.Conn) error {
		searchRequest := ldap.NewSearchRequest(
			s.env.LDAPUserBaseDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			fmt.Sprintf(s.env.LDAPUserFilter, username),
			[]string{"dn"},
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
		fmt.Printf(s.env.LDAPGroupFilter, dn)
		searchRequest := ldap.NewSearchRequest(
			s.env.LDAPGroupBaseDN,
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			fmt.Sprintf(s.env.LDAPGroupFilter, dn),
			[]string{"dn"},
			nil,
		)

		result, err := conn.Search(searchRequest)
		if err != nil {
			return err
		}

		for _, entry := range result.Entries {
			groups = append(groups, entry.DN)
		}

		return nil
	})

	return groups, err
}

// WithConnection handles connection setup, bind, and cleanup per operation
func (s *ldapSvc) withConnection(fn func(conn *ldap.Conn) error) error {
	conn, err := ldap.DialURL(s.env.LDAPURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.Bind(s.env.LDAPBindDN, s.env.LDAPBindPassword); err != nil {
		return err
	}

	return fn(conn)
}
