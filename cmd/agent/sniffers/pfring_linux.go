// +build linux,havepfring
/*
* Copyright (c) 2017 Couchbase, Inc.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*    http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*/

package sniffers

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pfring"
)

type PfringHandle struct {
	Ring *pfring.Ring
}

func NewPfringHandle(device string, snaplen int, promisc bool) (*PfringHandle, error) {
	var h PfringHandle
	var err error

	if device == "any" {
		return nil, fmt.Errorf("Pfring sniffing doesn't support 'any' as interface")
	}

	var flags pfring.Flag

	if promisc {
		flags = pfring.FlagPromisc
	}

	h.Ring, err = pfring.NewRing(device, uint32(snaplen), flags)

	return &h, err
}

func (h *PfringHandle) SetBPFFilter(expr string) (_ error) {
	return h.Ring.SetBPFFilter(expr)
}

func (h *PfringHandle) Enable() (_ error) {
	return h.Ring.Enable()
}

func (h *PfringHandle) GetPacketSource() *gopacket.PacketSource {
	return gopacket.NewPacketSource(Ring, layers.LinkTypeEthernet)
}

func (h *PfringHandle) Close() {
	h.Ring.Close()
}
