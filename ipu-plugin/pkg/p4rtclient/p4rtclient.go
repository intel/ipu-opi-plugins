// Copyright (c) 2023 Intel Corporation.  All Rights Reserved.
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

package p4rtclient

import (
	"fmt"
	"net"
	"errors"

	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/types"
	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/utils"
	log "github.com/sirupsen/logrus"
)

type p4rtclient struct {
	p4RtBin    string
	portMuxVsi int
	p4br       string
	bridgeType types.BridgeType
}

type fxpRuleBuilder struct {
        Action     string
        P4br       string
        Control    string
        Metadata   string
}

type fxpRuleParams []string

func NewP4RtClient(p4RtBin string, portMuxVsi int, p4BridgeName string, brType types.BridgeType) types.P4RTClient {
	log.Debug("Creating Linux P4Client instance")
	return &p4rtclient{
		p4RtBin:    p4RtBin,
		portMuxVsi: portMuxVsi,
		p4br:       p4BridgeName,
		bridgeType: brType,
	}
}

//TODO: Move this under utils pkg
func checkMacAddresses(macAddresses ...string) ([]byte, error) {
	for _, mac := range macAddresses {
            hwAddr, err := net.ParseMAC(mac)
            if err !=nil {
	         return hwAddr, errors.New("Invalid Mac Address format")
            }
        }
        return []byte{}, nil
}

//TODO: Move this under utils pkg
func programFXPP4Rules(p4RtBin string, ruleSets []fxpRuleBuilder) error {
        for _, r := range ruleSets {
		p4rule := []string{r.Action, r.P4br, r.Control, r.Metadata}
                err := utils.RunP4rtCtlCommand(p4RtBin, p4rule...)
		if err != nil {
			log.Info("WARNING: Failed to program p4rule: %v", p4rule)
                        return err
                }
        }
	return nil
}

//TODO: Move this under utils pkg
func getVsiVportInfo(macAddr string) (int, int) {
	vfVsi := int(macAddr[1])
        vfVport := utils.GetVportForVsi(vfVsi)
	return vfVsi, vfVport
}

func programPhyVportP4Rules(p4RtBin string, phyPort int, prMac string) error {
	_ , prVport := getVsiVportInfo(prMac)
        phyVportP4ruleSets := []fxpRuleBuilder{
                {
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.rx_source_port",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.port_id=%d,zero_padding=0,action=linux_networking_control.set_source_port(%d)",
                                phyPort, phyPort,
                        ),
                },
                {
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.rx_phy_port_to_pr_map",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.port_id=%d,zero_padding=0,action=linux_networking_control.fwd_to_vsi(%d)",
                                phyPort, prVport,
                        ),
                },
                {
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.source_port_to_pr_map",
                        Metadata: fmt.Sprintf(
                                "user_meta.cmeta.source_port=%d,zero_padding=0,action=linux_networking_control.fwd_to_vsi(%d)",
                                phyPort, prVport,
                        ),
                },
                {
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.tx_acc_vsi",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,zero_padding=0,action=linux_networking_control.l2_fwd_and_bypass_bridge(%d)",
                                prVport, phyPort,
                        ),
                },

        }
        return programFXPP4Rules(p4RtBin, phyVportP4ruleSets)
}

func deletePhyVportP4Rules(p4RtBin string, phyPort int, prMac string) error {
        _ , prVport := getVsiVportInfo(prMac)
        phyVportP4ruleSets := []fxpRuleBuilder{
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.rx_source_port",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.port_id=%d,zero_padding=0",
                                phyPort,
                        ),
                },
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.rx_phy_port_to_pr_map",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.port_id=%d,zero_padding=0",
                                phyPort,
                        ),
                },
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.source_port_to_pr_map",
                        Metadata: fmt.Sprintf(
                                "user_meta.cmeta.source_port=%d,zero_padding=0",
                                phyPort,
                        ),
                },
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.tx_acc_vsi",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,zero_padding=0",
                                prVport,
                        ),
                },

        }

        return programFXPP4Rules(p4RtBin, phyVportP4ruleSets)
}

