package main

import (
	"context"
	"log"
	"net"

	"github.com/go-redis/redis/v8"
	pubsub "github.com/izaakdale/abstract/api/pubsub/v1"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var _ pubsub.RemoteServer = (*server)(nil)

type (
	Specification struct {
		ListenAddr string `envconfig:"LISTEN_ADDR"`
		RedisAddr  string `envconfig:"REDIS_ADDR"`
	}
	RedisPubSubAPI interface {
		Subscribe(context.Context, ...string) *redis.PubSub
		Publish(context.Context, string, interface{}) *redis.IntCmd
	}
	server struct {
		rps RedisPubSubAPI
		pubsub.UnimplementedRemoteServer
	}
)

func (s *server) Subscribe(req *pubsub.SubscriptionRequest, stream pubsub.Remote_SubscribeServer) error {
	redisPS := s.rps.Subscribe(context.Background(), req.Topic)
	ch := redisPS.Channel()
	for msg := range ch {
		if err := stream.Send(&pubsub.Event{
			AckId: "redis pubsub does not support ack",
			Body:  []byte(msg.Payload),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) Ack(ctx context.Context, req *pubsub.AckRequest) (*pubsub.AckResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ack not implemented: redis pubsub does not support ack")
}

func (s *server) Publish(ctx context.Context, req *pubsub.PublishRequest) (*pubsub.PublishResponse, error) {
	err := s.rps.Publish(ctx, req.Topic, req.Body).Err()
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

	client := redis.NewClient(&redis.Options{
		Addr:     spec.RedisAddr, // Replace with your Redis server address
		Password: "",             // Set password if applicable
		DB:       0,              // Set the default Redis database
	})

	// Test the connection
	_, err = client.Ping(context.Background()).Result()
	if err != nil {
		panic(err)
	}

	// create a server
	s := &server{
		rps: client,
	}

	// register the server
	gsrv := grpc.NewServer()
	pubsub.RegisterRemoteServer(gsrv, s)

	reflection.Register(gsrv)
	// serve the server
	log.Printf("grpc serving at: %s", lis.Addr())
	gsrv.Serve(lis)
}
