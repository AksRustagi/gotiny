package gotiny

import (
	"reflect"
	"unsafe"
)

type Decoder struct {
	buf     []byte //buf
	index   int    //下一个要使用的字节在buf中的下标
	boolPos byte   //下一次要读取的bool在buf中的下标,即buf[boolPos]
	boolBit byte   //下一次要读取的bool的buf[boolPos]中的bit位

	engines []decEng //解码器集合
	length  int      //解码器数量
}

// Unmarshal decodes any object from byte array
// For performance recommended to create schema first using gotiny.New
func Unmarshal(buf []byte, is ...interface{}) int {
	return NewDecoderWithPtr(is...).Decode(buf, is...)
}

// NewDecoderWithPtr creates decoder using pointer
// Note: decoder is not threadsafe, use gotiny.NewWithPtr instead
func NewDecoderWithPtr(is ...interface{}) *Decoder {
	return NewWithPtr(is...).GetDecoder()
}

// NewDecoder creates decoder using element of object type
// Note: decoder is not threadsafe, use gotiny.New instead
func NewDecoder(is ...interface{}) *Decoder {
	return New(is...).GetDecoder()
}

// NewDecoderWithType creates decoder using reflect.Type
// Note: decoder is not threadsafe, use gotiny.NewWithType instead
func NewDecoderWithType(ts ...reflect.Type) *Decoder {
	return NewWithType(ts...).GetDecoder()
}

func (d *Decoder) reset() int {
	index := d.index
	d.index = 0
	d.boolPos = 0
	d.boolBit = 0
	return index
}

// Decode decode to buffer; is is pointer of variable
func (d *Decoder) Decode(buf []byte, is ...interface{}) int {
	d.buf = buf
	engines := d.engines
	for i := 0; i < len(engines) && i < len(is); i++ {
		engines[i](d, (*[2]unsafe.Pointer)(unsafe.Pointer(&is[i]))[1])
	}
	return d.reset()
}

// DecodePtr decode to pointer; ps is a unsafe.Pointer of the variable
func (d *Decoder) DecodePtr(buf []byte, ps ...unsafe.Pointer) int {
	d.buf = buf
	engines := d.engines
	for i := 0; i < len(engines) && i < len(ps); i++ {
		engines[i](d, ps[i])
	}
	return d.reset()
}

// DecodeValue decode to value
func (d *Decoder) DecodeValue(buf []byte, vs ...reflect.Value) int {
	d.buf = buf
	engines := d.engines
	for i := 0; i < len(engines) && i < len(vs); i++ {
		engines[i](d, unsafe.Pointer(vs[i].UnsafeAddr()))
	}
	return d.reset()
}