func programPhyVportBridgeId(p4RtBin string, phyPort, bridgeId int) error {
        phyBridgeIdP4ruleSet := []fxpRuleBuilder{
                {
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.source_port_to_bridge_map",
                        Metadata: fmt.Sprintf(
                                "user_meta.cmeta.source_port=%d/0xffff,hdrs.vlan_ext[vmeta.common.depth].hdr.vid=0/0xfff,priority=1,action=linux_networking_control.set_bridge_id(bridge_id=%d)",
                                phyPort, bridgeId,
                        ),
                },
	}
        return programFXPP4Rules(p4RtBin, phyBridgeIdP4ruleSet)
}

func deletePhyVportBridgeId(p4RtBin string, phyPort, bridgeId int) error {
        phyBridgeIdP4ruleSet := []fxpRuleBuilder{
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.source_port_to_bridge_map",
                        Metadata: fmt.Sprintf(
                                "user_meta.cmeta.source_port=%d/0xffff,hdrs.vlan_ext[vmeta.common.depth].hdr.vid=0/0xfff,priority=1",
                                phyPort,
                        ),
                },
        }
        return programFXPP4Rules(p4RtBin, phyBridgeIdP4ruleSet)
}

func programNfPrVportP4Rules(p4RtBin, ingressMac, egressMac string) error {
        ingressVsi, ingressVport := getVsiVportInfo(ingressMac)
        egressVsi, egressVport := getVsiVportInfo(egressMac)

        nfPrVportP4RuleSets := []fxpRuleBuilder{
                {
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.tx_source_port",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d/0x7ff,priority=1,action=linux_networking_control.set_source_port(%d)",
                                ingressVsi, ingressVport,
                        ),
                },
                {
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.tx_acc_vsi",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,zero_padding=0,action=linux_networking_control.l2_fwd_and_bypass_bridge(%d)",
                                ingressVsi, egressVport,
                        ),
                },
                {
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.tx_acc_vsi",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,zero_padding=0,action=linux_networking_control.l2_fwd_and_bypass_bridge(%d)",
                                egressVsi, ingressVport,
                        ),
                },
                {
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.vsi_to_vsi_loopback",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,target_vsi=%d,action=linux_networking_control.fwd_to_vsi(%d)",
                                egressVsi, ingressVsi, ingressVport,
                        ),
                },
                {
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.vsi_to_vsi_loopback",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,target_vsi=%d,action=linux_networking_control.fwd_to_vsi(%d)",
                                ingressVsi, egressVsi, egressVport,
                        ),
                },
                {
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.source_port_to_pr_map",
                        Metadata: fmt.Sprintf(
                                "user_meta.cmeta.source_port=%d,zero_padding=0,action=linux_networking_control.fwd_to_vsi(%d)",
                                ingressVport, egressVport,
                        ),
                },

        }

        return programFXPP4Rules(p4RtBin, nfPrVportP4RuleSets)
}

func deleteNfPrVportP4Rules(p4RtBin, ingressMac, egressMac string) error {
        ingressVsi, ingressVport := getVsiVportInfo(ingressMac)
        egressVsi, _ := getVsiVportInfo(egressMac)

        nfPrVportP4RuleSets := []fxpRuleBuilder{
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.tx_source_port",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d/0x7ff,priority=1",
                                ingressVsi,
                        ),
                },
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.tx_acc_vsi",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,zero_padding=0",
                                ingressVsi,
                        ),
                },
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.tx_acc_vsi",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,zero_padding=0",
                                egressVsi,
                        ),
                },
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.vsi_to_vsi_loopback",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,target_vsi=%d",
                                egressVsi, ingressVsi,
                        ),
                },
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.vsi_to_vsi_loopback",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,target_vsi=%d",
                                ingressVsi, egressVsi,
                        ),
                },
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.source_port_to_pr_map",
                        Metadata: fmt.Sprintf(
                                "user_meta.cmeta.source_port=%d,zero_padding=0",
                                ingressVport,
                        ),
                },

        }

        return programFXPP4Rules(p4RtBin, nfPrVportP4RuleSets)
}


func programVsiToVsiP4Rules(p4RtBin, mac1, mac2 string) error {
	mac1Vsi, mac1Vport := getVsiVportInfo(mac1)
        mac2Vsi, mac2Vport := getVsiVportInfo(mac2)

        VsiToVsip4RuleSets := []fxpRuleBuilder{
                {
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.vsi_to_vsi_loopback",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,target_vsi=%d,action=linux_networking_control.fwd_to_vsi(%d)",
                                mac1Vsi, mac2Vsi, mac2Vport,
                        ),
                },
                {
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.vsi_to_vsi_loopback",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,target_vsi=%d,action=linux_networking_control.fwd_to_vsi(%d)",
                                mac2Vsi, mac1Vsi, mac1Vport,
                        ),
                },

        }
        return programFXPP4Rules(p4RtBin, VsiToVsip4RuleSets)
}

