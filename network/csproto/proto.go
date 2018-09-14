package csproto

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"server/leaf/chanrpc"
	"server/leaf/log"
)

// 自定义协议解释
// -------------------------
// | tag| len | id | data |
// -------------------------
type Processor struct {
	littleEndian bool
	msgInfo      map[int]*MsgInfo
	clientInfo   map[reflect.Type]int
	serverInfo   map[reflect.Type]int
}

type MsgInfo struct {
	msgType       reflect.Type
	msgRouter     *chanrpc.Server
	msgHandler    MsgHandler
	msgRawHandler MsgHandler
}

type MsgHandler func([]interface{})

type MsgRaw struct {
	msgID      int
	msgRawData []byte
}

func NewProcessor() *Processor {
	p := new(Processor)
	p.littleEndian = true //false
	p.msgInfo = make(map[int]*MsgInfo)
	p.clientInfo = make(map[reflect.Type]int)
	p.serverInfo = make(map[reflect.Type]int)
	return p
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetByteOrder(littleEndian bool) {
	p.littleEndian = littleEndian
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) RegisterClient(msgID int, msg interface{}) int {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("csproto message pointer required")
	}
	if _, ok := p.msgInfo[msgID]; ok {
		log.Fatal("message %v is already registered", msgID)
	}

	i := new(MsgInfo)
	i.msgType = msgType
	p.msgInfo[msgID] = i
	p.clientInfo[msgType] = msgID
	return msgID
}

func (p *Processor) RegisterServer(msgID int, msg interface{}) int {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("csproto message pointer required")
	}
	_, ok := p.serverInfo[msgType]
	if ok {
		log.Fatal("RegisterServer message %v is registered", msgID)
	}
	p.serverInfo[msgType] = msgID
	return msgID
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetRouter(msg interface{}, msgRouter *chanrpc.Server) {
	msgType := reflect.TypeOf(msg)
	msgID, ok := p.clientInfo[msgType]
	if !ok {
		log.Fatal("SetRouter message %v not registered", msgID)
	}
	i, ok := p.msgInfo[msgID]
	if !ok {
		log.Fatal("SetRouter message %v not registered", msgID)
	}
	i.msgRouter = msgRouter
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetHandler(msgID int, msgHandler MsgHandler) {
	i, ok := p.msgInfo[msgID]
	if !ok {
		log.Fatal("SetHandler message %v not registered", msgID)
	}

	i.msgHandler = msgHandler
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetRawHandler(cmdId int, msgRawHandler MsgHandler) {
	if cmdId >= len(p.msgInfo) {
		log.Fatal("SetRawHandler message id %v not registered", cmdId)
	}

	p.msgInfo[cmdId].msgRawHandler = msgRawHandler
}

// goroutine safe
func (p *Processor) Route(msg interface{}, userData interface{}) error {
	msgType := reflect.TypeOf(msg)
	msgID, ok := p.clientInfo[msgType]

	if !ok {
		return fmt.Errorf("Route message %v not registered", msgType)
	}
	// protobuf
	i, ok := p.msgInfo[msgID]
	if !ok {
		return fmt.Errorf("Route message %s not registered", msgID)
	}
	if i.msgHandler != nil {
		i.msgHandler([]interface{}{msg, userData})
	}
	if i.msgRouter != nil {
		i.msgRouter.Go(msgType, msg, userData)
	}
	return nil
}

// goroutine safe
func (p *Processor) Unmarshal(data []byte) (interface{}, error) {
	if len(data) < 12 {
		return nil, errors.New("csproto data too short")
	}

	var tag, dataLen, cmdId int
	reader := bytes.NewReader(data)
	Read(reader, &tag, &dataLen, &cmdId)
	if cmdId <= 0 {
		return nil, fmt.Errorf("error cmdId <= 0")
	}

	i, ok := p.msgInfo[cmdId]
	if !ok {
		return nil, fmt.Errorf("error cmdId:%v, not registered msgInfo", cmdId)
	}
	msg := reflect.New(i.msgType.Elem()).Interface()
	value := reflect.ValueOf(msg).Elem()

	var err error
	msgType := i.msgType
	for idx := 0; idx < msgType.Elem().NumField(); idx++ {
		kind := msgType.Elem().Field(idx).Type.Kind()
		switch kind {
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			var v int
			err = Read(reader, &v)
			if err != nil {
				return nil, err
			}
			value.Field(idx).SetInt(int64(v))
		case reflect.String:
			var v string
			err = Read(reader, &v)
			if err != nil {
				return nil, err
			}
			value.Field(idx).SetString(v)
		case reflect.Float32, reflect.Float64:
			var v float64
			err = Read(reader, &v)
			if err != nil {
				return nil, err
			}
			value.Field(idx).SetFloat(float64(v))
		default:
			err = fmt.Errorf("data kind error :%v", kind)
			return nil, err
		}
	}
	/*
		if cmdId != 10001 {
			log.Debug("read msg cmdId:%v msgType:%v, msg:%v", cmdId, msgType, msg)
		}
	*/
	// msg
	if i.msgRawHandler != nil {
		return MsgRaw{cmdId, data[12:]}, nil
	} else {
		return msg, err
	}
}

func EncodeTypeStruct(writer *bytes.Buffer, value reflect.Value) error {
	for idx := 0; idx < value.NumField(); idx++ {
		v := value.Field(idx)
		err := EncodeTypeByte(writer, v.Kind(), v)
		if err != nil {
			return err
		}
	}
	return nil
}

func EncodeTypeByte(writer *bytes.Buffer, kind reflect.Kind, value reflect.Value) error {
	switch kind {
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		v := value.Int()
		Write(writer, int(v))
		return nil
	case reflect.String:
		v := value.String()
		Write(writer, v)
		return nil
	case reflect.Float32, reflect.Float64:
		v := value.Float()
		Write(writer, v)
		return nil
	case reflect.Struct:
		err := EncodeTypeStruct(writer, value)
		return err
	}

	err := fmt.Errorf("data kind error :%v", kind)
	return err
}

func (p *Processor) Marshal(msg interface{}) ([][]byte, error) {
	msgType := reflect.TypeOf(msg)
	cmdId, ok := p.serverInfo[msgType]
	if !ok {
		return nil, fmt.Errorf("Marshal message %v not registered", msgType)
	}

	/*
		if cmdId != 10001 {
			log.Debug("write cmdId:%v msgType:%v, msg:%v", cmdId, msgType, msg)
		}
	*/
	writer := bytes.NewBuffer([]byte{})
	writer.Write(header)
	writer.Write(GetBytes(cmdId))

	value := reflect.ValueOf(msg)
	for idx := 0; idx < msgType.Elem().NumField(); idx++ {
		kind := msgType.Elem().Field(idx).Type.Kind()
		v := value.Elem().Field(idx)
		switch kind {
		case reflect.Array, reflect.Slice:
			writer.Write(GetBytes(v.Len()))
			for i := 0; i < v.Len(); i++ {
				k := v.Index(i).Kind()
				err := EncodeTypeByte(writer, k, v.Index(i))
				if err != nil {
					return nil, err
				}
			}
		default:
			err := EncodeTypeByte(writer, kind, v)
			if err != nil {
				return nil, err
			}
		}
	}
	data := EncodeWriter(writer)
	//log.Debug("senddata ======== %v", data)
	return [][]byte{data}, nil
}

// goroutine safe
func (p *Processor) Range(f func(id uint16, t reflect.Type)) {
	for id, i := range p.msgInfo {
		f(uint16(id), i.msgType)
	}
}
