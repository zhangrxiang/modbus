package relay

import (
	"encoding/binary"
	"errors"
)

// Client is the interface that groups the Packager and Transporter methods.
type Client struct {
	packager    Packager
	transporter Transporter
}

// NewClient creates a new modbus client with given backend handler.
func NewClient(handler *ClientHandler) *Client {
	return &Client{packager: handler, transporter: handler}
}

func (c *Client) send(code byte, data []byte) ([]byte, error) {
	adu, err := c.packager.Encode(&ProtocolDataUnit{
		FunctionCode: code,
		Data:         data,
	})
	if err != nil {
		return nil, err
	}
	adu, err = c.transporter.Send(adu)
	if err != nil {
		return nil, err
	}
	pdu, err := c.packager.Decode(adu)
	if err != nil {
		return nil, err
	}
	return pdu.Data, nil
}

func (c *Client) one(i, code, result byte) error {
	if i <= 0 || i >= MaxBranchesLength {
		return errors.New("too large or small")
	}
	data, err := c.send(code, []byte{0, 0, 0, i})
	if err != nil {
		return err
	}
	//继电器输出板或者输入检测板：数据区域 4 个字节，每个字节 8 位，共 32 位。代表 32 路的状
	//态。最后一个字节的第 0 位代表第 1 路，依次类推。
	if result == 0 && byte(binary.BigEndian.Uint32(data)&(1<<(i-1))) == result {
		return nil
	} else if result == 1 && byte(binary.BigEndian.Uint32(data)&(1<<(i-1))>>(i-1)) == result {
		return nil
	} else if result == 2 {
		return nil
	}
	return errors.New("error")
}

func (c *Client) OffOne(i byte) error {
	return c.one(i+1, RequestOffOne, 0)
}

func (c *Client) OnOne(i byte) error {
	return c.one(i+1, RequestOnOne, 1)
}

func (c *Client) FlipOne(i byte) error {
	return c.one(i+1, RequestFlipOne, 2)
}

func (c *Client) StatusOne(i byte) (byte, error) {
	status, err := c.status()
	if err != nil {
		return 0, err
	}
	return status[i], nil
}

func (c *Client) Status() ([]byte, error) {
	return c.status()
}

func (c *Client) status() ([]byte, error) {
	status := make([]byte, MaxBranchesLength)
	data, err := c.send(RequestReadStatus, []byte{0, 0, 0, 0})
	if err != nil {
		return nil, err
	}
	for k, v := range data {
		for i := 0; i < 8; i++ {
			status[(3-k)*8+i] = v & (1 << i) >> i
		}
	}
	return status, nil
}

func (c *Client) sendNil(code byte, data []byte) error {
	adu, err := c.packager.Encode(&ProtocolDataUnit{
		FunctionCode: code,
		Data:         data,
	})
	if err != nil {
		return err
	}
	adu, err = c.transporter.Send(adu)
	return err
}

func (c *Client) group(branches ...byte) (status []byte) {
	origin := make([]byte, MaxBranchesLength)
	for key := range origin {
		for _, val := range branches {
			if byte(key) == val {
				origin[key] = 1
			}
		}
	}
	status = make([]byte, 4)
	for i := 0; i < 4; i++ {
		sum := byte(0)
		for k, v := range origin[(3-i)*8 : (4-i)*8] {
			sum += v << k
		}
		status[i] = sum
	}
	return
}

func (c *Client) OffGroup(i ...byte) error {
	_, err := c.send(RequestOffGroup, c.group(i...))
	return err
}

func (c *Client) OnGroup(i ...byte) error {
	_, err := c.send(RequestOnGroup, c.group(i...))
	return err
}

func (c *Client) FlipGroup(i ...byte) error {
	_, err := c.send(RequestFlipGroup, c.group(i...))
	return err
}

func (c *Client) point(code, i byte, time int) error {
	i++
	if i <= 0 || i >= MaxBranchesLength {
		return errors.New("too large or small")
	}
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(time))
	data, err := c.send(code, []byte{data[1], data[2], data[3], i})
	return err
}

func (c *Client) OffPoint(i byte, t int) error {
	return c.point(RequestOffPoint, i, t)
}

func (c *Client) OnPoint(i byte, t int) error {
	return c.point(RequestOnPoint, i, t)
}

func (c *Client) OffAll() error {
	return c.sendNil(RequestRunCMDNil, []byte{0, 0, 0, 0})
}

func (c *Client) OnAll() error {
	return c.sendNil(RequestRunCMDNil, []byte{0xff, 0xff, 0xff, 0xff})
}

func (c *Client) oneNil(i, code byte) error {
	if i <= 0 || i >= MaxBranchesLength {
		return errors.New("too large or small")
	}
	return c.sendNil(code, []byte{0, 0, 0, i})
}

func (c *Client) FlipOneNil(i byte) error {
	return c.oneNil(i+1, RequestFlipOneNil)
}

func (c *Client) OffOneNil(i byte) error {
	return c.oneNil(i+1, RequestOffOneNil)
}

func (c *Client) OnOneNil(i byte) error {
	return c.oneNil(i+1, RequestOnOneNil)
}

func (c *Client) OffGroupNil(i ...byte) error {
	return c.sendNil(RequestOffGroupNil, c.group(i...))
}

func (c *Client) OnGroupNil(i ...byte) error {
	return c.sendNil(RequestOnGroupNil, c.group(i...))
}

func (c *Client) FlipGroupNil(i ...byte) error {
	return c.sendNil(RequestFlipGroupNil, c.group(i...))
}

func (c *Client) pointNil(code, i byte, time int) error {
	i++
	if i <= 0 || i >= MaxBranchesLength {
		return errors.New("too large or small")
	}
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(time))
	return c.sendNil(code, []byte{data[1], data[2], data[3], i})
}

func (c *Client) OnPointNil(i byte, t int) error {
	return c.pointNil(RequestOnPointNil, i, t)
}

func (c *Client) OffPointNil(i byte, t int) error {
	return c.pointNil(RequestOffPointNil, i, t)
}
