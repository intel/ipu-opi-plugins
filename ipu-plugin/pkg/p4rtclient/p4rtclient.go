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
