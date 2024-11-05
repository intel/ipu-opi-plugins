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

package ipuplugin

import (
	"context"

	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/p4rtclient"
	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/types"
	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/utils"
	pb "github.com/openshift/dpu-operator/dpu-api/gen"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NetworkFunctionServiceServer struct {
	pb.UnimplementedNetworkFunctionServiceServer
	Ports      map[string]*types.BridgePortInfo
	bridgeCtlr types.BridgeController
	p4RtClient types.P4RTClient
	p4rtbin    string
}

func NewNetworkFunctionService(ports map[string]*types.BridgePortInfo, brCtlr types.BridgeController, p4Client types.P4RTClient, p4rtbin string) *NetworkFunctionServiceServer {
	return &NetworkFunctionServiceServer{
		Ports:      ports,
		bridgeCtlr: brCtlr,
		p4RtClient: p4Client,
		p4rtbin:    p4rtbin,
	}
}

func (s *NetworkFunctionServiceServer) CreateNetworkFunction(ctx context.Context, in *pb.NFRequest) (*pb.Empty, error) {
	vfMacList, err := utils.GetVfMacList()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unable to reach the IMC %v", err)
	}

	if len(vfMacList) == 0 {
		return nil, status.Error(codes.Internal, "No NFs initialized on the host")
	}

	ingressIndex, err := FindInterfaceIdForGivenMac(in.Input)
	if err != nil {
		return nil, err
	}
	egressIndex, err := FindInterfaceIdForGivenMac(in.Output)
	if err != nil {
		return nil, err
	}
	if err := s.bridgeCtlr.AddPort(AccIntfNames[ingressIndex]); err != nil {
		log.Errorf("failed to add port to bridge: %v, for interface->%v", err, AccIntfNames[ingressIndex])
		//return nil, fmt.Errorf("failed to add port to bridge: %v, for interface->%v", err, AccIntfNames[ingressIndex])
	}
	if err := s.bridgeCtlr.AddPort(AccIntfNames[egressIndex]); err != nil {
		log.Errorf("failed to add port to bridge: %v, for interface->%v", err, AccIntfNames[egressIndex])
		//return nil, fmt.Errorf("failed to add port to bridge: %v, for interface->%v", err, AccIntfNames[egressIndex])
	}
	if err := s.bridgeCtlr.AddPort(AccIntfNames[NF_IN_PR_INTF_INDEX]); err != nil {
		log.Errorf("failed to add port to bridge: %v, for interface->%v", err, AccIntfNames[NF_IN_PR_INTF_INDEX])
		//return nil, fmt.Errorf("failed to add port to bridge: %v, for interface->%v", err, AccIntfNames[NF_IN_PR_INTF_INDEX])
	}
	if err := s.bridgeCtlr.AddPort(AccIntfNames[NF_OUT_PR_INTF_INDEX]); err != nil {
		log.Errorf("failed to add port to bridge: %v, for interface->%v", err, AccIntfNames[NF_OUT_PR_INTF_INDEX])
		//return nil, fmt.Errorf("failed to add port to bridge: %v, for interface->%v", err, AccIntfNames[NF_OUT_PR_INTF_INDEX])
	}
	log.Infof("added interfaces:in->%s, out->%s, inPR->%s, outPR->%s",
		AccIntfNames[ingressIndex], AccIntfNames[egressIndex], AccIntfNames[NF_IN_PR_INTF_INDEX], AccIntfNames[NF_OUT_PR_INTF_INDEX])
	// Remove point-to-point between host VFs from the FXP
	//p4rtclient.DeletePointToPointVFRules(s.p4rtbin, vfMacList)
	/*Note: Currently this API does not have host-VF info, since there is no reference to what was passed by DPU in CreateBridgePort.
	As a work-around, we take full vfMacList, and write P4 rules, to connect all host VFs to NF.	*/
	// Generate the P4 rules and program the FXP with NF comms
	log.Infof("AddNFP4Rules, path->%s, 1-%v, 2-%v, 3-%v, 4-%v, 5-%v",
		s.p4rtbin, vfMacList, in.Input, in.Output, AccApfMacList[NF_IN_PR_INTF_INDEX], AccApfMacList[NF_OUT_PR_INTF_INDEX])
	p4rtclient.AddNFP4Rules(s.p4rtbin, vfMacList, in.Input, in.Output, AccApfMacList[NF_IN_PR_INTF_INDEX], AccApfMacList[NF_OUT_PR_INTF_INDEX])

	return &pb.Empty{}, nil
}