func deleteVsiToVsiP4Rules(p4RtBin, mac1, mac2 string) error {
        mac1Vsi, _ := getVsiVportInfo(mac1)
        mac2Vsi, _ := getVsiVportInfo(mac2)

        VsiToVsip4RuleSets := []fxpRuleBuilder{
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.vsi_to_vsi_loopback",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,target_vsi=%d",
                                mac1Vsi, mac2Vsi,
                        ),
                },
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.vsi_to_vsi_loopback",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,target_vsi=%d",
                                mac2Vsi, mac1Vsi,
                        ),
                },

        }
        return programFXPP4Rules(p4RtBin, VsiToVsip4RuleSets)
}


func (p *p4rtclient) AddRules(macAddr []byte, vlan int) {
	// For all rules  in RuleSets call
	// P4CP_INSTALL/bin/p4rt-ctl add-entry br0

	ruleSets := p.getAddRuleSets(macAddr, vlan)
	log.WithField("number of rules", len(ruleSets)).Debug("adding FXP rules")

	for _, r := range ruleSets {
		if err := utils.RunP4rtCtlCommand(p.p4RtBin, r...); err != nil {
			log.WithField("error", err).Errorf("error executing add rule command")
		}
	}
	log.Info("FXP rules were added")
}

func (p *p4rtclient) DeleteRules(macAddr []byte, vlan int) {
	// For all rules  in RuleSets call
	// P4CP_INSTALL/bin/p4rt-ctl del-entry br0

	ruleSets := p.getDelRuleSets(macAddr, vlan)
	log.WithField("number of rules", len(ruleSets)).Debug("deleting FXP rules")

	for _, r := range ruleSets {
		if err := utils.RunP4rtCtlCommand(p.p4RtBin, r...); err != nil {
			log.WithField("error", err).Errorf("error executing del rule command")
		}
	}
	log.Info("FXP rules were deleted")
}

func (p *p4rtclient) getAddRuleSets(macAddr []byte, vlan int) []fxpRuleParams {

	macAddrSize := len(macAddr)
	if macAddrSize < 1 || macAddrSize > 6 {
		// We do not have a valid mac address
		log.WithField("mac address", macAddr).Error("Invalid mac address")
		return []fxpRuleParams{}
	}
	vfVsi := int(macAddr[1])

	vfVport := utils.GetVportForVsi(vfVsi)
	portMuxVport := utils.GetVportForVsi(p.portMuxVsi)

	ruleSets := []fxpRuleParams{
		// Rules for control packets coming from overlay VF (vfVsi), IPU will add a VLAN tag (vlan) and send to PortMux Vport (portMuxVport)
		[]string{"add-entry", p.p4br, "linux_networking_control.handle_tx_from_host_to_ovs_and_ovs_to_wire_table", fmt.Sprintf("vmeta.common.vsi=%d,user_meta.cmeta.bit32_zeros=0,action=linux_networking_control.add_vlan_and_send_to_port(%d,%d)", vfVsi, vlan, portMuxVport)},
		[]string{"add-entry", p.p4br, "linux_networking_control.handle_tx_from_host_to_ovs_and_ovs_to_wire_table", fmt.Sprintf("vmeta.common.vsi=%d,user_meta.cmeta.bit32_zeros=0,action=linux_networking_control.add_vlan_and_send_to_port(%d,%d)", vfVsi, vlan, portMuxVport)},
		[]string{"add-entry", p.p4br, "linux_networking_control.handle_rx_loopback_from_host_to_ovs_table", fmt.Sprintf("vmeta.common.vsi=%d,user_meta.cmeta.bit32_zeros=0,action=linux_networking_control.set_dest(%d)", vfVsi, portMuxVport)},
		[]string{"add-entry", p.p4br, "linux_networking_control.vlan_push_mod_table", fmt.Sprintf("vmeta.common.mod_blob_ptr=%d,action=linux_networking_control.vlan_push(1,0,%d)", vlan, vlan)},

		// Rules for control packets coming from vlan port via PortMuxVsi(p.portMuxVsi), IPU will remove the VLAN tag (vlan) and send to overlay VF(vfVport)
		[]string{"add-entry", p.p4br, "linux_networking_control.handle_tx_from_ovs_to_host_table", fmt.Sprintf("vmeta.common.vsi=%d,hdrs.dot1q_tag[vmeta.common.depth].hdr.vid=%d,action=linux_networking_control.remove_vlan_and_send_to_port(%d,%d)", p.portMuxVsi, vlan, vlan, vfVport)},
		[]string{"add-entry", p.p4br, "linux_networking_control.handle_rx_loopback_from_ovs_to_host_table", fmt.Sprintf("vmeta.misc_internal.vm_to_vm_or_port_to_port[27:17]=%d,user_meta.cmeta.bit32_zeros=0,action=linux_networking_control.set_dest(%d)", vfVsi, vfVport)},
		[]string{"add-entry", p.p4br, "linux_networking_control.vlan_pop_mod_table", fmt.Sprintf("vmeta.common.mod_blob_ptr=%d,action=linux_networking_control.vlan_pop", vlan)},
	}
	if p.bridgeType == types.LinuxBridge {
		// Add additional add rules
		macToIntValue := utils.GetMacIntValueFromBytes(macAddr)
		ruleSets = append(ruleSets,
			[]string{"add-entry", p.p4br, "linux_networking_control.l2_fwd_tx_table", fmt.Sprintf("dst_mac=0x%x,user_meta.pmeta.tun_flag1_d0=0x00,action=linux_networking_control.l2_fwd(%d)", macToIntValue, vfVport)},
			[]string{"add-entry", p.p4br, "linux_networking_control.sem_bypass", fmt.Sprintf("dst_mac=0x%x,action=linux_networking_control.set_dest(%d)", macToIntValue, vfVport)},
		)

	}
	return ruleSets
}

