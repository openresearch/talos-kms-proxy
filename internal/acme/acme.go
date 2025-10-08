package acme

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"time"

	"github.com/go-acme/lego/v4/acme/api"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns"
	"github.com/go-acme/lego/v4/registration"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Acme struct {
	domains      []string
	email        string
	dev          bool
	workdir      string
	certsChannel chan map[string][]byte
	client       *lego.Client
	certs        map[string][]byte
	user         *AcmeUser
	timer        *time.Timer
}

type AcmeUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
	PKB          []byte
}

var (
	logger = log.With().Str("service", "acme").Logger().Output(zerolog.ConsoleWriter{Out: os.Stdout})
)

func (u *AcmeUser) GetEmail() string {
	return u.Email
}
func (u AcmeUser) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *AcmeUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func (a *Acme) Stop() {
	// not used
}

func New(domains []string, email, dir string, dev bool, certsChannel chan map[string][]byte) *Acme {

	return &Acme{
		domains:      domains,
		email:        email,
		dev:          dev,
		workdir:      dir,
		certsChannel: certsChannel,
		user:         &AcmeUser{},
	}
}

func (a *Acme) Serve(ctx context.Context) error {

	logger.Info().Msg("starting")

	if err := a.restore(); errors.Is(err, os.ErrNotExist) {
		logger.Debug().Msg("acme user not found - creating new user")

		// create new acme user
		if err := a.initLego(); err != nil {
			return err
		}

		// create new certificate
		if err := a.createCertificate(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// start renewal process
	a.timer = time.NewTimer(time.Until(time.Now().Local().Add(5 * time.Minute)))
	go a.renewCertificate()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-a.timer.C:
			logger.Debug().Msgf("certificate renewal timer triggered")

			if err := a.createCertificate(); err != nil {
				return err
			}

			// start renewal process again
			go a.renewCertificate()
		}
	}
}

// initLego prepares the lego user and client
// * creates a new private key
// * creates a new ACME registration
// * inits new lego client
// * creates new acme account registration
// * writes the account state data to filesystem
func (a *Acme) initLego() error {

	logger.Debug().Msgf("creating new lego account for %s", a.email)

	// Create a user. New accounts need an email and private key to start.
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	pkb, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return err
	}
	user := AcmeUser{
		Email: a.email,
		key:   privateKey,
		PKB:   pkb,
	}
	a.user = &user

	// create new lego client
	client, err := a.createLegoClient()
	if err != nil {
		return err
	}
	a.client = client

	// New users will need to register
	reg, err := a.client.Registration.Register(
		registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return err
	}
	a.user.Registration = reg

	if err := a.writeState(); err != nil {
		return err
	}

	logger.Debug().Msgf("successfully registered acme account for %s", a.email)

	return nil
}

// createCertificate will create new certificates
// * creates new DNS challenge
// * obtains certificate request
// * obtains certificates
// * send certs to KMS service (via channel)
// * pass certs to writeCerts method
func (a *Acme) createCertificate() error {

	logger.Debug().Msgf("creating new certs for %s", a.domains)

	// setup new cloudflare provider and dns challange
	provider, err := dns.NewDNSChallengeProviderByName("route53")
	if err != nil {
		return err
	}
	if err := a.client.Challenge.SetDNS01Provider(provider,
		dns01.CondOption(true, dns01.DisableAuthoritativeNssPropagationRequirement()),
	); err != nil {
		return err
	}

	// create new certificate request
	request := certificate.ObtainRequest{
		Domains: a.domains,
		Bundle:  true,
	}

	// obtain certificate
	certificates, err := a.client.Certificate.Obtain(request)
	if err != nil {
		return err
	}

	// send new certificate and private key to kms server
	certsMap := map[string][]byte{
		"certificate": certificates.Certificate,
		"privatekey":  certificates.PrivateKey,
	}
	a.certsChannel <- certsMap
	a.certs = certsMap

	if err := a.writeCerts(certificates); err != nil {
		return err
	}

	logger.Info().Msgf("successfully created new certs for: %s", a.domains)

	return nil
}

// renewCertificate renews certificates
func (a *Acme) renewCertificate() {

	cert, err := tls.X509KeyPair(a.certs["certificate"], a.certs["privatekey"])
	if err != nil {
		logger.Error().Err(err).Msg("convert cert to x509")
	}

	renewalInfo, err := a.client.Certificate.GetRenewalInfo(
		certificate.RenewalInfoRequest{
			Cert: cert.Leaf,
		},
	)
	if err != nil {
		if errors.Is(err, api.ErrNoARI) {
			// The server does not advertise a renewal info endpoint.
			logger.Warn().Msgf("acme renew: %v", err)
		}
		logger.Warn().Msgf("acme: calling renewal info endpoint: %v", err)
	}

	// get the renewal suggestion time and reset timer to that time
	renew := renewalInfo.RenewalInfoResponse.SuggestedWindow.Start
	a.timer.Reset(time.Until(renew))
	logger.Info().Msgf("reattempting certificate renewal at: %s", renew)
}

func (a *Acme) createLegoClient() (*lego.Client, error) {

	config := lego.NewConfig(a.user)
	if a.dev {
		logger.Debug().Msg("running in debug mode, using Let's Encrypt staging server")
		config.CADirURL = lego.LEDirectoryStaging
	} else {
		config.CADirURL = lego.LEDirectoryProduction
	}
	config.Certificate.KeyType = certcrypto.EC256

	// A client facilitates communication with the CA server.
	client, err := lego.NewClient(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}
