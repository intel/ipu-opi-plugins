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
	"context"
	"fmt"
	"strconv"

	pb "github.com/opiproject/opi-api/network/evpn-gw/v1alpha1/gen/go"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	outerVlanId = 0 // hardcoded s-tag
)

// CreateBridgePort executes the creation of the port
func (s *server) CreateBridgePort(_ context.Context, in *pb.CreateBridgePortRequest) (*pb.BridgePort, error) {
	s.log.WithField("CreateBridgePortRequest", in).Debug("CreateBridgePort")

	// The assumption here is that the second octet is the VSI number.
	// e.g.; a mac address of 00:08:00:00:03:14 the corresponding VSI is 08.
	// VSI = 0 should be invalid and this function will return 0 when there's an error converting
	// this octet to int value
	macAddrSize := len(in.BridgePort.Spec.MacAddress)
	if macAddrSize < 1 || macAddrSize > 6 {
		// We do not have a valid mac address
		return nil, fmt.Errorf("invalid mac address provided")
	}
	vfVsi := int(in.BridgePort.Spec.MacAddress[1])
	if in.BridgePort.Spec.LogicalBridges == nil || len(in.BridgePort.Spec.LogicalBridges) < 1 {
		return nil, fmt.Errorf("vlan id is not provided")
	}
	vlan := s.getFirstVlanID(in.BridgePort.Spec.LogicalBridges)

	if vlan < 2 || vlan > 4094 {
		s.log.WithField("vlan", vlan).Debug("invalid vlan")
		return nil, fmt.Errorf("invalid vlan %d, vlan must be within 2-4094 range", vlan)
	}

	if vfVsi < 1 {
		s.log.WithField("vfVsi", vfVsi).Debug("invalid VSI")
		return nil, fmt.Errorf("invalid VSI:%d in given mac address, the value in 2nd octed must be > 0", vfVsi)
	}

	if isBridgePortPresent(*s, in.BridgePort.Name) {
		return s.Ports[in.BridgePort.Name], nil
	}

	vlanIntfName, err := createAndSetUpInnerVlan(s.uplinkInterface, in.BridgePort.Spec.LogicalBridges)
	if err != nil {
		s.log.WithField("vlan", in.BridgePort.Name).Error("unable to create vlan: ")
		return nil, fmt.Errorf("unable to create bridge port: %v", err)
	}

	if err := s.bridgeCtlr.AddPort(vlanIntfName); err != nil {
		return nil, fmt.Errorf("failed to add port to bridge: %v", err)
	}
	// Add FXP rules
	s.p4RtClient.AddRules(in.BridgePort.Spec.MacAddress, vlan)

	resp := proto.Clone(in.BridgePort).(*pb.BridgePort)
	resp.Status = &pb.BridgePortStatus{OperStatus: pb.BPOperStatus_BP_OPER_STATUS_UP}
	s.Ports[in.BridgePort.Name] = resp
	return resp, nil
}

// isBridgePortPresent checks if the bridge port is present
func isBridgePortPresent(srv server, brPortName string) bool {
	_, ok := srv.Ports[brPortName]
	return ok
}

func createAndSetUpOuterVlan(uplinkInterface string) (string, error) {
	upLink, err := linkByNameFn(uplinkInterface)
	if err != nil {
		return "", fmt.Errorf("unable to find uplink interface: %s, because: %w", uplinkInterface, err)
	}

	// Assume that the uplink interface name is something like enp0s1f0d3
	// take only the last two characters from the name to avoid long names limit
	vlanIntfName := fmt.Sprintf("%v.%d", uplinkInterface[len(uplinkInterface)-2:], outerVlanId)

	if err := createOuterVlanInterface(upLink, vlanIntfName, outerVlanId); err != nil {
		return "", err
	}

	return vlanIntfName, nil
}

func createOuterVlanInterface(upLink netlink.Link, vlanIntfName string, vlanId int) error {

	link, err := linkByNameFn(vlanIntfName)
	if err == nil {
		// If the vlan interface already exist then do nothing and return
		if _, ok := link.(*netlink.Vlan); !ok {
			return fmt.Errorf("an interface %s is found but not a VLAN device", vlanIntfName)
		}
		log.Debugf("outer vlan interface %s already exist", vlanIntfName)
		return nil
	}

	vlan := netlink.Vlan{}
	vlan.ParentIndex = upLink.Attrs().Index
	vlan.Name = vlanIntfName
	vlan.VlanId = vlanId
	vlan.VlanProtocol = netlink.VLAN_PROTOCOL_8021AD

	log.Debugf("creating a new outer vlan interface %s", vlanIntfName)
	if err := linkAddFn(&vlan); err != nil {
		return fmt.Errorf("error creating outer vlan interface %s: %s", vlanIntfName, err)
	}
	log.Debugf("outer vlan interface %s is created", vlanIntfName)

	return nil
}

