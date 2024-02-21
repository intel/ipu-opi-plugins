// Copyright 2024 Intel Corp. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
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
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/exp/slices"

	"github.com/intel/ipu-device-plugin/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	sysClassNet     = "/sys/class/net"
	excludeFilePath = "/etc/ipu-dp/config.json"

	// Device plugin settings.
	pluginMountPath = "/var/lib/kubelet/device-plugins"
	SockDir         = "/var/lib/kubelet/plugins_registry"
	kubeletEndpoint = "kubelet.sock"
	pluginEndpoint  = "sriovNet.sock"
	resourceName    = "intel.com/ipu"
	prefix          = "intel.com"
	deviceCode      = "0x1452"
)

// cliParams presents CLI parameters for the APF Device Plugin
type cliParams struct {
	configFile string
}

// sriovManager manages sriov networking devices
type apfManager struct {
	cliParams
	socketFile string
	devices    map[string]pluginapi.Device // for Kubelet DP API
	grpcServer *grpc.Server
	pluginapi.DevicePluginServer
}

func newAPFManager(cp *cliParams) *apfManager {

	return &apfManager{
		cliParams:  *cp,
		devices:    make(map[string]pluginapi.Device),
		socketFile: pluginEndpoint,
	}
}

func (sm *apfManager) ListAndWatch(empty *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {
	changed := true
	for {
		for id, dev := range sm.devices {
			state := sm.GetDeviceState(id)
			if dev.Health != state {
				changed = true
				dev.Health = state
				sm.devices[id] = dev
			}
		}
		if changed {
			resp := new(pluginapi.ListAndWatchResponse)
			for _, dev := range sm.devices {
				resp.Devices = append(resp.Devices, &pluginapi.Device{ID: dev.ID, Health: dev.Health})
			}
			fmt.Printf("ListAndWatch: send devices %v\n", resp)
			if err := stream.Send(resp); err != nil {
				fmt.Printf("Error. Cannot update device states: %v\n", err)
				sm.grpcServer.Stop()
				return err
			}
		}
		changed = false
		time.Sleep(5 * time.Second)
	}
}

func (sm *apfManager) discoverHostDevices() error {

	excludeConfig := &types.ExcludeConfig{}

	rawBytes, err := os.ReadFile(sm.cliParams.configFile)

	if err != nil {
		return fmt.Errorf("error reading file %s, %v", sm.cliParams.configFile, err)
	}

	if err = json.Unmarshal(rawBytes, excludeConfig); err != nil {
		return fmt.Errorf("error unmarshalling raw bytes %v please make sure the config is in json format", err)
	}

	files, err := os.ReadDir(sysClassNet)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
	}

	for _, file := range files {
		deviceCodeByte, err := os.ReadFile(filepath.Join(sysClassNet, file.Name(), "device/device"))
		if err != nil {
			fmt.Printf("Error: %s\n", err)
		}

		device_code := strings.TrimSpace(string(deviceCodeByte))

		if device_code == deviceCode {
			if !slices.Contains(excludeConfig.ExcludeDevices, file.Name()) {
				sm.devices[file.Name()] = pluginapi.Device{ID: file.Name(), Health: pluginapi.Healthy}
			}
		}
	}
	return nil
}

func (sm *apfManager) GetDeviceState(DeviceName string) string {
	// TODO: Discover device health
	return pluginapi.Healthy
}

// Allocate passes the dev name as an env variable to the requesting container
func (sm *apfManager) Allocate(ctx context.Context, rqt *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resp := new(pluginapi.AllocateResponse)
	devName := ""
	for _, container := range rqt.ContainerRequests {
		containerResp := new(pluginapi.ContainerAllocateResponse)
		for _, id := range container.DevicesIDs {
			fmt.Printf("DeviceID in Allocate: %v \n", id)
			dev, ok := sm.devices[id]
			if !ok {
				fmt.Printf("Error. Invalid allocation request with non-existing device %s", id)
			}
			if dev.Health != pluginapi.Healthy {
				fmt.Printf("Error. Invalid allocation request with unhealthy device %s", id)
			}

			devName = devName + id + ","
		}

		fmt.Printf("device(s) allocated: %s\n", devName)
		envmap := make(map[string]string)
		envmap["APF-DEV"] = devName

		containerResp.Envs = envmap
		resp.ContainerResponses = append(resp.ContainerResponses, containerResp)
	}
	return resp, nil
}

