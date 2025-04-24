package app

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	pubsub "github.com/izaakdale/abstract/api/pubsub/v1"
	secrets "github.com/izaakdale/abstract/api/secrets/v1"
	rmqserver "github.com/izaakdale/abstract/internal/server/pubsub/rabbitmq"
	sbserver "github.com/izaakdale/abstract/internal/server/pubsub/servicebus"
	sqsserver "github.com/izaakdale/abstract/internal/server/pubsub/sqs"
	serversm "github.com/izaakdale/abstract/internal/server/secrets/secretsmanager"
	vserver "github.com/izaakdale/abstract/internal/server/secrets/vault"
	"github.com/kelseyhightower/envconfig"
	amqp "github.com/streadway/amqp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var _ pubsub.PubSubServer = (*server)(nil)
var _ secrets.SecretsServer = (*server)(nil)

type (
	Specification struct {
		ListenAddr                 string `envconfig:"LISTEN_ADDR"`
		ServiceBusConnectionString string `envconfig:"SERVICE_BUS_CONNECTION_STRING"`
		RabbitMQConnectionString   string `envconfig:"RABBIT_MQ_CONNECTION_STRING"`
	}
	server struct {
		pubsubs map[pubsub.Protocol]pubsub.PubSubServer
		secrets map[secrets.Protocol]secrets.SecretsServer

		pubsub.UnimplementedPubSubServer
		secrets.UnimplementedSecretsServer
	}
)

func (s *server) Subscribe(req *pubsub.SubscriptionRequest, stream pubsub.PubSub_SubscribeServer) error {
	switch req.Protocol {
	case pubsub.Protocol_SQS:
		underlying, ok := s.pubsubs[pubsub.Protocol_SQS]
		if !ok {
			return status.Error(codes.InvalidArgument, "protocol not initialised")
		}
		return underlying.Subscribe(req, stream)
	case pubsub.Protocol_SERVICEBUS:
		underlying, ok := s.pubsubs[pubsub.Protocol_SERVICEBUS]
		if !ok {
			return status.Error(codes.InvalidArgument, "protocol not initialised")
		}
		return underlying.Subscribe(req, stream)
	case pubsub.Protocol_RABBITMQ:
		underlying, ok := s.pubsubs[pubsub.Protocol_RABBITMQ]
		if !ok {
			return status.Error(codes.InvalidArgument, "protocol not initialised")
		}
		return underlying.Subscribe(req, stream)
	default:
		return status.Error(codes.Unimplemented, "protocol not implemented")
	}
}

func (s *server) Ack(ctx context.Context, req *pubsub.AckRequest) (*pubsub.AckResponse, error) {
	switch req.Protocol {
	case pubsub.Protocol_SQS:
		underlying, ok := s.pubsubs[pubsub.Protocol_SQS]
		if !ok {
			return &pubsub.AckResponse{}, status.Error(codes.InvalidArgument, "protocol not initialised")
		}
		return underlying.Ack(ctx, req)
	case pubsub.Protocol_SERVICEBUS:
		underlying, ok := s.pubsubs[pubsub.Protocol_SERVICEBUS]
		if !ok {
			return &pubsub.AckResponse{}, status.Error(codes.InvalidArgument, "protocol not initialised")
		}
		return underlying.Ack(ctx, req)
	case pubsub.Protocol_RABBITMQ:
		underlying, ok := s.pubsubs[pubsub.Protocol_RABBITMQ]
		if !ok {
			return &pubsub.AckResponse{}, status.Error(codes.InvalidArgument, "protocol not initialised")
		}
		return underlying.Ack(ctx, req)
	default:
		return &pubsub.AckResponse{}, status.Error(codes.InvalidArgument, "protocol not implemented")
	}
}

func (s *server) Publish(ctx context.Context, req *pubsub.PublishRequest) (*pubsub.PublishResponse, error) {
	switch req.Protocol {
	case pubsub.Protocol_SQS:
		underlying, ok := s.pubsubs[pubsub.Protocol_SQS]
		if !ok {
			return &pubsub.PublishResponse{}, status.Error(codes.InvalidArgument, "protocol not initialised")
		}
		return underlying.Publish(ctx, req)
	case pubsub.Protocol_SERVICEBUS:
		underlying, ok := s.pubsubs[pubsub.Protocol_SERVICEBUS]
		if !ok {
			return &pubsub.PublishResponse{}, status.Error(codes.InvalidArgument, "protocol not initialised")
		}
		return underlying.Publish(ctx, req)
	case pubsub.Protocol_RABBITMQ:
		underlying, ok := s.pubsubs[pubsub.Protocol_RABBITMQ]
		if !ok {
			return &pubsub.PublishResponse{}, status.Error(codes.InvalidArgument, "protocol not initialised")
		}
		return underlying.Publish(ctx, req)
	default:
		return &pubsub.PublishResponse{}, status.Error(codes.Unimplemented, "protocol not implemented")
	}
}

func (s *server) FetchSecret(ctx context.Context, req *secrets.FetchSecretRequest) (*secrets.FetchSecretResponse, error) {
	switch req.Protocol {
	case secrets.Protocol_SECRETS_MANAGER:
		underlying, ok := s.secrets[secrets.Protocol_SECRETS_MANAGER]
		if !ok {
			return &secrets.FetchSecretResponse{}, status.Error(codes.InvalidArgument, "protocol not initialised")
		}
		return underlying.FetchSecret(ctx, req)
	case secrets.Protocol_VAULT:
		underlying, ok := s.secrets[secrets.Protocol_VAULT]
		if !ok {
			return &secrets.FetchSecretResponse{}, status.Error(codes.InvalidArgument, "protocol not initialised")
		}
		return underlying.FetchSecret(ctx, req)
	default:
		return &secrets.FetchSecretResponse{}, status.Error(codes.Unimplemented, "protocol not implemented")
	}
}

func Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var spec Specification
	if err := envconfig.Process("", &spec); err != nil {
		log.Fatalf("Failed to process env: %v", err)
	}

	s := &server{
		pubsubs: make(map[pubsub.Protocol]pubsub.PubSubServer),
		secrets: make(map[secrets.Protocol]secrets.SecretsServer),
	}

	// AWS
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	sqscli := sqs.NewFromConfig(cfg)
	smcli := secretsmanager.NewFromConfig(cfg)

	s.pubsubs[pubsub.Protocol_SQS] = sqsserver.New(sqscli)
	s.secrets[secrets.Protocol_SECRETS_MANAGER] = serversm.New(smcli)

	// Azure
	client, err := azservicebus.NewClientFromConnectionString(spec.ServiceBusConnectionString, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close(context.TODO())

	s.pubsubs[pubsub.Protocol_SERVICEBUS] = sbserver.New(client)

	// RabbitMQ
	conn, err := amqp.Dial(spec.RabbitMQConnectionString)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	s.pubsubs[pubsub.Protocol_RABBITMQ] = rmqserver.New(ch)

	// Vault
	s.secrets[secrets.Protocol_VAULT] = vserver.New(nil)

	// register the server
	gsrv := grpc.NewServer()
	pubsub.RegisterPubSubServer(gsrv, s)
	secrets.RegisterSecretsServer(gsrv, s)

	lis, err := net.Listen("tcp", spec.ListenAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	reflection.Register(gsrv)
	// serve the server
	log.Printf("grpc serving at: %s", lis.Addr())
	gsrv.Serve(lis)

	return nil
}