func (p *p4rtclient) getDelRuleSets(macAddr []byte, vlan int) []fxpRuleParams {

	macAddrSize := len(macAddr)
	if macAddrSize < 1 || macAddrSize > 6 {
		// We do not have a valid mac address
		log.WithField("mac address", macAddr).Error("Invalid mac address")
		return []fxpRuleParams{}
	}
	vfVsi := int(macAddr[1])

	ruleSets := []fxpRuleParams{
		// Rules for control packets coming from overlay VF (vfVsi), IPU will add a VLAN tag (vlan) and send to PortMux Vport (portMuxVport)
		[]string{"del-entry", p.p4br, "linux_networking_control.handle_tx_from_host_to_ovs_and_ovs_to_wire_table", fmt.Sprintf("vmeta.common.vsi=%d,user_meta.cmeta.bit32_zeros=0", vfVsi)},
		[]string{"del-entry", p.p4br, "linux_networking_control.handle_rx_loopback_from_host_to_ovs_table", fmt.Sprintf("vmeta.common.vsi=%d,user_meta.cmeta.bit32_zeros=0", vfVsi)},
		[]string{"del-entry", p.p4br, "linux_networking_control.vlan_push_mod_table", fmt.Sprintf("vmeta.common.mod_blob_ptr=%d", vlan)},

		// Rules for control packets coming from vlan port via PortMuxVsi(p.portMuxVsi), IPU will remove the VLAN tag (vlan) and send to overlay VF(vfVport)
		[]string{"del-entry", p.p4br, "linux_networking_control.handle_tx_from_ovs_to_host_table", fmt.Sprintf("vmeta.common.vsi=%d,hdrs.dot1q_tag[vmeta.common.depth].hdr.vid=%d", p.portMuxVsi, vlan)},
		[]string{"del-entry", p.p4br, "linux_networking_control.handle_rx_loopback_from_ovs_to_host_table", fmt.Sprintf("vmeta.misc_internal.vm_to_vm_or_port_to_port[27:17]=%d,user_meta.cmeta.bit32_zeros=0", vfVsi)},
		[]string{"del-entry", p.p4br, "linux_networking_control.vlan_pop_mod_table", fmt.Sprintf("vmeta.common.mod_blob_ptr=%d", vlan)},
	}
	if p.bridgeType == types.LinuxBridge {
		// Add additional deletion rules
		macToIntValue := utils.GetMacIntValueFromBytes(macAddr)
		ruleSets = append(ruleSets,
			[]string{"del-entry", p.p4br, "linux_networking_control.l2_fwd_tx_table", fmt.Sprintf("dst_mac=0x%x,user_meta.pmeta.tun_flag1_d0=0x00", macToIntValue)},
			[]string{"del-entry", p.p4br, "linux_networking_control.sem_bypass", fmt.Sprintf("dst_mac=0x%x", macToIntValue)},
		)
	}

	return ruleSets
}

