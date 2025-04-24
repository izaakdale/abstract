package smserver

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/smithy-go/ptr"
	secrets "github.com/izaakdale/abstract/api/secrets/v1"
)

var _ secrets.SecretsServer = (*server)(nil)

type (
	Specification struct {
		ListenAddr string `envconfig:"LISTEN_ADDR"`
	}
	SecretsManagerAPI interface {
		GetSecretValue(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	}
	server struct {
		smcli SecretsManagerAPI
		secrets.UnimplementedSecretsServer
	}
)

func New(s SecretsManagerAPI) *server {
	return &server{
		smcli: s,
	}
}

func (s *server) FetchSecret(ctx context.Context, req *secrets.FetchSecretRequest) (*secrets.FetchSecretResponse, error) {
	out, err := s.smcli.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: ptr.String(req.Name),
	})
	if err != nil {
		return nil, err
	}
	return &secrets.FetchSecretResponse{
		Secret: *out.SecretString,
	}, nil
}
