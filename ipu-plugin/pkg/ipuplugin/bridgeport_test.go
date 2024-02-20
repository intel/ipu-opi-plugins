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

package ipuplugin

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pb "github.com/opiproject/opi-api/network/evpn-gw/v1alpha1/gen/go"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

var _ = Describe("bridgeport", Serial, func() {
	Describe("createVlanInterface", Serial, func() {
		Context("when vlan interface is found but not a valid vlan interface type", func() {
			It("should return error", func() {
				linkByNameFn = fakeLinkByName // it returns a dummy netlink instance; not a valid vlan interface
				dummyMasterLink := &netlink.Dummy{}
				err := createVlanInterface(dummyMasterLink, "dummyVlanIntf", 100)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when vlan interface is found and is a valid vlan interface type", func() {
			It("should create a new vlan interface and not return any error", func() {
				linkByNameFn = func(ifName string) (netlink.Link, error) {
					vLink := &netlink.Vlan{}
					vLink.Name = ifName
					return vLink, nil
				}
				linkAddFn = fakeLinkAdd
				dummyMasterLink := &netlink.Dummy{}
				err := createVlanInterface(dummyMasterLink, "dummyVlanIntf", 100)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("when vlan interface is not found", func() {
			It("should create a new vlan interface and not return any error", func() {
				linkByNameFn = fakeLinkByNameWithErr
				linkAddFn = fakeLinkAdd
				dummyMasterLink := &netlink.Dummy{}
				err := createVlanInterface(dummyMasterLink, "dummyVlanIntf", 100)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("when vlan interface is found but error occurred when adding it via netlink", func() {
			It("should create a new vlan interface and not return any error", func() {
				linkByNameFn = fakeLinkByNameWithErr
				linkAddFn = fakeLinkAddWithErr
				dummyMasterLink := &netlink.Dummy{}
				err := createVlanInterface(dummyMasterLink, "dummyVlanIntf", 100)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error creating vlan interface"))
			})
		})
	})

	Describe("CreateBridgePort", Serial, func() {
		var ipuServer *server
		BeforeEach(func() {
			fakeBrCtlr := &mockBrCtlr{}
			fakeP4rtClient := &mockP4rtClient{}
			ipuServer = &server{
				bridgeCtlr: fakeBrCtlr,
				p4RtClient: fakeP4rtClient,
				log:        log.WithField("pkg", "bridgeport_test.go"),
			}
		})
		Context("when mac address in CreateBridgePortRequest is not valid", func() {
			It("should return error", func() {
				fakeReq := &pb.CreateBridgePortRequest{
					BridgePort: &pb.BridgePort{
						Spec: &pb.BridgePortSpec{}, // []MacAddress is not initialized
					},
				}
				_, err := ipuServer.CreateBridgePort(context.TODO(), fakeReq)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when no vlan id in CreateBridgePortRequest", func() {
			It("should return error", func() {
				fakeMacAddr := []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
				fakeReq := &pb.CreateBridgePortRequest{
					BridgePort: &pb.BridgePort{
						Spec: &pb.BridgePortSpec{
							MacAddress: fakeMacAddr,
						},
					},
				}
				_, err := ipuServer.CreateBridgePort(context.TODO(), fakeReq)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when no vlan id is not in valid vlan range", func() {
			It("should return error", func() {
				fakeMacAddr := []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
				fakeReq := &pb.CreateBridgePortRequest{
					BridgePort: &pb.BridgePort{
						Spec: &pb.BridgePortSpec{
							MacAddress:     fakeMacAddr,
							LogicalBridges: []string{"0"},
						},
					},
				}
				_, err := ipuServer.CreateBridgePort(context.TODO(), fakeReq)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when no VF VSI is 0", func() {
			It("should return error", func() {
				fakeMacAddr := []byte{0xaa, 0x00, 0xcc, 0xdd, 0xee, 0xff} // the second octet here is the VSI number
				fakeReq := &pb.CreateBridgePortRequest{
					BridgePort: &pb.BridgePort{
						Spec: &pb.BridgePortSpec{
							MacAddress:     fakeMacAddr,
							LogicalBridges: []string{"100"},
						},
					},
				}
				_, err := ipuServer.CreateBridgePort(context.TODO(), fakeReq)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when port is present in internal Port Map", func() {
			It("should return without any error", func() {

				fakeMacAddr := []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
				fakePort := &pb.BridgePort{
					Name: "fakePort",
					Spec: &pb.BridgePortSpec{
						MacAddress:     fakeMacAddr,
						LogicalBridges: []string{"100"},
					},
				}
				fakeReq := &pb.CreateBridgePortRequest{BridgePort: fakePort}

				ipuServer.Ports = make(map[string]*pb.BridgePort)
				ipuServer.Ports["fakePort"] = fakePort // fakePort already exists in internal Map
				_, err := ipuServer.CreateBridgePort(context.TODO(), fakeReq)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when CreateBridgePortRequest is valid and port is present in internal Port Map", func() {
			It("should return without any error", func() {
				ipuServer.Ports = make(map[string]*pb.BridgePort)
				fakeMacAddr := []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
				fakePort := &pb.BridgePort{
					Name: "fakePort",
					Spec: &pb.BridgePortSpec{
						MacAddress:     fakeMacAddr,
						LogicalBridges: []string{"100"},
					},
				}

				// mock other internal functions and other dependencies
				linkByNameFn = func(ifName string) (netlink.Link, error) {
					vLink := &netlink.Vlan{}
					vLink.Name = ifName
					return vLink, nil
				}
				linkAddFn = fakeLinkAdd
				ipuServer.uplinkInterface = "dummyUplink"

				fakeReq := &pb.CreateBridgePortRequest{BridgePort: fakePort}
				_, err := ipuServer.CreateBridgePort(context.TODO(), fakeReq)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
