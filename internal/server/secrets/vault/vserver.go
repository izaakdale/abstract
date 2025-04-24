package vserver

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/hashicorp/vault-client-go"
	secrets "github.com/izaakdale/abstract/api/secrets/v1"
)

var _ secrets.SecretsServer = (*server)(nil)

type (
	VaultAPI interface {
	}
	server struct {
		vaultcli VaultAPI
		secrets.UnimplementedSecretsServer
	}
)

func New(s VaultAPI) *server {
	return &server{
		vaultcli: s,
	}
}

func (s *server) FetchSecret(ctx context.Context, req *secrets.FetchSecretRequest) (*secrets.FetchSecretResponse, error) {
	client, err := vault.New(
		vault.WithAddress("http://127.0.0.1:8200"),
		vault.WithRequestTimeout(30*time.Second),
	)
	if err != nil {
		return nil, err
	}

	// v1l, err := client.Secrets.KvV1List(ctx, req.Name, vault.WithToken(os.Getenv("VAULT_TOKEN")))
	// if err != nil {
	// 	return nil, err
	// }
	// log.Printf("%+v\n", v1l)

	resp, err := client.Read(ctx, req.Name, vault.WithToken(os.Getenv("VAULT_TOKEN")))
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data["data"].(map[string]interface{})
	if !ok {
		return nil, errors.New("secret value was not a string")
	}
	return &secrets.FetchSecretResponse{
		Secret: data["value"].(string),
	}, nil

	// resp, err := client.Secrets.KvV2Read(ctx, req.Name, vault.WithToken(os.Getenv("VAULT_TOKEN")))
	// if err != nil {
	// 	return nil, err
	// }
	// secret, ok := resp.Data.Data["value"].(string)
	// if !ok {
	// 	return nil, errors.New("secret value was not a string")
	// }
	// return &secrets.FetchSecretResponse{
	// 	Secret: secret,
	// }, nil
}
