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
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/types"
	pb "github.com/openshift/dpu-operator/dpu-api/gen"
	"github.com/pkg/sftp"
	"github.com/vishvananda/netlink"
	"golang.org/x/crypto/ssh"
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

type ExecutableHandler interface {
	validate() bool
}

type ExecutableHandlerImpl struct{}

type SSHHandler interface {
	sshFunc() error
}

type SSHHandlerImpl struct{}

var fileSystemHandler FileSystemHandler
var networkHandler NetworkHandler
var executableHandler ExecutableHandler
var sshHander SSHHandler

func initHandlers() {
	if fileSystemHandler == nil {
		fileSystemHandler = &FileSystemHandlerImpl{}
	}
	if networkHandler == nil {
		networkHandler = &NetworkHandlerImpl{}
	}
	if executableHandler == nil {
		executableHandler = &ExecutableHandlerImpl{}
	}
	if sshHander == nil {
		sshHander = &SSHHandlerImpl{}
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

				if list, _ := networkHandler.AddrList(linkList[i], netlink.FAMILY_V4); len(list) == 0 {
					pf = linkList[i]
					break
				}
			}
		}
	}

	if pf == nil {
		return nil, fmt.Errorf("check if the ip address already set")
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

	var pfList []netlink.Link

	if err := getFilteredPFs(&pfList); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	pf, err := getCommPf(mode, pfList)

	if pf == nil {
		// Address already set - we don't proceed with setting the ip
		return nil
	}

	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	var ip string

	if mode == "ipu" {
		ip = daemonIpuIp
	} else {
		ip = daemonHostIp
	}

	if err := setIP(pf, ip); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}

func (s *SSHHandlerImpl) sshFunc() error {
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password(""),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the remote server.
	client, err := ssh.Dial("tcp", "192.168.0.1:22", config)
	if err != nil {
		return fmt.Errorf("failed to dial: %s", err)
	}
	defer client.Close()

	// Create an SFTP client.
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %s", err)
	}
	defer sftpClient.Close()

	// Open the source file.
	localFilePath := "/linux_networking.pkg"
	srcFile, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %s", err)
	}
	defer srcFile.Close()

	// Create the destination file on the remote server.
	remoteFilePath := "/work/scripts/linux_networking.pkg"
	dstFile, err := sftpClient.Create(remoteFilePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %s", err)
	}
	defer dstFile.Close()

	// Copy the file contents to the destination file.
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %s", err)
	}

	// Ensure that the file is written to the remote filesystem.
	err = dstFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync file: %s", err)
	}

	fileShPath := "/work/scripts/pre_init_app.sh"
	initFile, err := sftpClient.Create(fileShPath)
	if err != nil {
		return fmt.Errorf("failed to create remote file.sh: %s", err)
	}
	defer initFile.Close()

	initScript := `#!/bin/sh
CURDIR=$(pwd)
WORKDIR=$(dirname $(realpath $0))
cd $WORKDIR	
	if [ -e load_custom_pkg.sh ]; then
	# Fix up the cp_init.cfg file
	./load_custom_pkg.sh
	fi
	python /usr/bin/scripts/cfg_acc_apf_x2.py
fi
cd $CURDIR
`
	_, err = initFile.Write([]byte(initScript))
	if err != nil {
		return fmt.Errorf("failed to write to file.sh: %s", err)
	}

	err = initFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync file.sh: %s", err)
	}

	// Start a session.
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %s", err)
	}
	defer session.Close()

	shellScript := `#!/bin/sh
	CP_INIT_CFG=/etc/dpcp/cfg/cp_init.cfg
	echo "Checking for custom package..."
	if [ -e linux_networking.pkg ]; then
		echo "Custom package linux_networking.pkg found. Overriding default package"
		cp  linux_networking.pkg /etc/dpcp/package/
		rm -rf /etc/dpcp/package/default_pkg.pkg
		ln -s /etc/dpcp/package/linux_networking.pkg /etc/dpcp/package/default_pkg.pkg
		sed -i 's/sem_num_pages = 1;/sem_num_pages = 25;/g' $CP_INIT_CFG
		sed -i 's/pf_mac_address = "00:00:00:00:03:14";/pf_mac_address = "00:00:00:0a:03:15";/g' $CP_INIT_CFG
		sed -i 's/acc_apf = 4;/acc_apf = 19;/g' $CP_INIT_CFG
		sed -i 's/comm_vports = ((\[5,0\],\[4,0\]));/comm_vports = ((\[5,0\],\[4,0\]),(\[0,3\],\[4,2\]));/g' $CP_INIT_CFG
	else
		echo "No custom package found. Continuing with default package"
	fi
	`

	commands := fmt.Sprintf(`
	cat <<'EOF' > /work/scripts/load_custom_pkg.sh
	%s
	EOF
	`, shellScript)

	// Run a command on the remote server and capture the output.
	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	err = session.Run(commands)
	if err != nil {
		return fmt.Errorf("failed to run commands: %s", err)
	}
	fmt.Println(stdoutBuf.String())

	session, err = client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %s", err)
	}
	defer session.Close()

	err = session.Run("reboot")
	if err != nil {
		return fmt.Errorf("failed to run commands: %s", err)
	}
	fmt.Println(stdoutBuf.String())

	return nil
}