func AddPhyPortRules(p4RtBin string, prP0mac string, prP1mac string) error {
        macAddr, macErr := checkMacAddresses(prP0mac, prP1mac)
        if macErr != nil {
                // We do not have a valid mac address
                log.WithField("mac address", macAddr).Error("Invalid mac address")
                return errors.New("Invalid Mac Address")
        }
        //Add Port 0 P4 rules
	programPhyVportP4Rules(p4RtBin, 0, prP0mac)
	//Add Port 1 P4 rules
        programPhyVportP4Rules(p4RtBin, 1, prP1mac)
	//Add bridge id for non P4 OVS bridge ports
	programPhyVportBridgeId(p4RtBin, 1, 77)

	return nil
}

func DeletePhyPortRules(p4RtBin string, prP0mac string, prP1mac string) error {
        macAddr, macErr := checkMacAddresses(prP0mac, prP1mac)
        if macErr != nil {
                // We do not have a valid mac address
                log.WithField("mac address", macAddr).Error("Invalid mac address")
                return errors.New("Invalid Mac Address")
        }
        //Add Port 0 P4 rules
        deletePhyVportP4Rules(p4RtBin, 0, prP0mac)
        //Add Port 1 P4 rules
        deletePhyVportP4Rules(p4RtBin, 1, prP1mac)
        //Add bridge id for non P4 OVS bridge ports
        deletePhyVportBridgeId(p4RtBin, 1, 77)

        return nil

}

func AddHostVfP4Rules(p4RtBin string, hostVfMac []byte, accMac string) error {
	hostMacAddr := net.HardwareAddr(hostVfMac)
	vfMac, hostMacErr := checkMacAddresses(hostMacAddr.String())
	if hostMacErr != nil {
                // We do not have a valid mac address for the host or apf interface
                log.WithField("mac address", vfMac).Error("Invalid mac address")
                return errors.New("Invalid Mac Address")
        }

	accAddr, apfMacErr := checkMacAddresses(accMac)
	if apfMacErr != nil {
                // We do not have a valid mac address for the host or apf interface
                log.WithField("mac address", accAddr).Error("Invalid mac address")
                return errors.New("Invalid Mac Address")
	}

	hostVfVsi, hostVfVport := getVsiVportInfo(hostMacAddr.String())
	apfPrVsi, apfPrVport := getVsiVportInfo(accMac)

        hostVfP4ruleSets := []fxpRuleBuilder{
		{
			Action:  "add-entry",
			P4br:    "br0",
			Control: "linux_networking_control.tx_source_port",
			Metadata: fmt.Sprintf(
				"vmeta.common.vsi=%d/0x7ff,priority=1,action=linux_networking_control.set_source_port(%d)",
				hostVfVsi, hostVfVport,
			),
		},
		{
			Action:  "add-entry",
			P4br:    "br0",
			Control: "linux_networking_control.tx_acc_vsi",
			Metadata: fmt.Sprintf(
				"vmeta.common.vsi=%d,zero_padding=0,action=linux_networking_control.l2_fwd_and_bypass_bridge(%d)",
				apfPrVsi, apfPrVport,
			),
		},
		{
			Action:  "add-entry",
			P4br:    "br0",
			Control: "linux_networking_control.vsi_to_vsi_loopback",
			Metadata: fmt.Sprintf(
				"vmeta.common.vsi=%d,target_vsi=%d,action=linux_networking_control.fwd_to_vsi(%d)",
				apfPrVsi, hostVfVsi, hostVfVport,
			),
		},
		{
			Action:  "add-entry",
			P4br:    "br0",
			Control: "linux_networking_control.vsi_to_vsi_loopback",
			Metadata: fmt.Sprintf(
				"vmeta.common.vsi=%d,target_vsi=%d,action=linux_networking_control.fwd_to_vsi(%d)",
				hostVfVsi, apfPrVsi, apfPrVport,
			),
		},
		{
			Action:  "add-entry",
			P4br:    "br0",
			Control: "linux_networking_control.source_port_to_pr_map",
			Metadata: fmt.Sprintf(
				"user_meta.cmeta.source_port=%d,zero_padding=0,action=linux_networking_control.fwd_to_vsi(%d)",
				hostVfVport, apfPrVport,
			),
		},
	}

        log.WithField("number of rules", len(hostVfP4ruleSets)).Debug("adding FXP rules")

	err := programFXPP4Rules(p4RtBin, hostVfP4ruleSets)
	if err != nil {
             log.Info("Host VF FXP P4 rules add failed")
	     return err
	} else {
             log.Info("Host VF FXP P4 rules were added successfully")
	}
	return nil
}

