package relay

import (
	"encoding/binary"
	"log"
	"testing"
)

var handle = func() *ClientHandler {
	handle := NewHandler("COM7")
	handle.SlaveId = 1
	handle.BaudRate = 9600
	handle.Parity = "N"
	handle.DataBits = 8
	handle.StopBits = 1
	err := handle.Connect()
	if err != nil {
		log.Fatal(err)
	}
	return handle
}()

func Do(handler *ClientHandler, function byte, data []byte) {
	adu, err := handler.Encode(&ProtocolDataUnit{
		FunctionCode: function,
		Data:         data,
	})
	if err != nil {
		log.Fatal(err)
	}
	response, err := handler.Send(adu)
	if err != nil {
		log.Fatal(err)
	}
	pdu, err := handler.Decode(response)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(pdu)
}

func TestRequestOnAll(t *testing.T) {
	Do(handle, 0x33, []byte{0xff, 0xff, 0xff, 0xff})
}

func TestRequestOffAll(t *testing.T) {
	Do(handle, 0x33, []byte{0, 0, 0, 0})
}

func RequestPointOn(index, time int) {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(time))
	Do(handle, 0x21, []byte{data[1], data[2], data[3], byte(index)})
}

func TestRequestPointOn(t *testing.T) {
	TestRequestOffAll(t)
	for i := 1; i <= 8; i++ {
		RequestPointOn(i, i*1000)
	}
}

func RequestPointOff(index, time int) {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(time))
	Do(handle, 0x22, []byte{data[1], data[2], data[3], byte(index)})
}

func TestRequestPointOff(t *testing.T) {
	TestRequestOnAll(t)
	for i := 1; i <= 8; i++ {
		RequestPointOff(i, i*1000)
	}
}
