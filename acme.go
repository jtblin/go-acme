package acme

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/jtblin/go-logger"
	"github.com/xenolf/lego/acme"

	"github.com/jtblin/go-acme/backend"
	_ "github.com/jtblin/go-acme/backend/backends" // import all backends.
	"github.com/jtblin/go-acme/types"
)

const (
	// #2 - important set to true to bundle CA with certificate and
	// avoid "transport: x509: certificate signed by unknown authority" error
	bundleCA        = true
	defaultCAServer = "https://acme-v01.api.letsencrypt.org/directory"
)

// ACME allows to connect to lets encrypt and retrieve certs.
type ACME struct {
	backend     backend.Interface
	Domain      *types.Domain
	Logger      logger.Interface
	BackendName string
	CAServer    string
	DNSProvider string
	Email       string
	SelfSigned  bool
}

func (a *ACME) retrieveCertificate(client *acme.Client, account *types.Account) (*tls.Certificate, error) {
	a.Logger.Println("Retrieving ACME certificate...")
	domain := []string{}
	domain = append(domain, a.Domain.Main)
	domain = append(domain, a.Domain.SANs...)
	certificate, err := a.getDomainCertificate(client, domain)
	if err != nil {
		return nil, fmt.Errorf("Error getting ACME certificate for domain %s: %s", domain, err.Error())
	}
	if err = account.DomainsCertificate.AddCertificate(certificate, a.Domain); err != nil {
		return nil, fmt.Errorf("Error adding ACME certificate for domain %s: %s", domain, err.Error())
	}
	if err = a.backend.SaveAccount(account); err != nil {
		return nil, fmt.Errorf("Error Saving ACME account %+v: %s", account, err.Error())
	}
	a.Logger.Println("Retrieved ACME certificate")
	return account.DomainsCertificate.TLSCert, nil
}

func needsUpdate(cert *tls.Certificate) bool {
	// Leaf will be nil because the parsed form of the certificate is not retained
	// so we need to parse the certificate manually.
	for _, c := range cert.Certificate {
		crt, err := x509.ParseCertificate(c)
		// If there's an error, we assume the cert is broken, and needs update.
		// <= 7 days left, renew certificate.
		if err != nil || crt.NotAfter.Before(time.Now().Add(24*7*time.Hour)) {
			return true
		}
	}
	return false
}

func (a *ACME) renewCertificate(client *acme.Client, account *types.Account) error {
	dc := account.DomainsCertificate
	if needsUpdate(dc.TLSCert) {
		renewedCert, err := client.RenewCertificate(acme.CertificateResource{
			Domain:        dc.Certificate.Domain,
			CertURL:       dc.Certificate.CertURL,
			CertStableURL: dc.Certificate.CertStableURL,
			PrivateKey:    dc.Certificate.PrivateKey,
			Certificate:   dc.Certificate.Cert,
		}, false)
		if err != nil {
			return err
		}
		renewedACMECert := &types.Certificate{
			Domain:        renewedCert.Domain,
			CertURL:       renewedCert.CertURL,
			CertStableURL: renewedCert.CertStableURL,
			PrivateKey:    renewedCert.PrivateKey,
			Cert:          renewedCert.Certificate,
		}
		err = dc.RenewCertificate(renewedACMECert, dc.Domain)
		if err != nil {
			return err
		}
		if err = a.backend.SaveAccount(account); err != nil {
			return err
		}
	}
	return nil
}

func (a *ACME) buildACMEClient(Account *types.Account) (*acme.Client, error) {
	caServer := defaultCAServer
	if len(a.CAServer) > 0 {
		caServer = a.CAServer
	}
	client, err := acme.NewClient(caServer, Account, acme.RSA4096)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (a *ACME) getDomainCertificate(client *acme.Client, domains []string) (*types.Certificate, error) {
	certificate, failures := client.ObtainCertificate(domains, bundleCA, nil)
	if len(failures) > 0 {
		return nil, fmt.Errorf("Cannot obtain certificates %s+v", failures)
	}
	a.Logger.Printf("Loaded ACME certificates %s\n", domains)
	return &types.Certificate{
		Domain:        certificate.Domain,
		CertURL:       certificate.CertURL,
		CertStableURL: certificate.CertStableURL,
		PrivateKey:    certificate.PrivateKey,
		Cert:          certificate.Certificate,
	}, nil
}

// CreateConfig creates a tls.config from using ACME configuration
func (a *ACME) CreateConfig(tlsConfig *tls.Config) error {
	if a.Logger == nil {
		a.Logger = log.New(os.Stdout, "[go-acme] ", log.Ldate|log.Ltime|log.Lshortfile)
	}
	if a.Domain == nil || a.Domain.Main == "" {
		a.Logger.Panic("The main domain name must be provided")
	}
	if a.SelfSigned {
		a.Logger.Println("Generating self signed certificate...")
		cert, err := generateSelfSignedCertificate(a.Domain.Main)
		if err != nil {
			return err
		}
		tlsConfig.Certificates = []tls.Certificate{*cert}
		return nil
	}

	acme.Logger = log.New(ioutil.Discard, "", 0)

	if a.BackendName == "" {
		a.BackendName = "fs"
	}
	b, err := backend.InitBackend(a.BackendName)
	if err != nil {
		return err
	}
	a.backend = b

	var account *types.Account
	var needRegister bool

	a.Logger.Println("Loading ACME certificate...")
	account, err = a.backend.LoadAccount(a.Domain.Main)
	if err != nil {
		return err
	}
	if account != nil {
		a.Logger.Printf("Loaded ACME config from storage %q\n", a.backend.Name())
		if err = account.DomainsCertificate.Init(); err != nil {
			return err
		}
	} else {
		a.Logger.Println("Generating ACME Account...")
		account, err = types.NewAccount(a.Email, a.Domain, a.Logger)
		if err != nil {
			return err
		}
		needRegister = true
	}

	client, err := a.buildACMEClient(account)
	if err != nil {
		return err
	}
	client.ExcludeChallenges([]acme.Challenge{acme.HTTP01, acme.TLSSNI01})
	provider, err := newDNSProvider(a.DNSProvider)
	if err != nil {
		return err
	}
	client.SetChallengeProvider(acme.DNS01, provider)

	if needRegister {
		// New users need to register.
		reg, err := client.Register()
		if err != nil {
			return err
		}
		account.Registration = reg

		// The client has a URL to the current Let's Encrypt Subscriber
		// Agreement. The user needs to agree to it.
		err = client.AgreeToTOS()
		if err != nil {
			return err
		}
	}

	dc := account.DomainsCertificate
	if len(dc.Certificate.Cert) > 0 && len(dc.Certificate.PrivateKey) > 0 {
		go func() {
			if err := a.renewCertificate(client, account); err != nil {
				a.Logger.Printf("Error renewing ACME certificate for %q: %s\n",
					account.DomainsCertificate.Domain.Main, err.Error())
			}
		}()
	} else {
		if _, err := a.retrieveCertificate(client, account); err != nil {
			return err
		}
	}
	tlsConfig.GetCertificate = func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		if clientHello.ServerName != a.Domain.Main {
			return nil, errors.New("Unknown server name")
		}
		return dc.TLSCert, nil
	}
	a.Logger.Println("Loaded certificate...")

	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			if err := a.renewCertificate(client, account); err != nil {
				a.Logger.Printf("Error renewing ACME certificate %q: %s\n",
					account.DomainsCertificate.Domain.Main, err.Error())
			}
		}
	}()
	return nil
}
