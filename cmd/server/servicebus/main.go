package main

import (
	"context"
	"errors"
	"log"
	"net"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	pubsub "github.com/izaakdale/abstract/api/pubsub/v1"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var _ pubsub.RemoteServer = (*server)(nil)

type (
	Specification struct {
		ListenAddr                 string `envconfig:"LISTEN_ADDR"`
		ServiceBusConnectionString string `envconfig:"SERVICE_BUS_CONNECTION_STRING"`
	}
	ServiceBusAPI interface {
		NewSender(string, *azservicebus.NewSenderOptions) (*azservicebus.Sender, error)
		NewReceiverForQueue(string, *azservicebus.ReceiverOptions) (*azservicebus.Receiver, error)
	}
	ServiceBusSendAPI interface {
		SendMessage(context.Context, *azservicebus.Message, *azservicebus.SendMessageOptions) error
	}
	ServiceBusReceiveAPI interface {
		ReceiveMessages(context.Context, int, *azservicebus.ReceiveMessagesOptions) ([]*azservicebus.ReceivedMessage, error)
		CompleteMessage(context.Context, *azservicebus.ReceivedMessage, *azservicebus.CompleteMessageOptions) error
		DeferMessage(context.Context, *azservicebus.ReceivedMessage, *azservicebus.DeferMessageOptions) error
		ReceiveDeferredMessages(context.Context, []int64, *azservicebus.ReceiveDeferredMessagesOptions) ([]*azservicebus.ReceivedMessage, error)
	}
	server struct {
		cli   ServiceBusAPI
		rcvrs map[string]ServiceBusReceiveAPI
		sndrs map[string]ServiceBusSendAPI
		pubsub.UnimplementedRemoteServer

		// deferredMsgs maps message IDs to sequence numbers
		// NOTE: This is stored in-mem for now, we should look into how to make this restart persistent.
		deferredMsgs map[string]int64
	}
)

func (s *server) Subscribe(req *pubsub.SubscriptionRequest, stream pubsub.Remote_SubscribeServer) error {
	receiver, err := s.GetReceiver(req.Topic)
	if err != nil {
		return err
	}
	ctx := stream.Context()
	for {
		messages, err := receiver.ReceiveMessages(ctx, 1, nil)
		if err != nil {
			return err
		}
		if len(messages) == 0 {
			continue
		}

		for _, msg := range messages {
			if err := stream.Send(&pubsub.Event{
				AckId: msg.MessageID,
				Body:  msg.Body,
			}); err != nil {
				return err
			}
			if err := receiver.DeferMessage(ctx, msg, nil); err != nil {
				return err
			}
			s.deferredMsgs[msg.MessageID] = *msg.SequenceNumber
		}
	}
}

func (s *server) Ack(ctx context.Context, req *pubsub.AckRequest) (*pubsub.AckResponse, error) {
	receiver, err := s.GetReceiver(req.Topic)
	if err != nil {
		return nil, err
	}
	deferredMsgSeq, ok := s.deferredMsgs[req.AckId]
	if !ok {
		return nil, errors.New("message not ackable")
	}
	msgs, err := receiver.ReceiveDeferredMessages(context.TODO(), []int64{deferredMsgSeq}, nil)
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 || msgs[0].MessageID != req.AckId || *msgs[0].SequenceNumber != deferredMsgSeq {
		return nil, errors.New("message not received")
	}
	if err := receiver.CompleteMessage(context.TODO(), msgs[0], nil); err != nil {
		return nil, err
	}
	return &pubsub.AckResponse{}, nil
}

func (s *server) Publish(ctx context.Context, req *pubsub.PublishRequest) (*pubsub.PublishResponse, error) {
	sender, err := s.GetSender(req.Topic)
	if err != nil {
		return nil, err
	}
	if err := sender.SendMessage(ctx, &azservicebus.Message{
		Body: req.Body,
	}, nil); err != nil {
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

	// Create a client
	client, err := azservicebus.NewClientFromConnectionString(spec.ServiceBusConnectionString, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close(context.TODO())

	sender, err := client.NewSender("test", nil)
	if err != nil {
		log.Fatalf("failed to create sender: %v", err)
	}
	defer sender.Close(context.TODO())

	receiver, err := client.NewReceiverForQueue("test", nil)
	if err != nil {
		log.Fatalf("failed to create receiver: %v", err)
	}
	defer receiver.Close(context.TODO())

	// create a server
	s := &server{
		cli:          client,
		rcvrs:        make(map[string]ServiceBusReceiveAPI),
		sndrs:        make(map[string]ServiceBusSendAPI),
		deferredMsgs: make(map[string]int64),
	}

	// register the server
	gsrv := grpc.NewServer()
	pubsub.RegisterRemoteServer(gsrv, s)

	reflection.Register(gsrv)
	// serve the server
	log.Printf("grpc serving at: %s", lis.Addr())
	gsrv.Serve(lis)
}

func (s *server) GetReceiver(topiqueue string) (ServiceBusReceiveAPI, error) {
	if receiver, ok := s.rcvrs[topiqueue]; ok {
		return receiver, nil
	}
	receiver, err := s.cli.NewReceiverForQueue(topiqueue, nil)
	if err != nil {
		return nil, err
	}
	s.rcvrs[topiqueue] = receiver
	return receiver, nil
}

func (s *server) GetSender(topiqueue string) (ServiceBusSendAPI, error) {
	if sender, ok := s.sndrs[topiqueue]; ok {
		return sender, nil
	}
	sender, err := s.cli.NewSender(topiqueue, nil)
	if err != nil {
		return nil, err
	}
	s.sndrs[topiqueue] = sender
	return sender, nil
}
