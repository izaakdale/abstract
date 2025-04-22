package main

import (
	"context"
	"log"
	"net"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/ptr"
	pubsub "github.com/izaakdale/abstract/api/pubsub/v1"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var _ pubsub.RemoteServer = (*server)(nil)

type (
	Specification struct {
		ListenAddr string `envconfig:"LISTEN_ADDR"`
	}
	SQSAPI interface {
		ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
		DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
		SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	}
	server struct {
		sqscli SQSAPI
		pubsub.UnimplementedRemoteServer
	}
)

func (s *server) Subscribe(req *pubsub.SubscriptionRequest, stream pubsub.Remote_SubscribeServer) error {
	for {
		msgout, err := s.sqscli.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
			QueueUrl: ptr.String(req.Topic),
		})
		if err != nil {
			return err
		}
		if msgout != nil && len(msgout.Messages) > 0 {
			for _, msg := range msgout.Messages {
				if err := stream.Send(&pubsub.Event{
					AckId: *msg.ReceiptHandle,
					Body:  []byte(*msg.Body),
				}); err != nil {
					return err
				}
			}
		}
	}
}

func (s *server) Ack(ctx context.Context, req *pubsub.AckRequest) (*pubsub.AckResponse, error) {
	if _, err := s.sqscli.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      ptr.String(req.Topic),
		ReceiptHandle: ptr.String(req.AckId),
	}); err != nil {
		return nil, err
	}
	return &pubsub.AckResponse{}, nil
}

func (s *server) Publish(ctx context.Context, req *pubsub.PublishRequest) (*pubsub.PublishResponse, error) {
	_, err := s.sqscli.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    ptr.String(req.Topic),
		MessageBody: ptr.String(string(req.Body)),
	})
	if err != nil {
		return nil, err
	}
	return &pubsub.PublishResponse{}, nil
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
	sqscli := sqs.NewFromConfig(cfg)

	// create a server
	s := &server{
		sqscli: sqscli,
	}

	// register the server
	gsrv := grpc.NewServer()
	pubsub.RegisterRemoteServer(gsrv, s)

	reflection.Register(gsrv)
	// serve the server
	log.Printf("grpc serving at: %s", lis.Addr())
	gsrv.Serve(lis)
}
