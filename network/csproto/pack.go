package csproto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
)

const (
	DEFAULT_TAG int = 0xccee
	HEAD_SIZE       = 12
)

var (
	ReadEOF = errors.New("pack.Read EOF")
	header  = GetBytes(DEFAULT_TAG, 0)
)

type LString string

func NewWriter(datas ...interface{}) *bytes.Buffer {
	writer := bytes.NewBuffer([]byte{})
	Write(writer, datas...)
	return writer
}

func GetBytes(datas ...interface{}) []byte {
	writer := NewWriter(datas...)
	return writer.Bytes()
}

func Read(reader *bytes.Reader, datas ...interface{}) error {
	for _, data := range datas {
		switch v := data.(type) {
		case *bool, *int8, *uint8, *int16, *uint16, *int32, *uint32, *int64, *uint64, *float32, *float64:
			err := binary.Read(reader, binary.LittleEndian, v)
			if err != nil {
				return err
			}
		case *int:
			var vv int32
			err := binary.Read(reader, binary.LittleEndian, &vv)
			if err != nil {
				return err
			}
			*v = int(vv)
		case *string:
			var l uint16
			err := binary.Read(reader, binary.LittleEndian, &l)
			if err != nil {
				return err
			}
			s := make([]byte, l)
			n, _ := reader.Read(s)
			if uint16(n) < l {
				return ReadEOF
			}
			*v = string(s)
			_, err = reader.ReadByte()
			if err != nil {
				return err
			}
		case *LString:
			var l uint64
			err := binary.Read(reader, binary.LittleEndian, &l)
			if err != nil {
				return err
			}
			s := make([]byte, l)
			n, _ := reader.Read(s)
			if uint64(n) < l {
				return ReadEOF
			}
			*v = LString(s)
			_, err = reader.ReadByte()
			if err != nil {
				return ReadEOF
			}
		default:
			println("pack.Read invalid type " + reflect.TypeOf(data).String())
			return ReadEOF
		}
	}
	return nil
}

func Write(writer *bytes.Buffer, datas ...interface{}) {
	for _, data := range datas {
		switch v := data.(type) {
		case bool, int8, uint8, int16, uint16, int32, uint32, int64, uint64, float32, float64:
			binary.Write(writer, binary.LittleEndian, v)
		case int:
			binary.Write(writer, binary.LittleEndian, int32(v))
		case []byte:
			writer.Write(v)
		case string:
			binary.Write(writer, binary.LittleEndian, uint16(len(v)))
			writer.Write([]byte(v))
			binary.Write(writer, binary.LittleEndian, byte(0))
		case LString:
			binary.Write(writer, binary.LittleEndian, uint64(len(v)))
			writer.Write([]byte(v))
			binary.Write(writer, binary.LittleEndian, byte(0))
		default:
			panic("pack.Write invalid type " + reflect.TypeOf(data).String())
		}
	}
}

func AllocPack(cmdId int, data ...interface{}) *bytes.Buffer {
	writer := NewWriter(header, cmdId)
	Write(writer, data...)
	return writer
}

func EncodeWriter(writer *bytes.Buffer) []byte {
	data := writer.Bytes()
	copy(data[4:], GetBytes(len(data)-HEAD_SIZE))
	return data
}
