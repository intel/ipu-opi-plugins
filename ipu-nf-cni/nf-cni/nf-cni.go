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
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/vishvananda/netlink"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "main/build"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ipam"
	"github.com/containernetworking/plugins/pkg/ns"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
)

// NetConf for nf-cni config, look the README to learn how to use those parameters
type NetConf struct {
	types.NetConf
	Device            string `json:"deviceID"` // Device-Name, something like eth0 or can0 etc.
	IpuManagerAddress string `json:"address"`  // Address to update the IPU Manager about the new APF
}

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

func loadConf(bytes []byte) (*NetConf, error) {
	n := &NetConf{}
	var err error
	if err = json.Unmarshal(bytes, n); err != nil {
		return nil, fmt.Errorf("failed to load netconf: %v", err)
	}

	if n.Device == "" {
		return nil, fmt.Errorf(`specify "deviceID"`)
	}

	return n, nil
}

func cmdAdd(args *skel.CmdArgs) error {
	cfg, err := loadConf(args.StdinData)
	if err != nil {
		return err
	}
	containerNs, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer containerNs.Close()

	result := &current.Result{}
	var contDev netlink.Link
	hostDev, err := getLink(cfg.Device)
	if err != nil {
		return fmt.Errorf("failed to find host device: %v", err)
	}

	contDev, err = moveLinkIn(hostDev, containerNs, args.IfName)
	if err != nil {
		return fmt.Errorf("failed to move link %v", err)
	}

	result.Interfaces = []*current.Interface{{
		Name:    contDev.Attrs().Name,
		Mac:     contDev.Attrs().HardwareAddr.String(),
		Sandbox: containerNs.Path(),
	}}
	if cfg.IPAM.Type == "" {
		return printLink(contDev, cfg.CNIVersion, containerNs)
	}

	// run the IPAM plugin and get back the config to apply
	r, err := ipam.ExecAdd(cfg.IPAM.Type, args.StdinData)
	if err != nil {
		return err
	}

	// Invoke ipam del if err to avoid ip leak
	defer func() {
		if err != nil {
			ipam.ExecDel(cfg.IPAM.Type, args.StdinData)
		}
	}()

	// Convert whatever the IPAM result was into the current Result type
	newResult, err := current.NewResultFromResult(r)
	if err != nil {
		return err
	}

	if len(newResult.IPs) == 0 {
		return errors.New("IPAM plugin returned missing IP config")
	}

	for _, ipc := range newResult.IPs {
		// All addresses apply to the container interface (move from host)
		ipc.Interface = current.Int(0)
	}

	newResult.Interfaces = result.Interfaces

	err = containerNs.Do(func(_ ns.NetNS) error {
		return ipam.ConfigureIface(args.IfName, newResult)
	})
	if err != nil {
		return err
	}

	newResult.DNS = cfg.DNS

	if err = createIpuManagerRequest(cfg, contDev.Attrs().Name, contDev.Attrs().HardwareAddr.String()); err != nil {
		return fmt.Errorf("failed to reach IPU Manager: %v", err)
	}

	return types.PrintResult(newResult, cfg.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	cfg, err := loadConf(args.StdinData)
	if err != nil {
		return err
	}
	if args.Netns == "" {
		return nil
	}
	containerNs, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer containerNs.Close()

	if cfg.IPAM.Type != "" {
		if err := ipam.ExecDel(cfg.IPAM.Type, args.StdinData); err != nil {
			return err
		}
	}

	if err := moveLinkOut(containerNs, args.IfName); err != nil {
		return err
	}

	return nil
}

func moveLinkIn(hostDev netlink.Link, containerNs ns.NetNS, ifName string) (netlink.Link, error) {
	if err := netlink.LinkSetNsFd(hostDev, int(containerNs.Fd())); err != nil {
		return nil, err
	}

	var contDev netlink.Link
	if err := containerNs.Do(func(_ ns.NetNS) error {
		var err error
		contDev, err = netlink.LinkByName(hostDev.Attrs().Name)
		if err != nil {
			return fmt.Errorf("failed to find %q: %v", hostDev.Attrs().Name, err)
		}
		// Devices can be renamed only when down
		if err = netlink.LinkSetDown(contDev); err != nil {
			return fmt.Errorf("failed to set %q down: %v", hostDev.Attrs().Name, err)
		}
		// Save host device name into the container device's alias property
		if err := netlink.LinkSetAlias(contDev, hostDev.Attrs().Name); err != nil {
			return fmt.Errorf("failed to set alias to %q: %v", hostDev.Attrs().Name, err)
		}
		// Rename container device to respect args.IfName
		if err := netlink.LinkSetName(contDev, ifName); err != nil {
			return fmt.Errorf("failed to rename device %q to %q: %v", hostDev.Attrs().Name, ifName, err)
		}
		// Bring container device up
		if err = netlink.LinkSetUp(contDev); err != nil {
			return fmt.Errorf("failed to set %q up: %v", ifName, err)
		}
		// Retrieve link again to get up-to-date name and attributes
		contDev, err = netlink.LinkByName(ifName)
		if err != nil {
			return fmt.Errorf("failed to find %q: %v", ifName, err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return contDev, nil
}

func moveLinkOut(containerNs ns.NetNS, ifName string) error {
	defaultNs, err := ns.GetCurrentNS()
	if err != nil {
		return err
	}
	defer defaultNs.Close()

	return containerNs.Do(func(_ ns.NetNS) error {
		dev, err := netlink.LinkByName(ifName)
		if err != nil {
			return fmt.Errorf("failed to find %q: %v", ifName, err)
		}

		// Devices can be renamed only when down
		if err = netlink.LinkSetDown(dev); err != nil {
			return fmt.Errorf("failed to set %q down: %v", ifName, err)
		}

		defer func() {
			// If moving the device to the host namespace fails, set its name back to ifName so that this
			// function can be retried. Also bring the device back up, unless it was already down before.
			if err != nil {
				_ = netlink.LinkSetName(dev, ifName)
				if dev.Attrs().Flags&net.FlagUp == net.FlagUp {
					_ = netlink.LinkSetUp(dev)
				}
			}
		}()

		// Rename the device to its original name from the host namespace
		if err = netlink.LinkSetName(dev, dev.Attrs().Alias); err != nil {
			return fmt.Errorf("failed to restore %q to original name %q: %v", ifName, dev.Attrs().Alias, err)
		}

		if err = netlink.LinkSetNsFd(dev, int(defaultNs.Fd())); err != nil {
			return fmt.Errorf("failed to move %q to host netns: %v", dev.Attrs().Alias, err)
		}
		return nil
	})
}

func printLink(dev netlink.Link, cniVersion string, containerNs ns.NetNS) error {
	result := current.Result{
		CNIVersion: current.ImplementedSpecVersion,
		Interfaces: []*current.Interface{
			{
				Name:    dev.Attrs().Name,
				Mac:     dev.Attrs().HardwareAddr.String(),
				Sandbox: containerNs.Path(),
			},
		},
	}
	return types.PrintResult(&result, cniVersion)
}

func getLink(devname string) (netlink.Link, error) {
	if len(devname) > 0 {
		return netlink.LinkByName(devname)
	}

	return nil, fmt.Errorf("failed to find physical interface %s", devname)
}

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("nf-cni"))
}

func cmdCheck(args *skel.CmdArgs) error {
	return nil
}

func createIpuManagerRequest(cfg *NetConf, deviceID, mac string) error {

	conn, err := grpc.Dial(cfg.IpuManagerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewIpuManagerServiceClient(conn)

	context, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	createRequest := &pb.IpuManagerRequest{
		DeviceID: &deviceID,
		HwAddr:   &mac,
	}

	if _, err := c.CreateIpuManagerRequest(context, createRequest); err != nil {
		return fmt.Errorf("could not create Ipu Manager request: %v", err)
	}

	return nil
}
