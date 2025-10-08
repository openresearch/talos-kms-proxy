package kms

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awskms "github.com/aws/aws-sdk-go/service/kms"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/siderolabs/kms-client/api/kms"
	oraws "github.openresearch.com/talos-kms-proxy/internal/aws"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

// Server implements gRPC API
type Server struct {
	kms.UnimplementedKMSServiceServer

	awscli       *oraws.AWS
	certsChannel chan map[string][]byte
	certs        map[string][]byte
	workdir      string
	endpoint     string
}

var (
	logger = log.With().Str("service", "kms").Logger().Output(zerolog.ConsoleWriter{Out: os.Stdout})
)

// NewServer initializes new server
func NewServer(endpoint, workdir string, certsChannel chan map[string][]byte) (*Server, error) {

	// load the AWS KMS keyID from env
	keyID := os.Getenv("AWS_KMS_KEY_ID")

	// create aws client session, credentials are taken from environment
	sess, err := session.NewSession(&aws.Config{Region: aws.String("eu-west-1")})
	if err != nil {
		return nil, err
	}

	// Create AWS KMS service client
	svc := awskms.New(sess)

	// create new OpenResearch KMS AWS helper
	awscli := oraws.NewAWS(svc)

	// and assign AWS KMS keyID to use and check for validity
	awscli.KeyID = keyID
	if err := awscli.CheckKeyExists(); err != nil {
		return nil, err
	}

	return &Server{
		awscli:       awscli,
		certsChannel: certsChannel,
		endpoint:     endpoint,
		workdir:      workdir,
	}, nil
}

// Serve implements the suture service
// It starts the grpc listener and then waits for events
// from the certsChannel to load new certificates
func (srv *Server) Serve(ctx context.Context) error {

	logger.Info().Msg("starting")

	// try to load existing certificates
	if err := srv.loadCerts(); err != nil {
		return fmt.Errorf("could not load existing certs: %w", err)
	}

	// we start the grpc service listener here
	// note that it will not start serving requests until the certificates
	// from the ACME service are created and sent via the certsChannel (see below)
	go srv.grpcListen(ctx)

	// reload certificates
	for {
		select {
		case <-ctx.Done():
			return nil
		case srv.certs = <-srv.certsChannel:
			logger.Info().Msgf("certificates rotated by acme client")
		}
	}
}

// loadCerts tries to load existing certs from filesystem
// if successfull it stores the certificates into Server.certs
func (srv *Server) loadCerts() error {

	certs := make(map[string][]byte)

	// try to read cert.pem
	crt, err := os.ReadFile(fmt.Sprintf("%s/certs/cert.pem", srv.workdir))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	certs["certificate"] = crt

	// try to read keu.pem
	pk, err := os.ReadFile(fmt.Sprintf("%s/certs/key.pem", srv.workdir))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	certs["privatekey"] = pk

	// assign existing certs to our server instance
	srv.certs = certs

	logger.Debug().Msgf("successfully loaded certs from %s/certs", srv.workdir)

	return nil
}

// getCerts parses the latest available certificates
// and returns them in *tls.Certificate format
func (srv *Server) getCerts(h *tls.ClientHelloInfo) (*tls.Certificate, error) {

	cert, err := tls.X509KeyPair(srv.certs["certificate"], srv.certs["privatekey"])
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

// grpcListen starts a goroutine that handles the GRPC calls for the KMS service
func (srv *Server) grpcListen(ctx context.Context) {

	// we load the server tls certificates here
	// this will call srv.getCerts on every new connection
	// and in turn enable "reloading" of certificates on the fly
	creds := credentials.NewTLS(&tls.Config{
		GetCertificate: srv.getCerts,
	})

	// create grpc server
	s := grpc.NewServer(grpc.Creds(creds))

	// register KMS Service servers
	kms.RegisterKMSServiceServer(s, srv)

	// register reflection
	reflection.Register(s)

	// start a tcp listener
	lis, err := net.Listen("tcp", srv.endpoint)
	if err != nil {
		logger.Fatal().Err(err).Msg("")
	}

	// start serving the KMS grpc service
	// and handle errors with an error group
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return s.Serve(lis)
	})
	eg.Go(func() error {
		<-ctx.Done()
		s.Stop()
		return nil
	})
	if err := eg.Wait(); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		logger.Fatal().Err(err).Msg("")
	}
}
