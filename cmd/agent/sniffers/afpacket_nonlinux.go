// +build !linux
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
	"time"
)

type AfpacketHandle struct {
}

func NewAfpacketHandle(device string, snaplen int, blockSize int, numBlocks int,
	timeout time.Duration) (*AfpacketHandle, error) {
	return nil, fmt.Errorf("Afpacket sniffing is only available on Linux")
}

func (h *AfpacketHandle) SetBPFFilter(expr string) (_ error) {
	return fmt.Errorf("Afpacket  sniffing is only available on Linux")
}

func (h *AfpacketHandle) GetPacketSource() *gopacket.PacketSource {
	return nil
}

func (h *AfpacketHandle) Close() {
}
