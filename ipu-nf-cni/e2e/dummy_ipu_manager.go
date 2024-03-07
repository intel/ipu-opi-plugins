// Copyright (c) 2024 Intel Corporation.  All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License")
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"

	pb "main/build"
)

var (
	portFlag = flag.Int("port", 50551, "The server port")
)

type server struct {
	pb.UnimplementedIpuManagerServiceServer
}

func (s *server) CreateIpuManagerRequest(ctx context.Context, in *pb.IpuManagerRequest) (*pb.IpuManagerResponse, error) {

	request, _ := protojson.Marshal(in)
	fmt.Println(string(request))

	message := "Received!"
	response := &pb.IpuManagerResponse{
		Message: &message,
	}
	return response, nil
}

func main() {
	flag.Parse()
	port := *portFlag
	// port can be provided as positional parameter to support backward compatibility
	// the positional parameter has higher significance than -port flag
	if flag.NArg() == 1 {
		value, err := strconv.Atoi(flag.Arg(0))
		if err != nil {
			log.Fatalf("Failed to parse \"port\": %v", err)
		}
		port = value
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	} else {
		fmt.Printf("listening on port %d\n", port)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterIpuManagerServiceServer(grpcServer, &server{})
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
