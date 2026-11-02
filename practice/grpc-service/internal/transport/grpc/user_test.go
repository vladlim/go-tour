package grpc_test

import (
	"context"
	"net"
	"testing"
	"time"

	userv1 "practice/grpc-service/api/user/v1"
	"practice/grpc-service/internal/repository"
	"practice/grpc-service/internal/service"
	transport "practice/grpc-service/internal/transport/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// Exercise all three layers over a real gRPC connection without opening a port.
func TestUserService(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	userv1.RegisterUserServiceServer(server, transport.New(service.New(repository.NewMemory())))
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	t.Cleanup(func() { _ = listener.Close() })
	conn, err := grpc.NewClient("passthrough:///test",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	client := userv1.NewUserServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	created, err := client.CreateUser(ctx, &userv1.CreateUserRequest{Name: "  Vlad  "})
	if err != nil {
		t.Fatal(err)
	}
	if created.GetId() == "" || created.GetName() != "Vlad" {
		t.Fatalf("unexpected user: %v", created)
	}
	found, err := client.GetUser(ctx, &userv1.GetUserRequest{Id: created.GetId()})
	if err != nil {
		t.Fatal(err)
	}
	if found.GetId() != created.GetId() || found.GetName() != created.GetName() {
		t.Fatalf("user changed: %v", found)
	}
	_, err = client.CreateUser(ctx, &userv1.CreateUserRequest{Name: "  "})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty name: %v", err)
	}
	_, err = client.GetUser(ctx, &userv1.GetUserRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty ID: %v", err)
	}
	_, err = client.GetUser(ctx, &userv1.GetUserRequest{Id: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("missing user: %v", err)
	}
}
