package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
)

type Config struct {
	NodeID               string
	WorkloadAPISocketDir string
	CSISocketPath        string
}

func Run(config Config) error {
	if config.NodeID == "" {
		return errors.New("node ID is required")
	}
	if config.WorkloadAPISocketDir == "" {
		return errors.New("workload API socket directory is required")
	}
	if config.CSISocketPath == "" {
		return errors.New("CSI socket path is required")
	}

	if err := os.Remove(config.CSISocketPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Unable to remove CSI socket: %v", err)
	}

	listener, err := net.Listen("unix", config.CSISocketPath)
	if err != nil {
		return fmt.Errorf("unable to create CSI socket listener: %w", err)
	}

	driver := &Driver{
		NodeID:               config.NodeID,
		WorkloadAPISocketDir: config.WorkloadAPISocketDir,
	}

	server := grpc.NewServer(
		grpc.UnaryInterceptor(unaryRPCLogger),
		grpc.StreamInterceptor(streamRPCLogger),
	)
	csi.RegisterIdentityServer(server, driver)
	csi.RegisterNodeServer(server, driver)

	log.Println("Listening...")
	return server.Serve(listener)
}

func unaryRPCLogger(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		log.Printf("[%s] error: %s", info.FullMethod, err)
	}
	return resp, err
}

func streamRPCLogger(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := handler(srv, ss)
	if err != nil {
		log.Printf("[%s] error: %s", info.FullMethod, err)
	}
	return err
}
