package rmqserver

import (
	"context"
	"strconv"

	pubsub "github.com/izaakdale/abstract/api/pubsub/v1"
	"github.com/streadway/amqp"
)

var _ pubsub.PubSubServer = (*server)(nil)

type (
	Specification struct {
		ListenAddr string `envconfig:"LISTEN_ADDR"`
	}
	RabbitMQAPI interface {
		Consume(queueName string, consumerTag string, autoAck bool, exclusive bool, noLocal bool, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
		Publish(exchange, routingKey string, mandatory, immediate bool, msg amqp.Publishing) error
		Ack(tag uint64, multiple bool) error
		QueueDeclare(queueName string, durable, autoDelete, exclusive, noWait bool, arguments amqp.Table) (amqp.Queue, error)
		Recover(requeue bool) error
		Cancel(consumerTag string, noWait bool) error
	}
	server struct {
		rmq RabbitMQAPI
		pubsub.UnimplementedPubSubServer
	}
)

func New(rmq RabbitMQAPI) pubsub.PubSubServer {
	return &server{
		rmq: rmq,
	}
}

func (s *server) Subscribe(req *pubsub.SubscriptionRequest, stream pubsub.PubSub_SubscribeServer) error {
	_, err := s.rmq.QueueDeclare(
		req.Topic,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	msgs, err := s.rmq.Consume(
		req.Topic,
		req.ConsumerId,
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return err
	}

	for {
		select {
		case <-stream.Context().Done():
			if err := s.rmq.Cancel(req.ConsumerId, false); err != nil {
				return err
			}
			if err := s.rmq.Recover(true); err != nil {
				return err
			}
			return nil
		case msg := <-msgs:
			if err := stream.Send(&pubsub.Event{
				AckId: strconv.FormatUint(msg.DeliveryTag, 10),
				Body:  msg.Body,
			}); err != nil {
				return err
			}
		}
	}
}

func (s *server) Ack(ctx context.Context, req *pubsub.AckRequest) (*pubsub.AckResponse, error) {
	tag, err := strconv.ParseUint(req.AckId, 10, 64)
	if err != nil {
		return nil, err
	}
	if err := s.rmq.Ack(tag, false); err != nil {
		return nil, err
	}
	return &pubsub.AckResponse{}, nil
}

func (s *server) Publish(ctx context.Context, req *pubsub.PublishRequest) (*pubsub.PublishResponse, error) {
	_, err := s.rmq.QueueDeclare(
		req.Topic,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, err
	}
	err = s.rmq.Publish(
		"",        // exchange
		req.Topic, // routing key (queue name)
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			Body: req.Body,
		})
	if err != nil {
		return nil, err
	}
	return &pubsub.PublishResponse{}, nil
}