// Discover devices
func (sm *apfManager) Start() error {
	fmt.Printf("Discovering device[s]\n")
	if err := sm.discoverHostDevices(); err != nil {
		return err
	}

	pluginEndpoint := filepath.Join(pluginapi.DevicePluginPath, sm.socketFile)
	fmt.Printf("Starting APF Device Plugin server at: %s\n", pluginEndpoint)
	lis, err := net.Listen("unix", pluginEndpoint)
	if err != nil {
		fmt.Printf("Error: Starting APF Device Plugin server failed: %v", err)
	}
	sm.grpcServer = grpc.NewServer()

	kubeletEndpoint := filepath.Join("unix:", types.DeprecatedSockDir, types.KubeEndPoint)

	conn, err := grpc.Dial(kubeletEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		fmt.Printf("%s device plugin unable connect to Kubelet : %v", resourceName, err)
		return err
	}
	defer conn.Close()

	pluginapi.RegisterDevicePluginServer(sm.grpcServer, sm)

	client := pluginapi.NewRegistrationClient(conn)

	go sm.grpcServer.Serve(lis)

	// Wait for server to start by launching a blocking connection
	ctx, _ := context.WithTimeout(context.TODO(), 5*time.Second)
	conn, err = grpc.DialContext(
		ctx, "unix:"+pluginEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())

	if err != nil {
		fmt.Printf("error. unable to establish test connection with %s gRPC server: %v", resourceName, err)
		return err
	}
	fmt.Printf("%s device plugin endpoint started serving \n", resourceName)
	conn.Close()

	ctx = context.Background()

	request := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     sm.socketFile,
		ResourceName: resourceName,
	}

	if _, err = client.Register(ctx, request); err != nil {
		fmt.Printf("%s device plugin unable to register with Kubelet : %v \n", resourceName, err)
		return err
	}
	fmt.Printf("%s device plugin registered with Kubelet\n", resourceName)
	return nil
}

func (sm *apfManager) Stop() error {
	fmt.Printf("Stopping Device Plugin gRPC server..")
	if sm.grpcServer == nil {
		return nil
	}

	sm.grpcServer.Stop()
	sm.grpcServer = nil

	return sm.cleanup()
}

// Removes existing socket if exists
// [adapted from https://github.com/redhat-nfvpe/k8s-dummy-device-plugin/blob/master/dummy.go ]
func (sm *apfManager) cleanup() error {
	pluginEndpoint := filepath.Join(pluginapi.DevicePluginPath, sm.socketFile)
	if err := os.Remove(pluginEndpoint); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (sm *apfManager) PreStartContainer(ctx context.Context, psRqt *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (sm *apfManager) GetDevicePluginOptions(ctx context.Context, empty *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		PreStartRequired: false,
	}, nil
}

func flagInit(cp *cliParams) {
	flag.StringVar(&cp.configFile, "config-file", excludeFilePath,
		"JSON exclude config file location")
}

func main() {
	cp := &cliParams{}
	flagInit(cp)
	flag.Parse()

	sm := newAPFManager(cp)
	if sm == nil {
		fmt.Printf("Unable to get instance of a APF-Manager")
		return
	}
	sm.cleanup()

	// respond to syscalls for termination
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Start server
	if err := sm.Start(); err != nil {
		fmt.Printf("\nsriovManager.Start() failed: %v \n", err)
		return
	}

	// Catch termination signals
	select {
	case sig := <-sigCh:
		fmt.Printf("Received signal \"%v\", shutting down.", sig)
		sm.Stop()
		return
	}

}