func createAndSetUpInnerVlan(uplinkInterface string, bridges []string) (string, error) {
	// Assume that the uplink interface name is something like enp0s1f0d3
	// take only the last two characters from the name to avoid long names limit
	outerVlanIntfName := fmt.Sprintf("%v.%d", uplinkInterface[len(uplinkInterface)-2:], outerVlanId)

	upLink, err := linkByNameFn(outerVlanIntfName)
	if err != nil {
		if err == err.(netlink.LinkNotFoundError) {

			// Outer vlan interface has not been created yet
			// We create it here
			_, err := createAndSetUpOuterVlan(uplinkInterface)
			if err != nil {
				return "", fmt.Errorf("unable to create outer vlan interface %s, because: %w", outerVlanIntfName, err)
			}

			upLink, err = linkByNameFn(outerVlanIntfName)

			if err != nil {
				return "", fmt.Errorf("unable to find uplink interface: %s, because: %w", uplinkInterface, err)
			}

		} else {
			return "", fmt.Errorf("unable to find uplink interface: %s, because: %w", uplinkInterface, err)
		}
	}

	innerVlanId, err := strconv.Atoi(bridges[0])
	if err != nil {
		return "", fmt.Errorf("unable to parse vlan ID: %s, because: %w", bridges[0], err)
	}

	// Assume that the uplink interface name is something like enp0s1f0d3
	// take only the last two characters from the name to avoid long names limit
	vlanIntfName := fmt.Sprintf("%v.%d.%d", uplinkInterface[len(uplinkInterface)-2:], outerVlanId, innerVlanId)

	if err := createInnerVlanInterface(upLink, vlanIntfName, outerVlanId); err != nil {
		return "", err
	}

	return vlanIntfName, nil
}

func createInnerVlanInterface(upLink netlink.Link, vlanIntfName string, vlanId int) error {

	link, err := linkByNameFn(vlanIntfName)
	if err == nil {
		// If the vlan interface already exist then do nothing and return
		if _, ok := link.(*netlink.Vlan); !ok {
			return fmt.Errorf("an interface %s is found but not a VLAN device", vlanIntfName)
		}
		log.Debugf("inner vlan interface %s already exist", vlanIntfName)
		return nil
	}

	vlan := netlink.Vlan{}
	vlan.ParentIndex = upLink.Attrs().Index
	vlan.Name = vlanIntfName
	vlan.VlanId = vlanId
	vlan.VlanProtocol = netlink.VLAN_PROTOCOL_8021Q

	log.Debugf("creating a new inner vlan interface %s", vlanIntfName)
	if err := linkAddFn(&vlan); err != nil {
		return fmt.Errorf("error creating vlan interface %s: %s", vlanIntfName, err)
	}
	log.Debugf("inner vlan interface %s is created", vlanIntfName)

	return nil
}

func removeVlanInterface(vlanIntfName string) error {
	// Look up interface if it exists
	// If the vlan interface does not exist then do nothing and return
	link, err := linkByNameFn(vlanIntfName)
	if err != nil {
		// Link for vlan interface is found, nothing to do
		log.WithField("vlan", vlanIntfName).Info("interface not found to remove, may have been removed already")
		return nil
	}

	if err := linkDelFn(link); err != nil {
		return fmt.Errorf("error removing vlan interface %s: %s", vlanIntfName, err.Error())
	}

	return nil
}

// DeleteBridgePort deletes a port
func (s *server) DeleteBridgePort(_ context.Context, in *pb.DeleteBridgePortRequest) (*emptypb.Empty, error) {
	s.log.WithField("DeleteBridgePortRequest", in).Info("DeleteBridgePort")

	var portInfo *pb.BridgePort
	portInfo, ok := s.Ports[in.Name]
	if !ok {
		s.log.WithField("interface name", in.Name).Info("port info is not found")
		// in such case avoid delete call loop from CNI Agent which otherwise will repeatedly call DeleteBridgePort as retry
		return &emptypb.Empty{}, nil
	}

	vlan := s.getFirstVlanID(portInfo.Spec.LogicalBridges)
	vlanIntfName := fmt.Sprintf("%v.%d.%d", s.uplinkInterface[len(s.uplinkInterface)-2:], outerVlanId, vlan)

	if err := s.bridgeCtlr.DeletePort(vlanIntfName); err != nil {
		log.Error("unable to remove port from bridge", err)
		return nil, fmt.Errorf("failed to delete port from bridge: %v", err)
	}

	if err := removeVlanInterface(vlanIntfName); err != nil {
		log.Error("unable to remove remove interface from host", err)
		return nil, fmt.Errorf("failed to remove interface from host: %v", err)
	}

	// Delete FXP rules
	s.p4RtClient.DeleteRules(portInfo.Spec.MacAddress, vlan)

	delete(s.Ports, in.Name)
	return &emptypb.Empty{}, nil
}

// UpdateBridgePort updates an Nvme Subsystem
func (s *server) UpdateBridgePort(_ context.Context, in *pb.UpdateBridgePortRequest) (*pb.BridgePort, error) {
	s.log.WithField("UpdateBridgePortRequest", in).Info("UpdateBridgePort")
	return &pb.BridgePort{}, nil
}

// GetBridgePort gets an BridgePort
func (s *server) GetBridgePort(_ context.Context, in *pb.GetBridgePortRequest) (*pb.BridgePort, error) {
	s.log.WithField("GetBridgePortRequest", in).Info("GetBridgePort")
	return &pb.BridgePort{Name: in.Name, Spec: &pb.BridgePortSpec{}}, nil
}

// GetBridgePort gets an BridgePort
func (s *server) ListBridgePorts(_ context.Context, in *pb.ListBridgePortsRequest) (*pb.ListBridgePortsResponse, error) {
	s.log.WithField("ListBridgePortsRequest", in).Info("ListBridgePorts")
	return &pb.ListBridgePortsResponse{}, nil
}

func (s *server) getFirstVlanID(bridges []string) int {
	vlanId, err := strconv.Atoi(bridges[0])
	if err != nil {
		s.log.Errorf("unable to parse vlan ID %s. conversion error %s", bridges[0], err)
		return 0
	}
	return vlanId
}
