package relay

import (
	"fmt"
	"testing"
	"time"
)

var client = NewClient(handle)

func TestNewClient(t *testing.T) {
	_ = client.OnAll()
	time.Sleep(time.Second)
	_ = client.OffAll()
	time.Sleep(time.Second)
	_ = client.OnAll()
}

func TestClient_OffOne(t *testing.T) {
	client.OnAll()
	client.OffOne(1)
	client.OffOne(2)
	client.OffOne(3)
	client.OffOne(4)
	client.OffOne(5)
	client.OffOne(6)
	client.OffOne(7)
	client.OffOne(8)
	client.OffOne(32)
}

func TestClient_OnOne(t *testing.T) {
	client.OffAll()
	fmt.Println(client.OnOne(1))
	fmt.Println(client.OnOne(2))
	fmt.Println(client.OnOne(4))
	fmt.Println(client.OnOne(6))
	fmt.Println(client.OnOne(8))
	fmt.Println(client.OnOne(9))
	fmt.Println(client.OnOne(10))
}

func TestClient_FlipOne(t *testing.T) {
	fmt.Println(client.FlipOne(1))
	fmt.Println(client.StatusOne(1))
}

func TestClient_Status(t *testing.T) {
	client.OffAll()
	fmt.Println(client.OnOne(1))
	fmt.Println(client.OnOne(7))
	fmt.Println(client.OnOne(10))
	fmt.Println(client.Status())
}

func TestClient_StatusOne(t *testing.T) {
	client.OffAll()
	fmt.Println(client.OnOne(0))
	fmt.Println(client.OnOne(1))
	fmt.Println(client.OnOne(2))
	fmt.Println(client.OnOne(7))
	fmt.Println(client.OnOne(8))
	fmt.Println(client.StatusOne(0))
	fmt.Println(client.StatusOne(1))
	fmt.Println(client.StatusOne(2))
	fmt.Println(client.StatusOne(7))
	fmt.Println(client.StatusOne(8))
}

func TestClient_OffGroup(t *testing.T) {
	client.OnAll()
	fmt.Println(client.OffGroup(0, 2, 4, 6))
}

func TestClient_OnGroup(t *testing.T) {
	client.OffAll()
	fmt.Println(client.OnGroup(0, 2, 4, 6))
}

func TestClient_FlipGroup(t *testing.T) {
	fmt.Println(client.FlipGroup(0, 2, 4, 6))
	fmt.Println(client.StatusOne(0))
	fmt.Println(client.StatusOne(2))
	fmt.Println(client.StatusOne(4))
	fmt.Println(client.StatusOne(6))
}

func TestClient_OffPoint(t *testing.T) {
	for i := 0; i < 8; i++ {
		fmt.Println(client.OffPoint(byte(i), 1000*(i+1)))
	}
}

func TestClient_OnPoint(t *testing.T) {
	for i := 0; i < 8; i++ {
		fmt.Println(client.OnPoint(byte(i), 1000*(i+1)))
	}
}
