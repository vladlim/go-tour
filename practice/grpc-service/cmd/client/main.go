package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	userv1 "practice/grpc-service/api/user/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:50051", "server address")
	name := flag.String("name", "Alice", "user name")
	flag.Parse()
	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := userv1.NewUserServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	created, err := client.CreateUser(ctx, &userv1.CreateUserRequest{Name: *name})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created: id=%s name=%s\n", created.GetId(), created.GetName())
	found, err := client.GetUser(ctx, &userv1.GetUserRequest{Id: created.GetId()})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found:   id=%s name=%s\n", found.GetId(), found.GetName())
}
