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
	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/utils"
	pb "github.com/openshift/dpu-operator/dpu-api/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NetworkFunctionServiceServer struct {
	pb.UnimplementedNetworkFunctionServiceServer
	p4rtbin string
}

func NewNetworkFunctionService(p4rtbin string) *NetworkFunctionServiceServer {
	return &NetworkFunctionServiceServer{
		p4rtbin: p4rtbin,
	}
}

func (s *NetworkFunctionServiceServer) CreateNetworkFunction(ctx context.Context, in *pb.NFRequest) (*pb.Empty, error) {

	vfMacList, err := utils.GetVfMacList()

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unable to reach the IMC %v", err)
	}

	// Remove point-to-point between host VFs from the FXP
	p4rtclient.DeletePointToPointVFRules(s.p4rtbin, vfMacList)

	// Generate the P4 rules and program the FXP with NF comms
	p4rtclient.CreateNetworkFunctionRules(s.p4rtbin, vfMacList, in.Input, in.Output)

	return &pb.Empty{}, nil
}

func (s *NetworkFunctionServiceServer) DeleteNetworkFunction(ctx context.Context, in *pb.NFRequest) (*pb.Empty, error) {

	vfMacList, err := utils.GetVfMacList()

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unable to reach the IMC %v", err)
	}

	// Remove the NF comms from the FXP
	p4rtclient.DeleteNetworkFunctionRules(s.p4rtbin, vfMacList, in.Input, in.Output)

	// Generate the P4 rules and program the FXP with point-to-point rules between host VFs
	p4rtclient.CreatePointToPointVFRules(s.p4rtbin, vfMacList)

	return &pb.Empty{}, nil
}
