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

package main

import (
	"../../logger"
	pb "../../rpc"
	"./sniffers"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"golang.org/x/net/context"
	"io"
	"os"
	"strconv"
	"sync"
)

type Agent struct {
	mutex         *sync.Mutex
	packetSource  *gopacket.PacketSource
	config        *Config
	isHandleAlive bool
	filter        string
	streams       map[uint64]*Stream
	logger        *logger.Logger
}

func afpacketComputeSize(targetSizeMb int, snaplen int, pageSize int) (
	frameSize int, blockSize int, numBlocks int, err error) {

	if snaplen < pageSize {
		frameSize = pageSize / (pageSize / snaplen)
	} else {
		frameSize = (snaplen/pageSize + 1) * pageSize
	}
	// 128 is the default from the gopacket library so just use that
	blockSize = frameSize * 128
	numBlocks = (targetSizeMb * 1024 * 1024) / blockSize

	if numBlocks == 0 {
		return 0, 0, 0, fmt.Errorf("Buffer size too small")
	}
	return frameSize, blockSize, numBlocks, nil
}

func (agent *Agent) Initialize() {
	snaplen := 1600
	filter := fmt.Sprint("tcp and port ", agent.config.InterfaceConfig.Port)
	if agent.config.InterfaceConfig.CaptureType == AF_PACKET {
		var afpacketHandle *sniffers.AfpacketHandle
		_, blockSize, numBlocks, err := afpacketComputeSize(agent.config.InterfaceConfig.AfPacketTragetSizeInMB,
			snaplen, os.Getpagesize())
		afpacketHandle, err = sniffers.NewAfpacketHandle(agent.config.InterfaceConfig.Device, snaplen, blockSize, numBlocks, 30)
		if err != nil {
			agent.logger.Error("%v", err)
			os.Exit(1)
		}
		if err = afpacketHandle.SetBPFFilter(filter); err != nil {
			agent.logger.Error("%v", err)
			os.Exit(1)
		} else {
			agent.packetSource = afpacketHandle.GetPacketSource()
		}
	} else if agent.config.InterfaceConfig.CaptureType == PF_RING {
		var pfringHandle *sniffers.PfringHandle
		pfringHandle, err := sniffers.NewPfringHandle(agent.config.InterfaceConfig.Device, snaplen, true)
		if err != nil {
			agent.logger.Error("%v", err)
			os.Exit(1)
		}
		pfringHandle.Enable()
		if err := pfringHandle.SetBPFFilter(filter); err != nil {
			agent.logger.Error("%v", err)
			os.Exit(1)
		} else {
			agent.packetSource = pfringHandle.GetPacketSource()
		}
	} else {
		var handle *pcap.Handle
		handle, err := pcap.OpenLive(agent.config.InterfaceConfig.Device, int32(snaplen), true, pcap.BlockForever)
		if err != nil {
			agent.logger.Error("%v", err)
			os.Exit(1)
		}
		if err := handle.SetBPFFilter(filter); err != nil {
			agent.logger.Error("%v", err)
			os.Exit(1)
		} else {
			agent.packetSource = gopacket.NewPacketSource(handle, handle.LinkType())
		}
	}

	agent.packetSource.DecodeOptions.NoCopy = true
}

func (agent *Agent) handlePacket(packet gopacket.Packet) {
	transport := packet.TransportLayer()
	streamKey := transport.TransportFlow().FastHash()
	if agent.streams[streamKey] == nil {
		agent.streams[streamKey] = &Stream{
			currentRequests:  make(map[uint32]*Command),
			currentResponses: make(map[uint32]*Command),
			src:              transport.TransportFlow().Src().String(),
			dst:              transport.TransportFlow().Dst().String(),
			mutex:            &sync.Mutex{},
		}
	}
	agent.streams[streamKey].HandlePacket(transport.LayerPayload())
}

func (agent *Agent) startCapture() {
	agent.mutex.Lock() //only one capture can proceed at any point
	agent.isHandleAlive = true
	agent.streams = make(map[uint64]*Stream)

	for agent.isHandleAlive {
		packet, err := agent.packetSource.NextPacket()

		if err == io.EOF {
			agent.isHandleAlive = false
			agent.logger.Info("Handle is no longer alive")
			break
		}
		agent.handlePacket(packet)
	}
	agent.mutex.Unlock()
}

func (agent *Agent) GetResults() map[string]*pb.AgentResultsResponse_CaptureInfo {
	responseStats := make(map[string]*pb.AgentResultsResponse_CaptureInfo)

	for streamkey, stream := range agent.streams {
		for _, row := range stream.latencyInfo {
			responseStats[strconv.Itoa(int(row.Opaque))+strconv.FormatUint(streamkey, 10)] = &pb.AgentResultsResponse_CaptureInfo{
				Opaque:    strconv.Itoa(int(row.Opaque)),
				Oplatency: fmt.Sprintf("%v", row.Latency/1000),
				Key:       row.Key,
			}
		}
	}
	return responseStats
}

func (agent *Agent) CaptureSignal(context.Context, *pb.CoordinatorCaptureRequest) (*pb.AgentCaptureResponse, error) {
	go agent.startCapture()
	return &pb.AgentCaptureResponse{Status: "success"}, nil
}

func (agent *Agent) stopCapture() {
	agent.isHandleAlive = false
}

func (agent *Agent) shutdown() {
	agent.stopCapture()
	agent.mutex.Lock()
	agent.streams = nil
	agent.mutex.Unlock()
}

func (agent *Agent) GoodByeSignal(context.Context, *pb.CoordinatorGoodByeRequest) (*pb.AgentGoodByeResponse, error) {
	agent.shutdown()
	return &pb.AgentGoodByeResponse{Status: "success"}, nil
}

func (agent *Agent) AgentResults(context.Context, *pb.CoordinatorResultsRequest) (*pb.AgentResultsResponse, error) {
	agent.stopCapture()
	captureMap := agent.GetResults()
	return &pb.AgentResultsResponse{
		Status:     "success",
		CaptureMap: captureMap,
	}, nil
}