func (s *NetworkFunctionServiceServer) DeleteNetworkFunction(ctx context.Context, in *pb.NFRequest) (*pb.Empty, error) {

	vfMacList, err := utils.GetVfMacList()

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unable to reach the IMC %v", err)
	}

	if len(vfMacList) == 0 {
		return nil, status.Error(codes.Internal, "No NFs initialized on the host")
	}

	ingressIndex, err := FindInterfaceIdForGivenMac(in.Input)
	if err != nil {
		return nil, err
	}
	egressIndex, err := FindInterfaceIdForGivenMac(in.Output)
	if err != nil {
		return nil, err
	}
	if err := s.bridgeCtlr.DeletePort(AccIntfNames[ingressIndex]); err != nil {
		log.Errorf("failed to delete port to bridge: %v, for interface->%v", err, AccIntfNames[ingressIndex])
		//return nil, fmt.Errorf("failed to add port to bridge: %v, for interface->%v", err, AccIntfNames[ingressIndex])
	}
	if err := s.bridgeCtlr.DeletePort(AccIntfNames[egressIndex]); err != nil {
		log.Errorf("failed to delete port to bridge: %v, for interface->%v", err, AccIntfNames[egressIndex])
		//return nil, fmt.Errorf("failed to add port to bridge: %v, for interface->%v", err, AccIntfNames[egressIndex])
	}
	if err := s.bridgeCtlr.DeletePort(AccIntfNames[NF_IN_PR_INTF_INDEX]); err != nil {
		log.Errorf("failed to delete port to bridge: %v, for interface->%v", err, AccIntfNames[NF_IN_PR_INTF_INDEX])
		//return nil, fmt.Errorf("failed to add port to bridge: %v, for interface->%v", err, AccIntfNames[NF_IN_PR_INTF_INDEX])
	}
	if err := s.bridgeCtlr.DeletePort(AccIntfNames[NF_OUT_PR_INTF_INDEX]); err != nil {
		log.Errorf("failed to delete port to bridge: %v, for interface->%v", err, AccIntfNames[NF_OUT_PR_INTF_INDEX])
		//return nil, fmt.Errorf("failed to add port to bridge: %v, for interface->%v", err, AccIntfNames[NF_OUT_PR_INTF_INDEX])
	}
	log.Infof("deleted interfaces:in->%s, out->%s, inPR->%s, outPR->%s",
		AccIntfNames[ingressIndex], AccIntfNames[egressIndex], AccIntfNames[NF_IN_PR_INTF_INDEX], AccIntfNames[NF_OUT_PR_INTF_INDEX])

	log.Infof("DeleteNFP4Rules, path->%s, 1-%v, 2-%v, 3-%v, 4-%v, 5-%v",
		s.p4rtbin, vfMacList, in.Input, in.Output, AccApfMacList[NF_IN_PR_INTF_INDEX], AccApfMacList[NF_OUT_PR_INTF_INDEX])
	p4rtclient.DeleteNFP4Rules(s.p4rtbin, vfMacList, in.Input, in.Output, AccApfMacList[NF_IN_PR_INTF_INDEX], AccApfMacList[NF_OUT_PR_INTF_INDEX])
	// Generate the P4 rules and program the FXP with point-to-point rules between host VFs
	//p4rtclient.CreatePointToPointVFRules(s.p4rtbin, vfMacList)

	return &pb.Empty{}, nil
}
