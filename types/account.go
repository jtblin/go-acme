package types

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"

	log "github.com/Sirupsen/logrus"
	"github.com/xenolf/lego/acme"
)

// Account is used to store lets encrypt registration info
// and implements the acme.User interface.
type Account struct {
	Email              string
	Registration       *acme.RegistrationResource
	PrivateKey         []byte
	DomainsCertificate *DomainCertificate
}

// GetEmail returns email.
func (a Account) GetEmail() string {
	return a.Email
}

// GetRegistration returns lets encrypt registration resource.
func (a Account) GetRegistration() *acme.RegistrationResource {
	return a.Registration
}

// GetPrivateKey returns private key.
func (a Account) GetPrivateKey() crypto.PrivateKey {
	if privateKey, err := x509.ParsePKCS1PrivateKey(a.PrivateKey); err == nil {
		return privateKey
	}
	log.Errorf("Cannot unmarshall private key %+v", a.PrivateKey)
	return nil
}

// NewAccount creates a new account for the specified email and domain.
func NewAccount(email string, domain *Domain) (*Account, error) {
	// Create a user. New accounts need an email and private key to start
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}
	account := &Account{
		Email:      email,
		PrivateKey: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	account.DomainsCertificate = &DomainCertificate{
		Certificate: &Certificate{},
		Domain:      domain,
	}
	return account, nil
}
