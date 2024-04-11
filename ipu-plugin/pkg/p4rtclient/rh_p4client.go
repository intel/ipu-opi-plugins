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
	log "github.com/sirupsen/logrus"
)

const (
	stag = 0 // Using single flat L2 network with vlan 0
)

type rhP4Client struct {
	p4RtBin    string
	portMuxVsi int
	p4br       string
	bridgeType types.BridgeType
}

func NewRHP4Client(p4RtBin string, portMuxVsi int, p4BridgeName string, brType types.BridgeType) types.P4RTClient {
	log.Debug("Creating Redhat P4Client instance")
	return &rhP4Client{
		p4RtBin:    p4RtBin,
		portMuxVsi: portMuxVsi,
		p4br:       p4BridgeName,
		bridgeType: brType,
	}
}

func (p *rhP4Client) AddRules(macAddr []byte, vlan int) {
	// For all rules  in RuleSets call
	// P4CP_INSTALL/bin/p4rt-ctl add-entry br0

	ruleSets := p.getAddRuleSets(macAddr, vlan)
	log.WithField("number of rules", len(ruleSets)).Debug("adding FXP rules")

	for _, r := range ruleSets {
		if err := runP4rtCtlCommand(p.p4RtBin, r...); err != nil {
			log.WithField("error", err).Errorf("error executing add rule command")
		}
	}
	log.Info("FXP rules were added")
}

func (p *rhP4Client) DeleteRules(macAddr []byte, vlan int) {
	// For all rules  in RuleSets call
	// P4CP_INSTALL/bin/p4rt-ctl del-entry br0

	ruleSets := p.getDelRuleSets(macAddr, vlan)
	log.WithField("number of rules", len(ruleSets)).Debug("deleting FXP rules")

	for _, r := range ruleSets {
		if err := runP4rtCtlCommand(p.p4RtBin, r...); err != nil {
			log.WithField("error", err).Errorf("error executing del rule command")
		}
	}
	log.Info("FXP rules were delete")
}

func (p *rhP4Client) getAddRuleSets(macAddr []byte, vlan int) []fxpRuleParams {

	macAddrSize := len(macAddr)
	if macAddrSize < 1 || macAddrSize > 6 {
		// We do not have a valid mac address
		log.WithField("mac address", macAddr).Error("Invalid mac address")
		return []fxpRuleParams{}
	}
	vfVsi := int(macAddr[1])

	// vfVport := getVportForVsi(vfVsi)
	// portMuxVport := getVportForVsi(p.portMuxVsi)

	ruleSets := []fxpRuleParams{
		// $P4CP_INSTALL/bin/p4rt-ctl add-entry br0 rh_mvp_control.vlan_push_ctag_stag_mod_table "meta.common.mod_blob_ptr=2,action=rh_mvp_control.mod_vlan_push_ctag_stag(1,1,301,1,1,300)"
		// $P4CP_INSTALL/bin/p4rt-ctl add-entry br0 rh_mvp_control.portmux_egress_req_table "vsi=0xe,vid=301,action=rh_mvp_control.vlan_pop_ctag_stag(5,37)"
		[]string{"add-entry", p.p4br, "rh_mvp_control.vlan_push_ctag_stag_mod_table", fmt.Sprintf("meta.common.mod_blob_ptr=%d,action=rh_mvp_control.mod_vlan_push_ctag_stag(1,1,%d,1,1,%d)", vfVsi, vlan, stag)},
		[]string{"add-entry", p.p4br, "rh_mvp_control.portmux_egress_req_table", fmt.Sprintf("vsi=%d,vid=%d,action=rh_mvp_control.vlan_pop_ctag_stag(5,37)", p.portMuxVsi, vlan)},
	}

	return ruleSets
}

func (p *rhP4Client) getDelRuleSets(macAddr []byte, vlan int) []fxpRuleParams {

	macAddrSize := len(macAddr)
	if macAddrSize < 1 || macAddrSize > 6 {
		// We do not have a valid mac address
		log.WithField("mac address", macAddr).Error("Invalid mac address")
		return []fxpRuleParams{}
	}
	vfVsi := int(macAddr[1])

	ruleSets := []fxpRuleParams{
		[]string{"del-entry", p.p4br, "rh_mvp_control.vlan_push_ctag_stag_mod_table", fmt.Sprintf("meta.common.mod_blob_ptr=%d", vfVsi)},
		[]string{"del-entry", p.p4br, "rh_mvp_control.portmux_egress_req_table", fmt.Sprintf("vsi=%d,vid=%d", p.portMuxVsi, vlan)},
	}

	return ruleSets
}