// refactor and use getFilteredPFs()
func countAPFDevices() int {
	var pfList []netlink.Link

	if err := getFilteredPFs(&pfList); err != nil {
		return 0
	}

	return len(pfList)
}

func formatMAC(mac string) (string, error) {
	if len(mac) != 12 {
		return "", fmt.Errorf("invalid MAC address length")
	}

	var pairs []string
	for i := 0; i < len(mac); i += 2 {
		pairs = append(pairs, mac[i:i+2])
	}

	formattedMAC := strings.Join(pairs, ":")

	return formattedMAC, nil
}

func checkMAC() (bool, string) {
	cmd := exec.Command("sh", "-c", "dmidecode | sha256sum | head -c 12")
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Println(err.Error())
		return false, ""
	}
	fmt.Println(string(stdout))

	mac, err := formatMAC(string(stdout))
	fmt.Println(mac)
	if err != nil {
		return false, fmt.Sprintf("error: %s", err)
	}

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password(""),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the remote server.
	client, err := ssh.Dial("tcp", "192.168.0.1:22", config)
	if err != nil {
		return false, fmt.Sprintf("failed to dial remote server: %s", err)
	}
	defer client.Close()

	// Start a session.
	session, err := client.NewSession()
	if err != nil {
		return false, fmt.Sprintf("failed to create session: %s", err)
	}
	defer session.Close()

	mac = "00:00:00:00:03:14"
	// commands := fmt.Sprintf("grep '%s' /etc/dpcp/cfg/cp_init.cfg", "00:00:00:00:03:14")
	commands := fmt.Sprintf("grep '%s' /etc/dpcp/cfg/cp_init.cfg", mac)

	// Run a command on the remote server and capture the output.
	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	err = session.Run(commands)
	if err != nil {
		return false, fmt.Sprintf("mac not found: %s", err)
	}

	fmt.Println(stdoutBuf.String())

	if stdoutBuf.Len() > 0 {
		return true, stdoutBuf.String()
	}

	return false, "mac not found"
}

func (e *ExecutableHandlerImpl) validate() bool {

	if numAPFs := countAPFDevices(); numAPFs < 19 {
		fmt.Printf("Not enough APFs %v", numAPFs)
		return false
	}

	if macPreFix, mac := checkMAC(); !macPreFix {
		fmt.Printf("incorrect Mac assigned : %v\n", mac)
		return false
	}

	return true
}

func (s *LifeCycleServiceServer) Init(ctx context.Context, in *pb.InitRequest) (*pb.IpPort, error) {
	initHandlers()

	if val := executableHandler.validate(); !val {
		fmt.Println("forcing state")
		if err := sshHander.sshFunc(); err != nil {
			return nil, fmt.Errorf("error calling sshFunc %s", err)
		}
	} else {
		fmt.Println("not forcing state")
	}

	if in.DpuMode && s.mode != types.IpuMode || !in.DpuMode && s.mode != types.HostMode {
		return nil, status.Errorf(codes.Internal, "Ipu plugin running in %s mode", s.mode)
	}

	if err := configureChannel(s.mode, s.daemonHostIp, s.daemonIpuIp); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := &pb.IpPort{Ip: s.daemonIpuIp, Port: int32(s.daemonPort)}

	return response, nil
}
