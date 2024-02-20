package ipuplugin

import (
	"fmt"
	"os/exec"

	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/types"
	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/utils"
	log "github.com/sirupsen/logrus"
)

type ovsBridge struct {
	brName    string
	ovsCliDir string
}

func NewOvsBridgeController(bridge, ovsCliDir string) types.BridgeController {
	return &ovsBridge{
		brName:    bridge,
		ovsCliDir: ovsCliDir,
	}
}

func (b *ovsBridge) EnsureBridgeExists() error {
	// ovs-vsctl --may-exist add-br br-tenant
	createBrParams := []string{"--may-exist", "add-br", b.brName}
	if err := utils.ExecOsCommand(b.ovsCliDir+"/ovs-vsctl", createBrParams...); err != nil {
		return fmt.Errorf("error creating ovs bridge %s with ovs-vsctl command %s", b.brName, err.Error())
	}

	return nil
}

func (b *ovsBridge) AddPort(portName string) error {
	cmd := exec.Command(b.ovsCliDir+"/ovs-vsctl", "add-port", b.brName, portName)
	log.WithField("ovs command", cmd.String()).Debug("adding ovs bridge port")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to add port to the bridge: %w", err)
	}
	log.WithField("portName", portName).Infof("port added to ovs bridge %s", b.brName)
	return nil
}

func (b *ovsBridge) DeletePort(portName string) error {
	cmd := exec.Command(b.ovsCliDir+"/ovs-vsctl", "del-port", b.brName, portName)
	log.WithField("ovs command", cmd.String()).Debug("deleting ovs bridge port")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to delete port from the bridge: %w", err)
	}
	log.WithField("portName", portName).Infof("port deleted from ovs bridge %s", b.brName)
	return nil
}
