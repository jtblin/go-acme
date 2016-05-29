package types

import (
	"crypto/tls"
	"errors"
	"reflect"
)

// Certificate is used to store certificate info.
type Certificate struct {
	Domain        string
	CertURL       string
	CertStableURL string
	PrivateKey    []byte
	Cert          []byte
}

// DomainCertificate contains a certificate for a domain and SANs.
type DomainCertificate struct {
	Certificate *Certificate
	Domain      *Domain
	TLSCert     *tls.Certificate `json:"-"`
}

// Domain holds a domain name with SANs.
type Domain struct {
	Main string
	SANs []string
}

func (dc *DomainCertificate) tlsCert() (*tls.Certificate, error) {
	cert, err := tls.X509KeyPair(dc.Certificate.Cert, dc.Certificate.PrivateKey)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

// Init initialises the tls certificate.
func (dc *DomainCertificate) Init() error {
	tlsCert, err := dc.tlsCert()
	if err != nil {
		return err
	}
	dc.TLSCert = tlsCert
	return nil
}

// RenewCertificate renew the certificate for the domain.
func (dc *DomainCertificate) RenewCertificate(acmeCert *Certificate, domain *Domain) error {
	if reflect.DeepEqual(domain, dc.Domain) {
		dc.Certificate = acmeCert
		if err := dc.Init(); err != nil {
			return err
		}
		return nil
	}
	return errors.New("Certificate to renew not found for domain " + domain.Main)
}

// AddCertificate add the certificate for the domain.
func (dc *DomainCertificate) AddCertificate(acmeCert *Certificate, domain *Domain) error {
	dc.Domain = domain
	dc.Certificate = acmeCert
	return dc.Init()
}