func DeleteHostVfP4Rules(p4RtBin string, hostVfMac []byte, accMac string) error {
        hostMacAddr := net.HardwareAddr(hostVfMac)
        vfMac, hostMacErr := checkMacAddresses(hostMacAddr.String())
        if hostMacErr != nil {
                // We do not have a valid mac address for the host or apf interface
                log.WithField("mac address", vfMac).Error("Invalid mac address")
                return errors.New("Invalid Mac Address")
        }

        accMacAddr, apfMacErr := checkMacAddresses(accMac)
        if apfMacErr != nil {
                // We do not have a valid mac address for the host or apf interface
                log.WithField("mac address", accMacAddr).Error("Invalid mac address")
                return errors.New("Invalid Mac Address")
        }

        hostVfVsi, hostVfVport := getVsiVportInfo(hostMacAddr.String())
        apfPrVsi, _ := getVsiVportInfo(accMac)

        hostVfP4ruleSets := []fxpRuleBuilder{
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.tx_source_port",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d/0x7ff,priority=1",
                                hostVfVsi,
                        ),
                },
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.tx_acc_vsi",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,zero_padding=0",
                                apfPrVsi,
                        ),
                },
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.vsi_to_vsi_loopback",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,target_vsi=%d",
                                apfPrVsi,
                        ),
                },
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.vsi_to_vsi_loopback",
                        Metadata: fmt.Sprintf(
                                "vmeta.common.vsi=%d,target_vsi=%d",
                                hostVfVsi, apfPrVsi,
                        ),
                },
                {
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.source_port_to_pr_map",
                        Metadata: fmt.Sprintf(
                                "user_meta.cmeta.source_port=%d,zero_padding=0",
                                hostVfVport,
                        ),
                },
        }

        log.WithField("number of rules", len(hostVfP4ruleSets)).Debug("adding FXP rules")

        err := programFXPP4Rules(p4RtBin, hostVfP4ruleSets)
        if err != nil {
             log.Info("Host VF FXP P4 rules delete failed")
	     return err
        } else {
             log.Info("Host VF FXP P4 rules were deleted successfully")
        }
	return nil
}

func AddNFP4Rules(p4RtBin string, vfMacList []string, ingressMac string, egressMac string, ingressPRMac string, egressPRMac string) error {
	for _, vfMac := range vfMacList {
             macAddr, macErr := checkMacAddresses(vfMac)
             if macErr != nil {
                   // We do not have a valid mac address for the host or apf interface
                   log.WithField("mac address", macAddr).Error("Invalid mac address in the VF Mac list")
                   return errors.New("Invalid Mac Address")
             }
	}

	macAddr, macErr := checkMacAddresses(ingressMac, egressMac, ingressPRMac, egressPRMac)
        if macErr != nil {
                // We do not have a valid mac address for the host or apf interface
                log.WithField("mac address", macAddr).Error("Invalid mac address in the VF Mac list")
                return errors.New("Invalid Mac Address")
        }

	err := programNfPrVportP4Rules(p4RtBin, ingressMac, ingressPRMac)
        if err != nil {
             log.Info("Add NF FXP P4 rules add failed for %s, %s", ingressMac, ingressPRMac)
	     return err
        } else {
             log.Info("Add NF FXP P4 rules were added successfully for %s, %s", ingressMac, ingressPRMac)
        }

        err = programNfPrVportP4Rules(p4RtBin, egressMac, egressPRMac)
        if err != nil {
             log.Info("Add NF FXP P4 rules add failed for %s, %s", egressMac, egressPRMac)
	     return err
        } else {
             log.Info("Add NF FXP P4 rules were added successfully for %s, %s", egressMac, egressPRMac)
        }

        for vfIdx := range vfMacList {
                vfMacAddr, err := utils.GetMacAsByteArray(vfMacList[vfIdx])
                if err != nil {
                     fmt.Printf("unable to extract octets from %s: %v", vfMacList[vfIdx], err)
                     return err
                }
                nfMacList := []string {ingressMac, egressMac}
                for nfIdx := range nfMacList {
		        nfMacAddr, err := utils.GetMacAsByteArray(nfMacList[vfIdx])
                        if err != nil {
                             fmt.Printf("unable to extract octets from %s: %v", vfMacList[nfIdx], err)
                             return err
		        }
			programVsiToVsiP4Rules(p4RtBin, string(vfMacAddr), string(nfMacAddr))
                }
         }
	 return nil
}

