package acme

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/rs/zerolog/log"
)

// restore loads the existing account state and certificates to memory
func (a *Acme) restore() error {

	log.Debug().Msgf("restoring user information")

	state, err := os.ReadFile(fmt.Sprintf("%s/state.json", a.workdir))
	if err != nil {
		return err
	}

	if err := json.Unmarshal(state, a.user); err != nil {
		return err
	}

	// restore private key from DER format
	pk, err := x509.ParseECPrivateKey(a.user.PKB)
	if err != nil {
		return err
	}
	a.user.key = pk

	// load certs
	if err := a.loadCerts(); err != nil {
		return err
	}

	// load lego client
	client, err := a.createLegoClient()
	if err != nil {
		return err
	}
	a.client = client

	return nil
}

// loadCerts reads the certificates from filesystem and stores them into
// the certificates map
func (a *Acme) loadCerts() error {

	certs := make(map[string][]byte)

	// try to read cert.pem
	crt, err := os.ReadFile(fmt.Sprintf("%s/certs/cert.pem", a.workdir))
	if err != nil {
		return err
	}
	certs["certificate"] = crt

	// try to read key.pem
	pk, err := os.ReadFile(fmt.Sprintf("%s/certs/key.pem", a.workdir))
	if err != nil {
		return err
	}
	certs["privatekey"] = pk

	// assign existing certs to our server instance
	a.certs = certs

	log.Debug().Msgf("successfully loaded certs from %s/certs", a.workdir)

	return nil
}

// writeCerts sends the new certificates to kms service via a channel
// and it saves the certificates to the filesystem
func (a *Acme) writeCerts(certificates *certificate.Resource) error {

	// create certificates directory
	// write certificate and private key to filesystem
	//
	if err := os.MkdirAll(fmt.Sprintf("%s/certs", a.workdir), 0750); err != nil {
		log.Error().Err(err).Msg("")
	}

	// write metadata
	cmd, err := json.Marshal(certificates)
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	if err := os.WriteFile(
		fmt.Sprintf("%s/certs/%s", a.workdir, a.domains[0]),
		cmd,
		0600,
	); err != nil {
		log.Error().Err(err).Msg("")

	}

	// * write certificate chain (issuer + certificate)
	if err := os.WriteFile(
		fmt.Sprintf("%s/certs/cert.pem", a.workdir),
		certificates.Certificate,
		0600,
	); err != nil {
		log.Error().Err(err).Msg("write to fs")
		return err
	}
	// * write certificate private key
	if err := os.WriteFile(
		fmt.Sprintf("%s/certs/key.pem", a.workdir),
		certificates.PrivateKey,
		0600,
	); err != nil {
		log.Error().Err(err).Msg("write to fs")
		return err
	}
	log.Debug().Msgf("successfully written new certs to: %s/certs", a.workdir)

	return nil
}

// writeState saves the account state to the filesystem
func (a *Acme) writeState() error {

	// store state to filesystem
	state, err := json.Marshal(a.user)
	if err != nil {
		return err
	}

	if err := os.WriteFile(
		fmt.Sprintf("%s/state.json", a.workdir),
		state,
		0600,
	); err != nil {
		return err
	}

	log.Debug().Msgf("successfully written state to %s/state.json", a.workdir)
	return nil
}
