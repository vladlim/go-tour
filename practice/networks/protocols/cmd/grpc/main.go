package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Production gRPC normally uses generated protobuf messages. JSON keeps this
// transport demo compact and removes the need to install protoc.
type jsonCodec struct{}

func (jsonCodec) Name() string                    { return "json" }
func (jsonCodec) Marshal(v any) ([]byte, error)   { return json.Marshal(v) }
func (jsonCodec) Unmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }

type HelloRequest struct {
	Name string `json:"name"`
}
type HelloReply struct {
	Message string `json:"message"`
}
type greeter interface {
	SayHello(context.Context, *HelloRequest) (*HelloReply, error)
	Countdown(*HelloRequest, grpc.ServerStream) error
}
type service struct{}

func (service) SayHello(ctx context.Context, in *HelloRequest) (*HelloReply, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	log.Printf("SERVER unary request=%+v metadata[x-request-id]=%v deadline-set=%v", in, md.Get("x-request-id"), hasDeadline(ctx))
	if in.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	return &HelloReply{Message: "hello, " + in.Name}, nil
}
func (service) Countdown(in *HelloRequest, stream grpc.ServerStream) error {
	log.Printf("SERVER starts HTTP/2 server-stream for %q", in.Name)
	for i := 3; i > 0; i-- {
		if err := stream.SendMsg(&HelloReply{Message: fmt.Sprintf("%s: %d", in.Name, i)}); err != nil {
			return err
		}
	}
	return nil
}
func hasDeadline(ctx context.Context) bool { _, ok := ctx.Deadline(); return ok }

func unaryHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(HelloRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	call := func(ctx context.Context, req any) (any, error) {
		return srv.(greeter).SayHello(ctx, req.(*HelloRequest))
	}
	if interceptor == nil {
		return call(ctx, in)
	}
	return interceptor(ctx, in, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/demo.Greeter/SayHello"}, call)
}
func streamHandler(srv any, stream grpc.ServerStream) error {
	in := new(HelloRequest)
	if err := stream.RecvMsg(in); err != nil {
		return err
	}
	return srv.(greeter).Countdown(in, stream)
}

var description = grpc.ServiceDesc{ServiceName: "demo.Greeter", HandlerType: (*greeter)(nil), Methods: []grpc.MethodDesc{{MethodName: "SayHello", Handler: unaryHandler}}, Streams: []grpc.StreamDesc{{StreamName: "Countdown", Handler: streamHandler, ServerStreams: true}}}

func main() {
	log.SetFlags(log.Lmicroseconds)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	server := grpc.NewServer(grpc.ForceServerCodec(jsonCodec{}))
	server.RegisterService(&description, service{})
	go func() {
		log.Printf("SERVER gRPC (HTTP/2) listen %s", lis.Addr())
		if err := server.Serve(lis); err != nil {
			log.Printf("serve: %v", err)
		}
	}()
	defer server.GracefulStop()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.ForceCodec(jsonCodec{})), grpc.WithBlock())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	log.Printf("CLIENT HTTP/2 connection ready")
	callCtx, callCancel := context.WithTimeout(metadata.AppendToOutgoingContext(ctx, "x-request-id", "req-42"), time.Second)
	defer callCancel()
	out := new(HelloReply)
	log.Printf("CLIENT unary /demo.Greeter/SayHello")
	err = conn.Invoke(callCtx, "/demo.Greeter/SayHello", &HelloRequest{Name: "Go"}, out)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("CLIENT unary response=%q", out.Message)
	desc := &grpc.StreamDesc{ServerStreams: true}
	stream, err := conn.NewStream(callCtx, desc, "/demo.Greeter/Countdown")
	if err != nil {
		log.Fatal(err)
	}
	if err = stream.SendMsg(&HelloRequest{Name: "stream"}); err != nil {
		log.Fatal(err)
	}
	if err = stream.CloseSend(); err != nil {
		log.Fatal(err)
	}
	for {
		msg := new(HelloReply)
		err = stream.RecvMsg(msg)
		if err == io.EOF {
			log.Printf("CLIENT stream EOF (normal completion)")
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("CLIENT stream message=%q", msg.Message)
	}
	err = conn.Invoke(callCtx, "/demo.Greeter/SayHello", &HelloRequest{}, new(HelloReply))
	st, _ := status.FromError(err)
	log.Printf("CLIENT structured error code=%s message=%q", st.Code(), st.Message())
}
