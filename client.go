package relay

import (
	"encoding/binary"
	"github.com/zing-dev/go-bit-bytes/bin"
	"github.com/zing-dev/go-bit-bytes/bit"
	"sync"
	"time"
)

// Client is the interface that groups the Packager and Transporter methods.
type Client struct {
	packager    Packager
	transporter Transporter
	length      byte

	from byte
	stat []uint16
	sync.Mutex
}

// NewClient creates a new modbus client with given backend handler.
func NewClient(handler *ClientHandler, length byte) *Client {
	if length < DefaultBranchesLength {
		length = DefaultBranchesLength
	}
	if length > MaxBranchesLength {
		length = MaxBranchesLength
	}
	stat := make([]uint16, length)
	return &Client{
		packager:    handler,
		transporter: handler,
		length:      length,
		stat:        stat,
	}
}

func NewDefaultClient(handler *ClientHandler) *Client {
	return NewClient(handler, DefaultBranchesLength)
}

func (c *Client) SetStatusFrom(from byte) {
	if from != GetStatusFromCache && from != GetStatusFromRelay {
		panic("err get status from")
	}
	c.from = from
}

func (c *Client) GetStats() []uint16 {
	c.Lock()
	defer c.Unlock()
	return c.stat
}

//send 发送有返回数据
func (c *Client) send(code byte, data []byte) ([]byte, error) {
	c.Lock()
	defer c.Unlock()
	if c.packager == nil || c.transporter == nil {
		return nil, ErrPackagerNil
	}
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

//单个继电器路数处理
func (c *Client) one(i, code, result byte) error {
	if i < 1 || i > c.length {
		return ErrBranchesLength
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
	return ErrReturnResult
}

// OffOne 断开某路
func (c *Client) OffOne(i byte) error {
	return c.one(i, RequestOffOne, 0)
}

// OnOne 闭合某路
func (c *Client) OnOne(i byte) error {
	return c.one(i, RequestOnOne, 1)
}

// FlipOne 翻转某路
func (c *Client) FlipOne(i byte) error {
	return c.one(i, RequestFlipOne, 2)
}

// StatusOne 某路继电器状态
func (c *Client) StatusOne(i byte) (byte, error) {
	if i < 1 || i > c.length {
		return 0, ErrBranchesLength
	}
	status, err := c.status()
	if err != nil {
		return 0, err
	}
	return status[i-1], nil
}

// Status 继电器状态
func (c *Client) Status() ([]byte, error) {
	status, err := c.status()
	if err != nil {
		return nil, err
	}
	return status[:c.length], err
}

//最大继电器路数状态 MaxBranchesLength
func (c *Client) status() ([]byte, error) {
	status := make([]byte, MaxBranchesLength)
	data, err := c.send(RequestReadStatus, []byte{0, 0, 0, 0})
	if err != nil {
		return nil, err
	}
	if len(data) != 4 {
		return nil, ErrReturnResult
	}
	for k, v := range data {
		for i := 0; i < 8; i++ {
			status[(3-k)*8+i] = v & (1 << i) >> i
		}
	}
	return status, nil
}

//sendNil 发送无返回数据
func (c *Client) sendNil(code byte, data []byte) error {
	if c.packager == nil || c.transporter == nil {
		return ErrPackagerNil
	}
	adu, err := c.packager.Encode(&ProtocolDataUnit{
		FunctionCode: code,
		Data:         data,
	})
	if err != nil {
		return err
	}
	adu, err = c.transporter.Send(adu)
	c.onNil(code, data)
	return err
}

//组操作
func (c *Client) group(branches ...byte) ([]byte, error) {
	min, max := byte(0), byte(0)
	for _, v := range branches {
		if min > v {
			min = v
		}

		if max < v {
			max = v
		}
	}
	if min < 0 || max >= c.length {
		return nil, ErrBranchesLength
	}

	origin := make([]byte, MaxBranchesLength)
	for key := range origin {
		for _, val := range branches {
			if byte(key) == val {
				origin[key] = 1
			}
		}
	}
	status := make([]byte, 4)
	for i := 0; i < 4; i++ {
		sum := byte(0)
		for k, v := range origin[(3-i)*8 : (4-i)*8] {
			sum += v << k
		}
		status[i] = sum
	}
	return status, nil
}

// OffGroup 断开组
func (c *Client) OffGroup(i ...byte) error {
	group, err := c.group(i...)
	if err != nil {
		return err
	}
	_, err = c.send(RequestOffGroup, group)
	return err
}

// OnGroup 闭合组
func (c *Client) OnGroup(i ...byte) error {
	group, err := c.group(i...)
	if err != nil {
		return err
	}
	_, err = c.send(RequestOnGroup, group)
	return err
}

// FlipGroup 组翻转
func (c *Client) FlipGroup(i ...byte) error {
	group, err := c.group(i...)
	if err != nil {
		return err
	}
	_, err = c.send(RequestFlipGroup, group)
	return err
}

//点动操作
//时间毫秒
func (c *Client) point(code, i byte, time int) error {
	i++
	if i <= 0 || i > c.length {
		return ErrBranchesLength
	}
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(time))
	data, err := c.send(code, []byte{data[1], data[2], data[3], i})
	return err
}

// OffPoint 点动断开某路
func (c *Client) OffPoint(i byte, t int) error {
	return c.point(RequestOffPoint, i, t)
}

// OnPoint 点动闭合某路
func (c *Client) OnPoint(i byte, t int) error {
	return c.point(RequestOnPoint, i, t)
}

// OffAll 断开所有
func (c *Client) OffAll() error {
	return c.sendNil(RequestRunCMDNil, []byte{0, 0, 0, 0})
}

// OnAll 吸合所有
func (c *Client) OnAll() error {
	return c.sendNil(RequestRunCMDNil, []byte{0xff, 0xff, 0xff, 0xff})
}

//某路操作无返回数据
func (c *Client) oneNil(i, code byte) error {
	i++
	if i <= 0 || i > c.length {
		return ErrBranchesLength
	}
	return c.sendNil(code, []byte{0, 0, 0, i})
}

// FlipOneNil 翻转某路
func (c *Client) FlipOneNil(i byte) error {
	return c.oneNil(i, RequestFlipOneNil)
}

// OffOneNil 断开某路
func (c *Client) OffOneNil(i byte) error {
	return c.oneNil(i, RequestOffOneNil)
}

// OnOneNil 吸合某路
func (c *Client) OnOneNil(i byte) error {
	return c.oneNil(i+1, RequestOnOneNil)
}

// OffGroupNil 断开组
func (c *Client) OffGroupNil(i ...byte) error {
	group, err := c.group(i...)
	if err != nil {
		return err
	}
	return c.sendNil(RequestOffGroupNil, group)
}

// OnGroupNil 吸合组
func (c *Client) OnGroupNil(i ...byte) error {
	group, err := c.group(i...)
	if err != nil {
		return err
	}
	return c.sendNil(RequestOnGroupNil, group)
}

// FlipGroupNil 翻转组
func (c *Client) FlipGroupNil(i ...byte) error {
	group, err := c.group(i...)
	if err != nil {
		return err
	}
	return c.sendNil(RequestFlipGroupNil, group)
}

//点动处理无返回数据
//0 <= i && i < c.length time 毫秒
func (c *Client) pointNil(code, i byte, time int) error {
	i++
	if i <= 0 || i > c.length {
		return ErrBranchesLength
	}
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(time))
	return c.sendNil(code, []byte{data[1], data[2], data[3], i})
}

