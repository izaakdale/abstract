package main

import (
	"context"
	"log"

	pubsub "github.com/izaakdale/abstract/api/pubsub/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:7777", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pubsub.NewPubSubClient(conn)

	// req := &pubsub.SubscriptionRequest{
	// 	Topic: "abstract-queue",
	// }

	// stream, err := client.Subscribe(context.Background(), req)
	// if err != nil {
	// 	log.Fatalf("failed to subscribe: %v", err)
	// }

	// go func() {
	client.Publish(context.Background(), &pubsub.PublishRequest{
		Topic: "abstract-queue",
		Body:  []byte("hello world"),
	})
	// }()

	// for {
	// 	event, err := stream.Recv()
	// 	if err != nil {
	// 		log.Fatalf("failed to receive event: %v", err)
	// 	}
	// 	log.Printf("received event: %v", string(event.Body))

	// 	if _, err := client.Ack(context.Background(), &pubsub.AckRequest{
	// 		Topic:  req.Topic,
	// 		AckId:  event.AckId,
	// 		Result: pubsub.AckResult_SUCCESS,
	// 	}); err != nil {
	// 		log.Fatalf("failed to ack event: %v", err)
	// 	}
	// }
}