func DeleteNFP4Rules(p4RtBin string, vfMacList []string, ingressMac string, egressMac string, ingressPRMac string, egressPRMac string) error {
        for _, vfMac := range vfMacList {
	     macAddr, macErr := checkMacAddresses(vfMac)
             if macErr != nil {
                   // We do not have a valid mac address for the host or apf interface
                   log.WithField("mac address", macAddr).Error("Invalid mac address in the VF Mac list")
                   return errors.New("Invalid Mac Address")
             }
	}

	macAddr, macErr := checkMacAddresses(ingressMac, egressMac, ingressPRMac, egressPRMac)
        if macErr != nil {
                // We do not have a valid mac address for the host or apf interface
                log.WithField("mac address", macAddr).Error("Invalid mac address in the VF Mac list")
                return errors.New("Invalid Mac Address")
        }

        err := deleteNfPrVportP4Rules(p4RtBin, ingressMac, ingressPRMac)
        if err != nil {
             log.Info("Delete NF FXP P4 rules add failed for %s, %s", ingressMac, ingressPRMac)
	     return err
        } else {
             log.Info("Delete NF FXP P4 rules were added successfully for %s, %s", ingressMac, ingressPRMac)
        }

        err = deleteNfPrVportP4Rules(p4RtBin, egressMac, egressPRMac)
        if err != nil {
             log.Info("Delete NF FXP P4 rules add failed for %s, %s", egressMac, egressPRMac)
	     return err
        } else {
             log.Info("Delete NF FXP P4 rules were added successfully for %s, %s", egressMac, egressPRMac)
        }

        for vfIdx := range vfMacList {
                vfMacAddr, err := utils.GetMacAsByteArray(vfMacList[vfIdx])
                if err != nil {
                     fmt.Printf("unable to extract octets from %s: %v", vfMacList[vfIdx], err)
                     return err
                }
                nfMacList := []string {ingressMac, egressMac}
                for nfIdx := range nfMacList {
                        nfMacAddr, err := utils.GetMacAsByteArray(nfMacList[vfIdx])
                        if err != nil {
                             fmt.Printf("unable to extract octets from %s: %v", vfMacList[nfIdx], err)
                             return err
                        }
                        deleteVsiToVsiP4Rules(p4RtBin, string(vfMacAddr), string(nfMacAddr))
                }
         }
return nil
}

func AddPeerToPeerP4Rules(p4RtBin string, vfMacList []string) error {
        for _, vfMac := range vfMacList {
            macAddr, macErr := checkMacAddresses(vfMac)
            if macErr != nil {
                  // We do not have a valid mac address for the host VF interface
                  log.WithField("mac address", macAddr).Error("Invalid mac address in the VF Mac list")
                  return errors.New("Invalid Mac Address")
            }
	}
        for i := range vfMacList {
                for j := range vfMacList {
                        if i != j {
                                srcVfMac, err := utils.GetMacAsByteArray(vfMacList[i])
                                if err != nil {
                                        fmt.Printf("unable to extract octets from %s: %v", vfMacList[i], err)
                                        return err
                                }

                                dstVfMac, err := utils.GetMacAsByteArray(vfMacList[j])
                                if err != nil {
                                        fmt.Printf("unable to extract octets from %s: %v", vfMacList[j], err)
                                        return err
                                }
				programVsiToVsiP4Rules(p4RtBin, string(srcVfMac), string(dstVfMac))
                        }
                }
        }
return nil
}

