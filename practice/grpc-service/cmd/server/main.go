package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	userv1 "practice/grpc-service/api/user/v1"
	"practice/grpc-service/internal/repository"
	"practice/grpc-service/internal/service"
	transport "practice/grpc-service/internal/transport/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:50051", "listen address")
	flag.Parse()
	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	server := grpc.NewServer()
	userv1.RegisterUserServiceServer(server, transport.New(service.New(repository.NewMemory())))
	reflection.Register(server)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)
	go func() {
		<-stop
		timer := time.AfterFunc(5*time.Second, server.Stop)
		defer timer.Stop()
		server.GracefulStop()
	}()
	log.Printf("gRPC listening on %s", *addr)
	if err := server.Serve(listener); err != nil {
		log.Fatal(err)
	}
}
