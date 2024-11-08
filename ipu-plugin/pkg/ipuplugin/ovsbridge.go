package ipuplugin

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/types"
	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/utils"
	log "github.com/sirupsen/logrus"
)

type ovsBridge struct {
	bridgeName string
	ovsCliDir  string
	ovsDbPath  string
}

const (
	ACC_VM_PR_IP = "192.168.100.252/24"
)

func NewOvsBridgeController(bridgeName, ovsCliDir string, ovsDbPath string) types.BridgeController {
	return &ovsBridge{
		bridgeName: bridgeName,
		ovsCliDir:  ovsCliDir,
		ovsDbPath:  ovsDbPath,
	}
}

func createDbParam(ovsDbPath string) string {
	return "--db=unix:" + ovsDbPath
}

func getInfraPodNamespace() string {
	var buffer bytes.Buffer

	nsCmdParams := []string{"-a", "|", "grep", "entrypoint.sh", "|", "head", "-n", "1",
		"|", "awk", "'{print $1;}'", "|", "xargs", "ip", "netns", "identify"}
	log.Debugf("Namespace command= ps %s", nsCmdParams)
	cmd := exec.Command("ps", nsCmdParams...)
	cmd.Stdout = &buffer
	err := cmd.Run()
	if err != nil {
		log.Errorf("unable to get Namespace: %v", err)
		return ""
	}
	log.Debugf("Got output = %s", buffer.String())
	return buffer.String()
}

func (b *ovsBridge) EnsureBridgeExists() error {
	// ovs-vsctl --may-exist add-br br-infra
	createBrParams := []string{createDbParam(b.ovsDbPath), "--may-exist", "add-br", b.bridgeName}
	if err := utils.ExecOsCommand(b.ovsCliDir+"/ovs-vsctl", createBrParams...); err != nil {
		return fmt.Errorf("error creating ovs bridge %s with ovs-vsctl command %s", b.bridgeName, err.Error())
	}
	//assigning IP for bridge interface.
	ipAddr := ACC_VM_PR_IP
	cmdParams := []string{"addr", "add", "dev", b.bridgeName, ipAddr}
	if err := utils.ExecOsCommand("ip", cmdParams...); err != nil {
		return fmt.Errorf("error->%v, assigning IP->%v to ovs bridge %s", err.Error(), ipAddr, b.bridgeName)
	}
	//bring the interface up.
	cmdParams = []string{"link", "set", "dev", b.bridgeName, "up"}
	if err := utils.ExecOsCommand("ip", cmdParams...); err != nil {
		return fmt.Errorf("error->%v, bringing UP bridge interface->%v", err.Error(), b.bridgeName)
	}
	return nil
}

// Note:: This is expected to be called, when plugin exits(Stop),
// so continue to delete, without exiting for any error.
// Note: Deleting bridge, automatically deletes any ports added to it.
func (b *ovsBridge) DeleteBridges() error {
	brParams := []string{createDbParam(b.ovsDbPath), "--may-exist", "del-br", b.bridgeName}
	if err := utils.ExecOsCommand(b.ovsCliDir+"/ovs-vsctl", brParams...); err != nil {
		log.Errorf("error deleting ovs bridge %s with ovs-vsctl command %s", b.bridgeName, err.Error())
	}
	return nil
}

func (b *ovsBridge) AddPort(portName string) error {
	ipParams := []string{"-a", "|", "grep", "entrypoint.sh", "|", "head", "-n", "1",
		"|", "awk", "'{print $1;}'", "|", "xargs", "ip", "netns", "identify", "|",
		"xargs", "ip", "link", "set", portName, "netns"}
	log.Errorf("The IP command = ps %s", ipParams)
	err := utils.ExecOsCommand("ps", ipParams...)
	if err != nil {
		log.Errorf("error moving interface %s to infra namespace with error %s", portName, err.Error())
	}

	brParams := []string{createDbParam(b.ovsDbPath), "add-port", b.bridgeName, portName}
	err = utils.ExecOsCommand(b.ovsCliDir+"/ovs-vsctl", brParams...)
	if err != nil {
		return fmt.Errorf("unable to add port to the bridge: %w", err)
	}
	log.WithField("portName", portName).Infof("port added to ovs bridge %s", b.bridgeName)
	return nil
}

func (b *ovsBridge) DeletePort(portName string) error {
	brParams := []string{createDbParam(b.ovsDbPath), "del-port", b.bridgeName, portName}
	err := utils.ExecOsCommand(b.ovsCliDir+"/ovs-vsctl", brParams...)
	if err != nil {
		return fmt.Errorf("unable to delete port from the bridge: %w", err)
	}
	log.WithField("portName", portName).Infof("port deleted from ovs bridge %s", b.bridgeName)
	return nil
}