func DeletePeerToPeerP4Rules(p4RtBin string, vfMacList []string) error {
        for _, vfMac := range vfMacList {
            macAddr, macErr := checkMacAddresses(vfMac)
            if macErr != nil {
                  // We do not have a valid mac address for the host VF interface
                  log.WithField("mac address", macAddr).Error("Invalid mac address in the VF Mac list")
                  return errors.New("Invalid Mac Address")
            }
        }
        for i := range vfMacList {
                for j := range vfMacList {
                        if i != j {
                                srcVfMac, err := utils.GetMacAsByteArray(vfMacList[i])
                                if err != nil {
                                        fmt.Printf("unable to extract octets from %s: %v", vfMacList[i], err)
                                        return err
                                }

                                dstVfMac, err := utils.GetMacAsByteArray(vfMacList[j])
                                if err != nil {
                                        fmt.Printf("unable to extract octets from %s: %v", vfMacList[j], err)
                                        return err
                                }
                                deleteVsiToVsiP4Rules(p4RtBin, string(srcVfMac), string(dstVfMac))
                        }
                }
        }
return nil
}

func AddLAGP4Rules(p4RtBin string) error {
        var LAGP4ruleSets []fxpRuleBuilder

        LAGP4ruleSets = append(LAGP4ruleSets,
                     fxpRuleBuilder{
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.ipv4_lpm_root_lut",
                        Metadata:"user_meta.cmeta.bit16_zeros=4/65535,priority=2048,action=linux_networking_control.ipv4_lpm_root_lut_action(0)",
                     })
         err := programFXPP4Rules(p4RtBin, LAGP4ruleSets)
         if err != nil {
              log.Info("LAG LPM ROOT LUT FXP P4 rules add failed")
         } else {
              log.Info("LAG LPM ROOT LUT FXP P4 rules were added successfully")
         }

        log.Info("ARUN: %v", LAGP4ruleSets)
	LAGP4ruleSets = []fxpRuleBuilder{}
	log.Info("ARUN: %v", LAGP4ruleSets)
	for idx := 0; idx < 8; idx++ {
              LAGP4ruleSets = append(LAGP4ruleSets,
                     fxpRuleBuilder{
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.tx_lag_table",
                        Metadata: fmt.Sprintf("user_meta.cmeta.lag_group_id=0/255,hash=%d/7,priority=1,action=linux_networking_control.bypass", idx),
                     },
	      )
	 }
	 log.Info("ARUN: %v", LAGP4ruleSets)
	 err = programFXPP4Rules(p4RtBin, LAGP4ruleSets)
         if err != nil {
              log.Info("Host VF FXP P4 rules add failed")
              return err
         } else {
             log.Info("Host VF FXP P4 rules were added successfully")
         }
         return nil
}

func DeleteLAGP4Rules(p4RtBin string) error {
	var LAGP4ruleSets []fxpRuleBuilder

        LAGP4ruleSets = append(LAGP4ruleSets,
                     fxpRuleBuilder{
                        Action:  "add-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.ipv4_lpm_root_lut",
                        Metadata:"user_meta.cmeta.bit16_zeros=4/65535,priority=2048",
                     })
        log.Info("ARUN: %v", LAGP4ruleSets)
        LAGP4ruleSets = []fxpRuleBuilder{}
        log.Info("ARUN: %v", LAGP4ruleSets)
        for idx := 0; idx < 8; idx++ {
              LAGP4ruleSets = append(LAGP4ruleSets,
                     fxpRuleBuilder{
                        Action:  "del-entry",
                        P4br:    "br0",
                        Control: "linux_networking_control.tx_lag_table",
                        Metadata: fmt.Sprintf("user_meta.cmeta.lag_group_id=0/255,hash=%d/7,priority=1", idx),
                     },
              )
         }
	 log.Info("ARUN: %v", LAGP4ruleSets)
         err := programFXPP4Rules(p4RtBin, LAGP4ruleSets)
         if err != nil {
              log.Info("LAG FXP P4 rules delete failed")
              return err
         } else {
             log.Info("LAG FXP P4 rules were delete successfully")
         }
         return nil
}

