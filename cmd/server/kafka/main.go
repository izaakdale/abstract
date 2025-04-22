package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	pubsub "github.com/izaakdale/abstract/api/pubsub/v1"
	"github.com/kelseyhightower/envconfig"
	"github.com/segmentio/kafka-go"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var _ pubsub.RemoteServer = (*server)(nil)

type (
	Specification struct {
		ListenAddr   string   `envconfig:"LISTEN_ADDR"`
		KafkaBrokers []string `envconfig:"KAFKA_BROKERS"`
	}
	server struct {
		kafkaBrokers []string
		readers      map[string]*kafka.Reader
		writers      map[string]*kafka.Writer
		pubsub.UnimplementedRemoteServer
	}
)

func (s *server) Subscribe(req *pubsub.SubscriptionRequest, stream pubsub.Remote_SubscribeServer) error {
	r, ok := s.readers[req.Topic]
	if !ok {
		r = kafka.NewReader(kafka.ReaderConfig{
			Brokers:     s.kafkaBrokers,
			Topic:       req.Topic,
			MaxWait:     10 * time.Second,
			StartOffset: kafka.LastOffset,
			MaxBytes:    10e2,
		})
		s.readers[req.Topic] = r
	}
	defer r.Close()

	for {
		m, err := r.ReadMessage(context.TODO())
		if err != nil {
			return err
		}
		if err := stream.Send(&pubsub.Event{
			AckId: fmt.Sprintf("%s-:-%d-:-%d", m.Topic, m.Partition, m.Offset),
			Body:  m.Value,
		}); err != nil {
			return err
		}
	}
}

func (s *server) Ack(ctx context.Context, req *pubsub.AckRequest) (*pubsub.AckResponse, error) {
	r, ok := s.readers[req.Topic]
	if !ok {
		r = kafka.NewReader(kafka.ReaderConfig{
			Brokers:     s.kafkaBrokers,
			Topic:       req.Topic,
			MaxWait:     10 * time.Second,
			StartOffset: kafka.LastOffset,
			MaxBytes:    10e2,
		})
		s.readers[req.Topic] = r
	}

	tpo := strings.Split(req.AckId, "-:-")
	if len(tpo) != 3 {
		return nil, status.Error(codes.InvalidArgument, "invalid ack id")
	}
	topic, partition, offset := tpo[0], tpo[1], tpo[2]
	partitionInt, err := strconv.Atoi(partition)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid partition")
	}
	offsetInt, err := strconv.Atoi(offset)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid offset")
	}
	return &pubsub.AckResponse{}, r.CommitMessages(context.Background(), kafka.Message{
		Topic:     topic,
		Partition: partitionInt,
		Offset:    int64(offsetInt),
	})
}

func (s *server) Publish(ctx context.Context, req *pubsub.PublishRequest) (*pubsub.PublishResponse, error) {
	w, ok := s.writers[req.Topic]
	if !ok {
		w = kafka.NewWriter(kafka.WriterConfig{
			Brokers: s.kafkaBrokers,
			Topic:   req.Topic,
		})
		s.writers[req.Topic] = w
	}
	return &pubsub.PublishResponse{}, w.WriteMessages(ctx, kafka.Message{
		Value: req.Body,
	})
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

	// create a server
	s := &server{
		kafkaBrokers: spec.KafkaBrokers,
	}

	// register the server
	gsrv := grpc.NewServer()
	pubsub.RegisterRemoteServer(gsrv, s)

	reflection.Register(gsrv)
	// serve the server
	log.Printf("grpc serving at: %s", lis.Addr())
	gsrv.Serve(lis)
}
