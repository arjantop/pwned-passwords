package grpctest

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
)

type Server struct {
	srv        *grpc.Server
	clientConn *grpc.ClientConn
}

func NewServer(init func(srv *grpc.Server)) *Server {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic("failed to listen")
	}

	srv := grpc.NewServer()

	init(srv)

	conn, err := grpc.Dial(listener.Addr().String(), grpc.WithInsecure())
	if err != nil {
		panic(fmt.Sprintf("failed to dial: %v", err))
	}

	go func() {
		if err := srv.Serve(listener); err != nil {
			panic(fmt.Sprintf("failed to serve on %s: %v", listener.Addr().String(), err))
		}
	}()

	return &Server{
		srv:        srv,
		clientConn: conn,
	}
}

func (s *Server) ClientConn() *grpc.ClientConn {
	return s.clientConn
}

func (s *Server) Close() {
	s.clientConn.Close()
	s.srv.Stop()
}
