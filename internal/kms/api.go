package kms

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/siderolabs/kms-client/api/kms"
)

// Seal encrypts the incoming data
func (srv *Server) Seal(ctx context.Context, req *kms.Request) (*kms.Response, error) {

	log.Info().Msgf("Sealing fde key for node %s", req.NodeUuid)

	encdata, err := srv.awscli.EncryptData(string(req.Data), ctx)
	if err != nil {
		return nil, err
	}

	return &kms.Response{
		Data: encdata.CiphertextBlob,
	}, nil
}

// Unseal decrypts the incoming data
func (srv *Server) Unseal(ctx context.Context, req *kms.Request) (*kms.Response, error) {

	log.Info().Msgf("Unsealing fde key for node %s", req.NodeUuid)

	data, err := srv.awscli.DecryptData(string(req.Data), ctx)
	if err != nil {
		return nil, err
	}

	return &kms.Response{
		Data: data.Plaintext,
	}, nil
}
