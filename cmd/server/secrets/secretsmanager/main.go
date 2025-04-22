package main

import (
	"context"
	"log"
	"net"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/smithy-go/ptr"
	secrets "github.com/izaakdale/abstract/api/secrets/v1"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var _ secrets.RemoteServer = (*server)(nil)

type (
	Specification struct {
		ListenAddr string `envconfig:"LISTEN_ADDR"`
	}
	SecretsManagerAPI interface {
		GetSecretValue(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	}
	server struct {
		smcli SecretsManagerAPI
		secrets.UnimplementedRemoteServer
	}
)

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

func main() {
	var spec Specification
	if err := envconfig.Process("", &spec); err != nil {
		log.Fatalf("Failed to process env: %v", err)
	}

	lis, err := net.Listen("tcp", spec.ListenAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(err)
	}
	smcli := secretsmanager.NewFromConfig(cfg)

	// create a server
	s := &server{
		smcli: smcli,
	}

	// register the server
	gsrv := grpc.NewServer()
	secrets.RegisterRemoteServer(gsrv, s)

	reflection.Register(gsrv)
	// serve the server
	log.Printf("grpc serving at: %s", lis.Addr())
	gsrv.Serve(lis)
}
