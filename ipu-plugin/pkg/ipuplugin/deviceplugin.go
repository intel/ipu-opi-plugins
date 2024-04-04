package ipuplugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	pb2 "github.com/openshift/dpu-operator/tree/main/dpu-api/gen"
	"google.golang.org/protobuf/types/known/emptypb"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type DevicePluginService struct {
	pb2.UnimplementedDeviceServiceServer
}

var (
	exclude     = []string{"enp0s1f0", "enp0s1f0d2"}
	sysClassNet = "/sys/class/net"
	deviceCode  = "0x1452"
)

func NewDevicePluginService(list map[string]pluginapi.Device) *DevicePluginService {
	return &DevicePluginService{}
}

func (*DevicePluginService) GetDevices(context.Context, *emptypb.Empty) (*pb2.DeviceListResponse, error) {

	devices, err := discoverHostDevices()
	if err != nil {
		return &pb2.DeviceListResponse{}, err
	}

	response := &pb2.DeviceListResponse{
		Devices: devices,
	}

	fmt.Println(response)
	return response, nil
}

func discoverHostDevices() (map[string]*pb2.Device, error) {

	devices := make(map[string]*pb2.Device)

	files, err := os.ReadDir(sysClassNet)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*pb2.Device), nil
		}
	}

	for _, file := range files {
		deviceCodeByte, err := os.ReadFile(filepath.Join(sysClassNet, file.Name(), "device/device"))
		if err != nil {
			fmt.Printf("Error: %s\n", err)
		}

		device_code := strings.TrimSpace(string(deviceCodeByte))

		if device_code == deviceCode {
			if !slices.Contains(exclude, file.Name()) {
				devices[file.Name()] = &pb2.Device{ID: file.Name(), Health: pluginapi.Healthy}
			}
		}
	}
	return devices, nil
}
