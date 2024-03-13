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
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/types"
	pb "github.com/openshift/dpu-operator/dpu-api/gen"
	"github.com/vishvananda/netlink"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LifeCycleServiceServer struct {
	pb.UnimplementedLifeCycleServiceServer
	daemonHostIp string
	daemonIpuIp  string
	daemonPort   int
	mode         string
}

const (
	hostVportId = "03"
	accVportId  = "02"
	deviceId    = "0x1452"
	vendorId    = "0x8086"
)

func NewLifeCycleService(daemonHostIp, daemonIpuIp string, daemonPort int, mode string) *LifeCycleServiceServer {
	return &LifeCycleServiceServer{
		daemonHostIp: daemonHostIp,
		daemonIpuIp:  daemonIpuIp,
		daemonPort:   daemonPort,
		mode:         mode,
	}
}

type NetworkHandler interface {
	AddrAdd(link netlink.Link, addr *netlink.Addr) error
	AddrList(link netlink.Link, family int) ([]netlink.Addr, error)
	LinkList() ([]netlink.Link, error)
}

type NetworkHandlerImpl struct{}

func (h *NetworkHandlerImpl) AddrAdd(link netlink.Link, addr *netlink.Addr) error {
	return netlink.AddrAdd(link, addr)
}
func (h *NetworkHandlerImpl) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	return netlink.AddrList(link, family)
}
func (h *NetworkHandlerImpl) LinkList() ([]netlink.Link, error) {
	return netlink.LinkList()
}

type FileSystemHandler interface {
	GetDevice(iface string) ([]byte, error)
	GetVendor(iface string) ([]byte, error)
}
type FileSystemHandlerImpl struct{}

func (fs *FileSystemHandlerImpl) GetDevice(iface string) ([]byte, error) {
	return os.ReadFile(fmt.Sprintf("/sys/class/net/%s/device/device", iface))
}
func (fs *FileSystemHandlerImpl) GetVendor(iface string) ([]byte, error) {
	return os.ReadFile(fmt.Sprintf("/sys/class/net/%s/device/vendor", iface))
}

var fileSystemHandler FileSystemHandler
var networkHandler NetworkHandler

func initHandlers() {
	if fileSystemHandler == nil {
		fileSystemHandler = &FileSystemHandlerImpl{}
	}
	if networkHandler == nil {
		networkHandler = &NetworkHandlerImpl{}
	}
}

func isPF(iface string) (bool, error) {
	device, err := fileSystemHandler.GetDevice(iface)
	if err != nil {
		return false, fmt.Errorf("cannot identify device with code: %s; error %v ", deviceId, err.Error())
	}

	vendor, err := fileSystemHandler.GetVendor(iface)
	if err != nil {
		return false, fmt.Errorf("cannot identify vendor device with code: %s; error %v ", vendorId, err.Error())
	}

	return strings.TrimSpace(string(device)) == deviceId && strings.TrimSpace(string(vendor)) == vendorId, nil
}

func getCommPf(mode string, linkList []netlink.Link) (netlink.Link, error) {
	var pf netlink.Link
	for i := 0; i < len(linkList); i++ {
		mac := linkList[i].Attrs().HardwareAddr.String()
		octets := strings.Split(mac, ":")

		if mode == types.IpuMode {

			// Check the 4th octet which is used to identify the PF
			if octets[3] == accVportId {

				// On ACC, the 4th octet in the base mac address may already be set to accVportId and used by
				// the another APF (i.e., the first one). If it is the first APF, then it already has an IP.
				// Two distinguish between the two, we select the one which doesn't have an IP set already.
				if list, _ := networkHandler.AddrList(linkList[i], netlink.FAMILY_V4); len(list) == 0 {
					pf = linkList[i]
					break
				}
			}
		} else {

			// Check the 4th octet which is used to identify the PF
			if octets[3] == hostVportId {
				pf = linkList[i]
				break
			}
		}
	}

	if pf == nil {
		return nil, fmt.Errorf("unable to identify any PFs. Check that the idpf driver is available")
	}

	return pf, nil
}

func setIP(link netlink.Link, ip string) error {
	list, err := networkHandler.AddrList(link, netlink.FAMILY_V4)

	if err != nil {
		return fmt.Errorf("unable to get the ip address of link: %v", err)
	}

	if len(list) == 0 {

		ipAddr := net.ParseIP(ip)

		if ipAddr.To4() == nil {
			return fmt.Errorf("not a valid IPv4 address: %v", err)
		}

		// Set the IP address on PF
		addr := &netlink.Addr{IPNet: &net.IPNet{IP: ipAddr, Mask: net.CIDRMask(24, 32)}}

		if err = networkHandler.AddrAdd(link, addr); err != nil {
			return fmt.Errorf("unable to add address: %v", err)
		}
	} else {
		return fmt.Errorf("address already set. Unset ip address for interface %s and run again", link.Attrs().Name)
	}

	return nil
}

func getFilteredPFs(pfList *[]netlink.Link) error {

	linkList, err := networkHandler.LinkList()

	if err != nil || len(linkList) == 0 {
		return fmt.Errorf("unable to retrieve link list: %v", err)
	}

	for i := 0; i < len(linkList); i++ {
		result, err := isPF(linkList[i].Attrs().Name)

		if result && err == nil {
			*pfList = append(*pfList, linkList[i])
		}
	}

	return nil
}

func configureChannel(mode, daemonHostIp, daemonIpuIp string) error {
	initHandlers()

	var pfList []netlink.Link

	if err := getFilteredPFs(&pfList); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	pf, err := getCommPf(mode, pfList)

	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	var ip string

	if mode == "ipu" {
		ip = daemonHostIp
	} else {
		ip = daemonIpuIp
	}

	if err := setIP(pf, ip); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}

func (s *LifeCycleServiceServer) Init(ctx context.Context, in *pb.InitRequest) (*pb.IpPort, error) {
	if in.DpuMode && s.mode != types.IpuMode || !in.DpuMode && s.mode != types.HostMode {
		return nil, status.Errorf(codes.Internal, "Ipu plugin running in %s mode", s.mode)
	}

	if err := configureChannel(s.mode, s.daemonHostIp, s.daemonIpuIp); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := &pb.IpPort{Ip: s.daemonIpuIp, Port: int32(s.daemonPort)}

	return response, nil
}
