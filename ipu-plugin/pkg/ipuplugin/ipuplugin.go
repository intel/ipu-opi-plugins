// Copyright (c) 2023 Intel Corporation.  All Rights Reserved.
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

package ipuplugin

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/types"
	pb2 "github.com/openshift/dpu-operator/dpu-api/gen"
	pb "github.com/opiproject/opi-api/network/evpn-gw/v1alpha1/gen/go"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const ()

type server struct {
	//pb.UnimplementedBridgePortServiceServer
	servingHost     string
	servingPort     int
	netProtocol     string
	bridgeName      string
	uplinkInterface string
	grpcSrvr        *grpc.Server
	listener        net.Listener
	log             *log.Entry
	p4cpInstall     string
	Ports           map[string]*pb.BridgePort
	bridgeCtlr      types.BridgeController
	p4RtClient      types.P4RTClient
	mode            string
	daemonHostIp    string
	daemonIpuIp     string
	daemonPort      int
}

func NewIpuPlugin(port int, brCtlr types.BridgeController,
	p4Client types.P4RTClient, host, net, bridge, intf, p4cpInstall, mode, daemonHostIp, daemonIpuIp string, daemonPort int) types.Runnable {
	return &server{
		servingHost:     host,
		servingPort:     port,
		netProtocol:     net,
		bridgeName:      bridge,
		uplinkInterface: intf,
		grpcSrvr:        grpc.NewServer(),
		log:             log.WithField("pkg", "ipuplugin"),
		p4cpInstall:     p4cpInstall,
		Ports:           make(map[string]*pb.BridgePort),
		bridgeCtlr:      brCtlr,
		p4RtClient:      p4Client,
		mode:            mode,
		daemonHostIp:    daemonHostIp,
		daemonIpuIp:     daemonIpuIp,
		daemonPort:      daemonPort,
	}
}

func (s *server) Run() error {
	var err error
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	listen, err := net.Listen(s.netProtocol, fmt.Sprintf("%s:%d", s.servingHost, s.servingPort))
	if err != nil {
		log.Fatalf("failed to open the listener: %v", err)
		return fmt.Errorf("unable to run IPU manager")
	}

	if err := s.bridgeCtlr.EnsureBridgeExists(); err != nil {
		log.Fatalf("error while checking host bridge existance: %v", err)
		return fmt.Errorf("host bridge error")
	}

	pb.RegisterBridgePortServiceServer(s.grpcSrvr, s)
	pb2.RegisterLifeCycleServiceServer(s.grpcSrvr, NewLifeCycleService(s.daemonHostIp, s.daemonIpuIp, s.daemonPort, s.mode))

	s.log.WithField("addr", listen.Addr().String()).Info("IPU Manager server listening on at:")
	go func() {
		if err = s.grpcSrvr.Serve(listen); err != nil {
			log.Fatalf("IPU Manager failed to serve: %v", err)
			return
		}
	}()

	// Wait for SIGTERM signal
	<-signalChannel
	s.log.Infof("SIGINT received, exiting")
	s.Stop()
	return nil
}

func (s *server) Stop() {
	s.log.Info("Stopping IPU Manager")
	s.grpcSrvr.GracefulStop()
	if s.listener != nil {
		s.listener.Close()
	}
	s.log.Info("IPU Manager has stopped")
}
