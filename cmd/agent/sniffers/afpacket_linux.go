// +build linux
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
	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"time"
)

type AfpacketHandle struct {
	TPacket *afpacket.TPacket
}

func NewAfpacketHandle(device string, snaplen int, block_size int, num_blocks int,
	timeout time.Duration) (*AfpacketHandle, error) {

	h := &AfpacketHandle{}
	var err error

	if device == "any" {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptFrameSize(snaplen),
			afpacket.OptBlockSize(block_size),
			afpacket.OptNumBlocks(num_blocks),
			afpacket.OptPollTimeout(timeout))
	} else {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptInterface(device),
			afpacket.OptFrameSize(snaplen),
			afpacket.OptBlockSize(block_size),
			afpacket.OptNumBlocks(num_blocks),
			afpacket.OptPollTimeout(timeout))
	}

	return h, err
}

func (h *AfpacketHandle) SetBPFFilter(expr string) (_ error) {
	return h.TPacket.SetBPFFilter(expr)
}

func (h *AfpacketHandle) Close() {
	h.TPacket.Close()
}

func (h *AfpacketHandle) GetPacketSource() *gopacket.PacketSource {
	return gopacket.NewPacketSource(TPacket, layers.LinkTypeEthernet)
}
