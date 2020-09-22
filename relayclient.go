// Copyright 2014 Quoc-Viet Nguyen. All rights reserved.
// This software may be modified and distributed under the terms
// of the BSD license. See the LICENSE file for details.

package relay

import (
	"fmt"
	"io"
	"log"
	"time"
)

const (
	relayMinSize = 4
	relayMaxSize = 256

	relayExceptionSize = 5
)

// RELAYClientHandler implements Packager and Transporter interface.
type ClientHandler struct {
	relayPackager
	relaySerialTransporter
}

// NewClientHandler allocates and initializes a RELAYClientHandler.
func NewClientHandler(address string) *ClientHandler {
	handler := &ClientHandler{}
	handler.Address = address
	handler.Timeout = serialTimeout
	handler.IdleTimeout = serialIdleTimeout
	return handler
}

// relayPackager implements Packager interface.
type relayPackager struct {
	SlaveId byte
}

// Encode encodes PDU in a RELAY frame:
//  Data Header     : 1 byte
//  Slave Address   : 1 byte
//  Function        : 1 byte
//  Data            : 4 bytes
//  CRC             : 1 byte
func (mb *relayPackager) Encode(pdu *ProtocolDataUnit) (adu []byte, err error) {
	length := len(pdu.Data) + 4
	if length > relayMaxSize {
		err = fmt.Errorf("serial: length of data '%v' must not be bigger than '%v'", length, relayMaxSize)
		return
	}
	adu = make([]byte, length)
	adu[0] = 0x55
	adu[1] = mb.SlaveId
	adu[2] = pdu.FunctionCode
	copy(adu[3:], pdu.Data)
	adu[7] = Sign(adu)
	return
}

// Verify verifies response length and slave id.
func (mb *relayPackager) Verify(aduRequest []byte, aduResponse []byte) (err error) {
	length := len(aduResponse)
	// Minimum size (including address, function and CRC)
	if length < relayMinSize {
		err = fmt.Errorf("serial: response length '%v' does not meet minimum '%v'", length, relayMinSize)
		return
	}
	// Slave address must match
	if aduResponse[0] != aduRequest[0] {
		err = fmt.Errorf("serial: response slave id '%v' does not match request '%v'", aduResponse[0], aduRequest[0])
		return
	}
	return
}

// Decode extracts PDU from RELAY frame and verify CRC.
func (mb *relayPackager) Decode(adu []byte) (pdu *ProtocolDataUnit, err error) {
	if len(adu) < 8 {
		return
	}
	length := len(adu)
	if adu[7] != Sign(adu) {
		err = fmt.Errorf("serial: response crc '%v' does not match expected '%v'", adu[7], Sign(adu))
		return
	}
	// Function code & data
	pdu = &ProtocolDataUnit{}
	pdu.FunctionCode = adu[2]
	pdu.Data = adu[3 : length-1]
	return
}

// relaySerialTransporter implements Transporter interface.
type relaySerialTransporter struct {
	serialPort
}

func (mb *relaySerialTransporter) Send(aduRequest []byte) (aduResponse []byte, err error) {
	// Make sure port is connected
	if err = mb.serialPort.connect(); err != nil {
		return
	}
	// Start the timer to close when idle
	mb.serialPort.lastActivity = time.Now()
	mb.serialPort.startCloseTimer()

	// Send the request
	mb.serialPort.logf("serial: sending % x\n", aduRequest)
	//aduRequest = []byte{0x55, 0x01, 0x33, 0xff, 0xff, 0xff, 0xff, 0x85}
	if _, err = mb.port.Write(aduRequest); err != nil {
		return
	}
	function := aduRequest[2]
	functionFail := aduRequest[2] & 0x80
	bytesToRead := calculateRelayResponseLength(function)
	if bytesToRead == 0 {
		return
	}
	time.Sleep(mb.calculateDelay(len(aduRequest) + bytesToRead))

	var n int
	var n1 int
	var data [relayMaxSize]byte
	//We first read the minimum length and then read either the full package
	//or the error package, depending on the error status (byte 2 of the response)
	n, err = io.ReadAtLeast(mb.port, data[:], relayMinSize)
	if err != nil {
		return
	}
	//if the function is correct
	if data[2] == function {
		//we read the rest of the bytes
		if n < bytesToRead {
			if bytesToRead > relayMinSize && bytesToRead <= relayMaxSize {
				if bytesToRead > n {
					n1, err = io.ReadFull(mb.port, data[n:bytesToRead])
					n += n1
				}
			}
		}
	} else if data[2] == functionFail {
		//for error we need to read 5 bytes
		if n < relayExceptionSize {
			n1, err = io.ReadFull(mb.port, data[n:relayExceptionSize])
		}
		n += n1
	}

	if err != nil {
		return
	}
	log.Println(n)
	aduResponse = data[:n]
	mb.serialPort.logf("serial: received % x\n", aduResponse)
	return
}

// calculateDelay roughly calculates time needed for the next frame.
// See serial over Serial Line - Specification and Implementation Guide (page 13).
func (mb *relaySerialTransporter) calculateDelay(chars int) time.Duration {
	var characterDelay, frameDelay int // us

	if mb.BaudRate <= 0 || mb.BaudRate > 19200 {
		characterDelay = 750
		frameDelay = 1750
	} else {
		characterDelay = 15000000 / mb.BaudRate
		frameDelay = 35000000 / mb.BaudRate
	}
	return time.Duration(characterDelay*chars+frameDelay) * time.Microsecond
}

func calculateRelayResponseLength(function byte) int {
	if 0x30 <= function && function <= 0x38 {
		return 0
	}
	return 8
}

//数据校验位赋值
func Sign(data []byte) byte {
	sum := byte(0)
	for i := byte(0); i < 7; i += 1 {
		sum += data[i]
	}
	return 0xff & sum
}
