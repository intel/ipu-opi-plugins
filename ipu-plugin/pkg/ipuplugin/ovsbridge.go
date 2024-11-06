package ipuplugin

import (
	"fmt"
	"os/exec"

	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/types"
	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/utils"
	log "github.com/sirupsen/logrus"
)

type ovsBridge struct {
	bridgeName string
	ovsCliDir  string
}

const (
	ACC_VM_PR_IP = "192.168.100.252/24"
)

func NewOvsBridgeController(bridgeName, ovsCliDir string) types.BridgeController {
	return &ovsBridge{
		bridgeName: bridgeName,
		ovsCliDir:  ovsCliDir,
	}
}

func (b *ovsBridge) EnsureBridgeExists() error {
	// ovs-vsctl --may-exist add-br br-infra
	createBrParams := []string{"--may-exist", "add-br", b.bridgeName}
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
	brParams := []string{"--may-exist", "del-br", b.bridgeName}
	if err := utils.ExecOsCommand(b.ovsCliDir+"/ovs-vsctl", brParams...); err != nil {
		log.Errorf("error deleting ovs bridge %s with ovs-vsctl command %s", b.bridgeName, err.Error())
	}
	return nil
}

func (b *ovsBridge) AddPort(portName string) error {
	cmd := exec.Command(b.ovsCliDir+"/ovs-vsctl", "add-port", b.bridgeName, portName)
	log.WithField("ovs command", cmd.String()).Debug("adding ovs bridge port")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to add port to the bridge: %w", err)
	}
	log.WithField("portName", portName).Infof("port added to ovs bridge %s", b.bridgeName)
	return nil
}

func (b *ovsBridge) DeletePort(portName string) error {
	cmd := exec.Command(b.ovsCliDir+"/ovs-vsctl", "del-port", b.bridgeName, portName)
	log.WithField("ovs command", cmd.String()).Debug("deleting ovs bridge port")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to delete port from the bridge: %w", err)
	}
	log.WithField("portName", portName).Infof("port deleted from ovs bridge %s", b.bridgeName)
	return nil
}
