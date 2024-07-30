package ipuplugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/types"
	pb "github.com/openshift/dpu-operator/dpu-api/gen"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type DevicePluginService struct {
	pb.UnimplementedDeviceServiceServer
	mode string
}

var (
	exclude     = []string{"enp0s1f0", "enp0s1f0d1", "enp0s1f0d2", "enp0s1f0d3"}
	sysClassNet = "/sys/class/net"
	deviceCode  = "0x1452"
	deviceCodeVf = "0x145c"
)

func NewDevicePluginService(mode string) *DevicePluginService {
	return &DevicePluginService{mode: mode}
}

func (s *DevicePluginService) GetDevices(context.Context, *pb.Empty) (*pb.DeviceListResponse, error) {

	devices, err := discoverHostDevices(s.mode)
	if err != nil {
		return &pb.DeviceListResponse{}, err
	}

	response := &pb.DeviceListResponse{
		Devices: devices,
	}

	fmt.Println(response)
	return response, nil
}

func discoverHostDevices(mode string) (map[string]*pb.Device, error) {

	devices := make(map[string]*pb.Device)

	files, err := os.ReadDir(sysClassNet)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*pb.Device), nil
		}
	}

	for _, file := range files {
		deviceCodeByte, err := os.ReadFile(filepath.Join(sysClassNet, file.Name(), "device/device"))
		if err != nil {
			fmt.Printf("Error: %s\n", err)
		}

		device_code := strings.TrimSpace(string(deviceCodeByte))
		if mode == types.IpuMode {
			if device_code == deviceCode {
				if !slices.Contains(exclude, file.Name()) {
					devices[file.Name()] = &pb.Device{ID: file.Name(), Health: pluginapi.Healthy}
				}
			}
		} else if mode == types.HostMode {
			if device_code == deviceCodeVf {
				devices[file.Name()] = &pb.Device{ID: file.Name(), Health: pluginapi.Healthy}
			}
		}
	}
	return devices, nil
}
