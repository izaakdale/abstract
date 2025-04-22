gen:
	protoc api/*/*/*.proto --go_out=. --go-grpc_out=.

client:
	go run cmd/client/main.go

server:
	go run cmd/server/main.go