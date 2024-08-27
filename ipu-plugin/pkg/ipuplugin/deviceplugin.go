package ipuplugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/types"
	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/utils"
	log "github.com/sirupsen/logrus"

	pb "github.com/openshift/dpu-operator/dpu-api/gen"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type DevicePluginService struct {
	pb.UnimplementedDeviceServiceServer
	mode string
}

var (
	exclude          = []string{"enp0s1f0", "enp0s1f0d1", "enp0s1f0d2", "enp0s1f0d3", "enp0s1f0d4"}
	sysClassNet      = "/sys/class/net"
	sysBusPciDevices = "/sys/bus/pci/devices"
	deviceCode       = "0x1452"
	deviceCodeVf     = "0x145c"
	intelVendor      = "0x8086"
	configNumVfs     = 8
	maxVfsSupported  = 64
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

	log.Debugf("GetDevices, response->%v\n", response)
	return response, nil
}

// Recommended to wait upto 2 seconds, for the max VFs(64) currently supported.
func pollToCheckVfDevices(mode string, count int) error {
	var err error
	cnt := 0
	ticker := time.NewTicker(time.Second / 3)
	done := make(chan bool, 1)

	go func() {
		for _ = range ticker.C {
			cnt, err = utils.GetVfDeviceCount(mode)
			if cnt == count && err == nil {
				ticker.Stop()
				break
			}
		}
		done <- true
	}()

	timer := time.NewTimer(time.Second * 2)
	select {
	case <-done:
		log.Debugf("ticker Done")
		timer.Stop()
	case <-timer.C:
		log.Debugf("timer.C Done")
		ticker.Stop()
	}

	if cnt == count && err == nil {
		log.Debugf("pollToCheckVfDevices: match expected count, %v\n", cnt)
		return nil
	}
	log.Debugf("pollToCheckVfDevices: Timed out:attempt->%v, result->%v, err->%v\n", count, cnt, err)
	return fmt.Errorf("pollToCheckVfDevices: Timed out:attempt->%v, result->%v, err->%v\n", count, cnt, err)
}

func SetNumSriovVfs(mode string, pciAddr string, vfCnt int32) error {

	//Note: Upto 64 VFs have been validated.
	if vfCnt <= 0 || vfCnt > int32(maxVfsSupported) {
		return fmt.Errorf("SetNumSriovVfs(): Invalid/unsupported, vfCnt->%v \n", vfCnt)
	}

	pathToNumVfsFile := filepath.Join(sysBusPciDevices, pciAddr, "sriov_numvfs")

	//Need to first write 0 for num of VFs, before updating it.
	err := os.WriteFile(pathToNumVfsFile, []byte("0"), 0644)
	if err != nil {
		return fmt.Errorf("SetNumSriovVfs(): reset fail %s: %v", pathToNumVfsFile, err)
	}
	zeroVfs := 0
	err = pollToCheckVfDevices(mode, zeroVfs)
	if err != nil {
		return fmt.Errorf("cli-client query failed count->%v, err->%v\n", zeroVfs, err)
	}

	// Note: Post-writing, it can take some time for the VFs to be created.
	err = os.WriteFile(pathToNumVfsFile, []byte(strconv.Itoa(int(vfCnt))), 0644)
	if err != nil {
		return fmt.Errorf("SetNumSriovVfs():error in updating %s: %v", pathToNumVfsFile, err)
	}

	err = pollToCheckVfDevices(mode, int(vfCnt))
	if err != nil {
		return fmt.Errorf("cli-client query failed count->%v, err->%v\n", vfCnt, err)
	}

	log.Debugf("SetNumSriovVfs(): updated file->%s, sriov_numvfs to %v\n", pathToNumVfsFile, vfCnt)

	return nil
}

func SetNumVfs(mode string, numVfs int32) (int32, error) {
	deviceVfsSet := false

	if mode != types.HostMode {
		return 0, fmt.Errorf("setNumVfs(): only supported on host: mode %s\n", mode)
	}

	//Note: Per internal discussion, for now setting VFs to hardcoded value.
	log.Debugf("setNumVfs(): requested VFs->%v, will allocate VFs->%v\n", numVfs, configNumVfs)
	numVfs = int32(configNumVfs)

	files, err := os.ReadDir(sysBusPciDevices)
	if err != nil {
		return 0, fmt.Errorf("setNumVfs(): error-> %v\n", err)
	}

	for _, file := range files {
		deviceByte, err := os.ReadFile(filepath.Join(sysBusPciDevices, file.Name(), "device"))
		if err != nil {
			log.Errorf("Error reading PCIe deviceID: %s\n", err)
			continue
		}

		vendorByte, err := os.ReadFile(filepath.Join(sysBusPciDevices, file.Name(), "vendor"))
		if err != nil {
			log.Errorf("Error reading VendorID: %s\n", err)
			continue
		}

		deviceId := strings.TrimSpace(string(deviceByte))
		vendorId := strings.TrimSpace(string(vendorByte))

		if deviceId == deviceCode && vendorId == intelVendor {
			err = SetNumSriovVfs(mode, file.Name(), numVfs)
			if err != nil {
				return 0, fmt.Errorf("setNumVfs(): error from SetSriovNumVfs-> %v", err)
			}
			deviceVfsSet = true
			break
		}
	}
	if deviceVfsSet == true {
		return numVfs, nil
	} else {
		return 0, fmt.Errorf("setNumVfs(): unable to set VFs for device->%s", deviceCode)
	}
}

func (s *DevicePluginService) SetNumVfs(ctx context.Context, vfCountReq *pb.VfCount) (*pb.VfCount, error) {
	var res *pb.VfCount
	numVfs, err := SetNumVfs(types.HostMode, vfCountReq.VfCnt)

	log.Debugf("setNumVfs(): requested VFs->%v, allocated VFs->%v, err->%v\n", vfCountReq.VfCnt, numVfs, err)
	if err != nil {
		res = &pb.VfCount{VfCnt: 0}
	} else {
		res = &pb.VfCount{VfCnt: numVfs}
	}

	log.Debugf("SetNumVfs res->%v\n", res)
	return res, err
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
