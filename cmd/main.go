package main

import (
	"github.com/zing-dev/relay-xk-sdk"
	"log"
	"time"
)

func main() {
	address := "/dev/ttyUSB0"
	handle := relay.NewHandler(address)
	handle.SlaveId = 1
	handle.BaudRate = 9600
	handle.Parity = "N"
	handle.DataBits = 8
	handle.StopBits = 1
	err := handle.Connect()
	if err != nil {
		log.Fatal("Connect: ", err)
	}
	client := relay.NewClient(handle, relay.DefaultBranchesLength)
	_ = client.OnAll()
	time.Sleep(time.Second)
	_ = client.OffAll()
	time.Sleep(time.Second)
	_ = client.OnAll()
	time.Sleep(time.Second)
	i := 1
	for i < 8 {
		_ = client.OffOne(byte(i))
		i++
	}
	time.Sleep(time.Second)
}
