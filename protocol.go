package relay

import "errors"

const (
	DefaultBranchesLength = 0x8
	MaxBranchesLength     = 32
	DataLength            = 0x8
	RequestHeader         = 0x55 //发送帧数据头
	ResponseHeader        = 0x22 //接受帧数据头

	//功能码
	RequestReadStatus = 0x10 //读取状态

	RequestOffOne  = 0x11 //断开某路
	RequestOnOne   = 0x12 //吸合某路
	RequestFlipOne = 0x20 //翻转某路

	RequestRunCMD = 0x13 //命令执行

	RequestOffGroup  = 0x14 //组断开
	RequestOnGroup   = 0x15 //组吸合
	RequestFlipGroup = 0x16 //组翻转

	RequestOnPoint  = 0x21 //点动闭合
	RequestOffPoint = 0x22 //点动断开

	RequestFlipOneNil = 0x30 //翻转某路 下位机不返回数据，指0令可以连续发送
	RequestOffOneNil  = 0x31 //断开某路
	RequestOnOneNil   = 0x32 //吸合某路
	RequestRunCMDNil  = 0x33 //命令执行

	RequestOffGroupNil  = 0x34 //组断开
	RequestOnGroupNil   = 0x35 //组吸合
	RequestFlipGroupNil = 0x36 //组翻转

	RequestOnPointNil  = 0x37 //点动闭合
	RequestOffPointNil = 0x38 //点动断开

	RequestReadAddress  = 0x40 //读地址
	RequestWriteAddress = 0x41 //写地址

	RequestReadVariable  = 0x70 //读变量
	RequestWriteVariable = 0x71 //写变量

	ResponseReadStatus         = 0x10 //读取状态
	ResponseCloseOne           = 0x11 //关闭某一路
	ResponseOpenOne            = 0x12 //打开某一路
	ResponseRunCMD             = 0x13 //命令执行
	ResponseCloseGroup         = 0x14 //组断开
	ResponseOpenGroup          = 0x15 //组吸合
	ResponseFlipGroup          = 0x16 //组翻转
	ResponseModelAddress       = 0x40 //返回模块地址
	ResponseReadInnerVariable  = 0x70 //读内部变量
	ResponseWriteInnerVariable = 0x71 //写内部变量
)

var (
	ErrPackagerNil    = errors.New("packager 未实例化")
	ErrBranchesLength = errors.New("继电器路数超出范围,1~32")
	ErrReturnResult   = errors.New("串口返回数据格式异常")
)

// ProtocolDataUnit (PDU) is independent of underlying communication layers.
type ProtocolDataUnit struct {
	FunctionCode byte
	Data         []byte
}

// Packager specifies the communication layer.
type Packager interface {
	Encode(pdu *ProtocolDataUnit) (adu []byte, err error)
	Decode(adu []byte) (pdu *ProtocolDataUnit, err error)
	Verify(aduRequest []byte, aduResponse []byte) (err error)
}

// Transporter specifies the transport layer.
type Transporter interface {
	Send(aduRequest []byte) (aduResponse []byte, err error)
}
