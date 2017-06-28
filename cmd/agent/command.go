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
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"time"
)

type Command struct {
	state              ParserState
	commandType        CommandType
	opcode             string
	magic              uint8
	opaque             uint32
	keyLength          uint16
	extrasLength       uint8
	valueLength        uint32
	cas                uint32
	key                []byte
	partial            []byte
	captureTimeInNanos int64
}

type ParserState int

const (
	parseStateHeader ParserState = iota
	parseStateExtras
	parseStateKey
	parseStateValue
	parseStateComplete
)

type CommandType int

const (
	REQUEST CommandType = iota
	RESPONSE
)

type Opcode string

const (
	GET     = "0"
	SET     = "1"
	IGNORED = "IGNORED"
)

func NewCommand() *Command {
	return &Command{
		state:              parseStateHeader,
		captureTimeInNanos: time.Now().UnixNano(),
	}
}

func (c *Command) ReadNewPacketData(data *bytes.Buffer) error {
	if c.state == parseStateHeader {
		partialLen := len(c.partial)
		if data.Len() < 24 && partialLen == 0 {
			data.Read(c.partial)
			return io.EOF
		} else if data.Len() < 24 && partialLen > 0 {
			if b, err := data.ReadByte(); err != nil {
				if err == io.EOF {
					return err
				} else {
					log.Fatalf("Failed parsing packet at partial %v", c.state, err)
				}
			} else {
				c.partial = append(c.partial, b)
			}
			return io.EOF
		}

		var header bytes.Buffer

		if len(c.partial) > 0 {
			if n, err := header.Write(c.partial); err != nil {
				log.Fatalf("Failed parsing packet at partial %v", c.state, err)
			} else {
				header.Write(data.Next(24 - n))
			}
		} else {
			header.Write(data.Next(24))
		}

		if magic, err := header.ReadByte(); err != nil {
			log.Fatal("Failed parsing packet at magic %v", c.state, err)
		} else {
			c.magic = magic
			if c.magic == 0x80 {
				c.commandType = REQUEST
			} else if c.magic == 0x81 {
				c.commandType = RESPONSE
			}
		}

		if opcode, err := header.ReadByte(); err != nil {
			log.Fatal("Failed parsing packet opcode %v", err)
		} else {
			if opcode == 0x0 {
				c.opcode = GET
			} else if opcode == 0x1 {
				c.opcode = SET
			} else {
				c.opcode = IGNORED
			}

		}

		keyLenBytes := header.Next(2)
		c.keyLength = binary.BigEndian.Uint16(keyLenBytes)

		extrasLenBytes, _ := header.ReadByte()
		c.extrasLength = extrasLenBytes

		header.Next(1) //datatype
		header.Next(2) //vbucket or status

		totalBodyLength := binary.BigEndian.Uint32(header.Next(4))
		c.valueLength = totalBodyLength - uint32(c.keyLength) - uint32(c.extrasLength)

		opaqueBytes := header.Next(4)
		c.opaque = binary.BigEndian.Uint32(opaqueBytes)
		header.Next(2) //cas

		if data.Len() > 0 && c.extrasLength > 0 {
			c.state = parseStateExtras
		} else if data.Len() > 0 && c.extrasLength == 0 && c.keyLength > 0 {
			c.state = parseStateKey
		} else if data.Len() > 0 && c.extrasLength == 0 && c.valueLength > 0 {
			c.state = parseStateValue
		} else {
			c.state = parseStateComplete
		}
		c.partial = nil

	}

	if c.state == parseStateExtras {
		extrasLen := int(c.extrasLength)

		if data.Len() >= extrasLen {
			data.Next(extrasLen)
			if c.keyLength > 0 {
				c.state = parseStateKey
			} else if c.valueLength > 0 {
				c.state = parseStateValue
			} else {
				c.state = parseStateComplete
			}
		} else {
			available := data.Len()
			data.Next(available)
			c.extrasLength -= uint8(available)
			return io.EOF
		}
	}

	if c.state == parseStateKey {
		keyLen := int(c.keyLength)

		if data.Len() >= keyLen {
			if c.key == nil {
				c.key = data.Next(keyLen)
			} else {
				for i := 0; i < keyLen; i++ {
					byte, _ := data.ReadByte()
					c.key = append(c.key, byte)
				}
			}
			if c.valueLength > 0 {
				c.state = parseStateValue
			} else {
				c.state = parseStateComplete
			}
		} else {
			available := data.Len()
			data.Next(available)
			c.keyLength -= uint16(available)
			return io.EOF
		}

	}

	if c.state == parseStateValue {
		valueLen := int(c.valueLength)

		if data.Len() >= valueLen {
			data.Next(valueLen)
			c.state = parseStateComplete

		} else {
			available := data.Len()
			data.Next(available)
			c.valueLength -= uint32(available)
			return io.EOF
		}
	}
	return nil
}

func (c *Command) isComplete() bool {
	return c.state == parseStateComplete
}

func (c *Command) isResponse() bool {
	return c.commandType == RESPONSE
}