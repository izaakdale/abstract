package sqsserver

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/ptr"
	pubsub "github.com/izaakdale/abstract/api/pubsub/v1"
)

var _ pubsub.PubSubServer = (*serverSQS)(nil)

type (
	Specification struct {
		ListenAddr string `envconfig:"LISTEN_ADDR"`
	}
	SQSAPI interface {
		ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
		DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
		SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	}
	serverSQS struct {
		sqscli SQSAPI
		pubsub.UnimplementedPubSubServer
	}
)

func New(s SQSAPI) *serverSQS {
	return &serverSQS{
		sqscli: s,
	}
}

func (s *serverSQS) Subscribe(req *pubsub.SubscriptionRequest, stream pubsub.PubSub_SubscribeServer) error {
	ctx := stream.Context()
	for {
		msgout, err := s.sqscli.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
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

func (s *serverSQS) Ack(ctx context.Context, req *pubsub.AckRequest) (*pubsub.AckResponse, error) {
	if _, err := s.sqscli.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      ptr.String(req.Topic),
		ReceiptHandle: ptr.String(req.AckId),
	}); err != nil {
		return nil, err
	}
	return &pubsub.AckResponse{}, nil
}

func (s *serverSQS) Publish(ctx context.Context, req *pubsub.PublishRequest) (*pubsub.PublishResponse, error) {
	_, err := s.sqscli.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    ptr.String(req.Topic),
		MessageBody: ptr.String(string(req.Body)),
	})
	if err != nil {
		return nil, err
	}
	return &pubsub.PublishResponse{}, nil
}
