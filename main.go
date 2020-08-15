package main

import (
	"log"
	"net"
	pb "parsley-app/proto"
	"parsley-app/server"

	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", "0.0.0.0:8010")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	srv, err := server.Create()
	if err != nil {
		log.Fatalln(err)
	}
	pb.RegisterExecutionContainerServer(grpcServer, srv)
	log.Println("Serving at :8010")
	grpcServer.Serve(lis)
}