// OnPointNil 点动闭合
func (c *Client) OnPointNil(i byte, t int) error {
	return c.pointNil(RequestOnPointNil, i, t)
}

// OffPointNil 点动断开
func (c *Client) OffPointNil(i byte, t int) error {
	return c.pointNil(RequestOffPointNil, i, t)
}

//当不需要操作的返回值,继电器的状态值有自己控制
func (c *Client) onNil(code byte, data []byte) {
	c.Lock()
	defer c.Unlock()
	switch code {
	case RequestOffOneNil:
		c.stat[data[3]-1] = 0
	case RequestOnPointNil:
		time.AfterFunc(time.Millisecond*time.Duration(bit.ToUint32(append([]byte{0}, data[:3]...))-10), func() {
			c.Lock()
			c.stat[data[3]-1] = 0
			defer c.Unlock()
		})
	case RequestOffPointNil:
		time.AfterFunc(time.Millisecond*time.Duration(bit.ToUint32(append([]byte{0}, data[:3]...))-10), func() {
			c.Lock()
			c.stat[data[3]-1] = 1
			defer c.Unlock()
		})
	case RequestOffGroupNil, RequestOnGroupNil:
		d := bin.Revert(bin.FromInt(bit.ToInt(data)))
		//数据区域共4个字节，每个字节8位，共32位。
		//最多代表对32路的操作，1代表断开 0代表保持原来状态。最后一个字节的第0位(BIT0)代表第一路，依次类推。
		if code == RequestOffGroupNil {
			for i := range c.stat {
				if c.stat[i] == 1 && d[i] == 1 {
					c.stat[i] = 0
				}
			}
		} else {
			//数据区域共4个字节，每个字节8位，共32位。
			//最多代表对32路的操作，1 代表吸合 0代表保持原来状态。最后一个字节的第0位(BIT0)代表第一路，依次类推。
			for i := range c.stat {
				if c.stat[i] == 0 && d[i] == 1 {
					c.stat[i] = 1
				}
			}
		}
	case RequestOnOneNil:
		c.stat[data[3]-1] = 1
	case RequestFlipOneNil:
		s := c.stat[data[3]-1]
		if s == 0 {
			c.stat[data[3]-1] = 1
		} else {
			c.stat[data[3]-1] = 0
		}
	case RequestRunCMDNil:
		if data[0] == 0 {
			c.stat = make([]uint16, c.length)
		} else {
			for i := range c.stat {
				c.stat[i] = 1
			}
		}
	}
}
