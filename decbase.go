package gotiny

import (
	"reflect"
	"sync"
	"time"
	"unsafe"
)

type decEng func(*Decoder, unsafe.Pointer) // 解码器

var (
	decEngines = []decEng{
		reflect.Invalid:       decIgnore,
		reflect.Bool:          decBool,
		reflect.Int:           decInt,
		reflect.Int8:          decInt8,
		reflect.Int16:         decInt16,
		reflect.Int32:         decInt32,
		reflect.Int64:         decInt64,
		reflect.Uint:          decUint,
		reflect.Uint8:         decUint8,
		reflect.Uint16:        decUint16,
		reflect.Uint32:        decUint32,
		reflect.Uint64:        decUint64,
		reflect.Uintptr:       decUintptr,
		reflect.UnsafePointer: decPointer,
		reflect.Float32:       decFloat32,
		reflect.Float64:       decFloat64,
		reflect.Complex64:     decComplex64,
		reflect.Complex128:    decComplex128,
		reflect.String:        decString,
	}
	decLock sync.RWMutex
)

func (d *Decoder) decBool() (b bool) {
	if d.boolBit == 0 {
		d.boolBit = 1
		d.boolPos = d.buf[d.index]
		d.index++
	}
	b = d.boolPos&d.boolBit != 0
	d.boolBit <<= 1
	return
}

func (d *Decoder) decUint64() uint64 {
	buf, i := d.buf, d.index
	x := uint64(buf[i])
	if x < 0x80 {
		d.index++
		return x
	}
	x1 := buf[i+1]
	x += uint64(x1) << 7
	if x1 < 0x80 {
		d.index += 2
		return x - 1<<7
	}
	x2 := buf[i+2]
	x += uint64(x2) << 14
	if x2 < 0x80 {
		d.index += 3
		return x - (1<<7 + 1<<14)
	}
	x3 := buf[i+3]
	x += uint64(x3) << 21
	if x3 < 0x80 {
		d.index += 4
		return x - (1<<7 + 1<<14 + 1<<21)
	}
	x4 := buf[i+4]
	x += uint64(x4) << 28
	if x4 < 0x80 {
		d.index += 5
		return x - (1<<7 + 1<<14 + 1<<21 + 1<<28)
	}
	x5 := buf[i+5]
	x += uint64(x5) << 35
	if x5 < 0x80 {
		d.index += 6
		return x - (1<<7 + 1<<14 + 1<<21 + 1<<28 + 1<<35)
	}
	x6 := buf[i+6]
	x += uint64(x6) << 42
	if x6 < 0x80 {
		d.index += 7
		return x - (1<<7 + 1<<14 + 1<<21 + 1<<28 + 1<<35 + 1<<42)
	}
	x7 := buf[i+7]
	x += uint64(x7) << 49
	if x7 < 0x80 {
		d.index += 8
		return x - (1<<7 + 1<<14 + 1<<21 + 1<<28 + 1<<35 + 1<<42 + 1<<49)
	}
	d.index += 9
	return x + uint64(buf[i+8])<<56 - (1<<7 + 1<<14 + 1<<21 + 1<<28 + 1<<35 + 1<<42 + 1<<49 + 1<<56)
}

func (d *Decoder) decUint16() uint16 {
	buf, i := d.buf, d.index
	x := uint16(buf[i])
	if x < 0x80 {
		d.index++
		return x
	}
	x1 := buf[i+1]
	x += uint16(x1) << 7
	if x1 < 0x80 {
		d.index += 2
		return x - 1<<7
	}
	d.index += 3
	return x + uint16(buf[i+2])<<14 - (1<<7 + 1<<14)
}

func (d *Decoder) decUint32() uint32 {
	buf, i := d.buf, d.index
	x := uint32(buf[i])
	if x < 0x80 {
		d.index++
		return x
	}
	x1 := buf[i+1]
	x += uint32(x1) << 7
	if x1 < 0x80 {
		d.index += 2
		return x - 1<<7
	}
	x2 := buf[i+2]
	x += uint32(x2) << 14
	if x2 < 0x80 {
		d.index += 3
		return x - (1<<7 + 1<<14)
	}
	x3 := buf[i+3]
	x += uint32(x3) << 21
	if x3 < 0x80 {
		d.index += 4
		return x - (1<<7 + 1<<14 + 1<<21)
	}
	x4 := buf[i+4]
	x += uint32(x4) << 28
	d.index += 5
	return x - (1<<7 + 1<<14 + 1<<21 + 1<<28)
}

func (d *Decoder) decLength() int    { return int(d.decUint32()) }
func (d *Decoder) decIsNotNil() bool { return d.decBool() }

func decIgnore(*Decoder, unsafe.Pointer)        {}
func decBool(d *Decoder, p unsafe.Pointer)      { *(*bool)(p) = d.decBool() }
func decInt(d *Decoder, p unsafe.Pointer)       { *(*int)(p) = int(uint64ToInt64(d.decUint64())) }
func decInt8(d *Decoder, p unsafe.Pointer)      { *(*int8)(p) = int8(d.buf[d.index]); d.index++ }
func decInt16(d *Decoder, p unsafe.Pointer)     { *(*int16)(p) = uint16ToInt16(d.decUint16()) }
func decInt32(d *Decoder, p unsafe.Pointer)     { *(*int32)(p) = uint32ToInt32(d.decUint32()) }
func decInt64(d *Decoder, p unsafe.Pointer)     { *(*int64)(p) = uint64ToInt64(d.decUint64()) }
func decUint(d *Decoder, p unsafe.Pointer)      { *(*uint)(p) = uint(d.decUint64()) }
func decUint8(d *Decoder, p unsafe.Pointer)     { *(*uint8)(p) = d.buf[d.index]; d.index++ }
func decUint16(d *Decoder, p unsafe.Pointer)    { *(*uint16)(p) = d.decUint16() }
func decUint32(d *Decoder, p unsafe.Pointer)    { *(*uint32)(p) = d.decUint32() }
func decUint64(d *Decoder, p unsafe.Pointer)    { *(*uint64)(p) = d.decUint64() }
func decUintptr(d *Decoder, p unsafe.Pointer)   { *(*uintptr)(p) = uintptr(d.decUint64()) }
func decPointer(d *Decoder, p unsafe.Pointer)   { *(*uintptr)(p) = uintptr(d.decUint64()) }
func decFloat32(d *Decoder, p unsafe.Pointer)   { *(*float32)(p) = uint32ToFloat32(d.decUint32()) }
func decFloat64(d *Decoder, p unsafe.Pointer)   { *(*float64)(p) = uint64ToFloat64(d.decUint64()) }
func decTime(d *Decoder, p unsafe.Pointer)      { *(*time.Time)(p) = time.Unix(0, int64(d.decUint64())) }
func decComplex64(d *Decoder, p unsafe.Pointer) { *(*uint64)(p) = d.decUint64() }
func decComplex128(d *Decoder, p unsafe.Pointer) {
	*(*uint64)(p) = d.decUint64()
	*(*uint64)(unsafe.Pointer(uintptr(p) + ptr1Size)) = d.decUint64()
}

func decString(d *Decoder, p unsafe.Pointer) {
	l, val := int(d.decUint32()), (*string)(p)
	*val = string(d.buf[d.index : d.index+l])
	d.index += l
}

func decBytes(d *Decoder, p unsafe.Pointer) {
	bytes := (*[]byte)(p)
	if d.decIsNotNil() {
		l := int(d.decUint32())
		*bytes = d.buf[d.index : d.index+l]
		d.index += l
	} else if !isNil(p) {
		*bytes = nil
	}
}

func skipUint64(d *Decoder, p unsafe.Pointer)     { d.decUint64() }
func skipUint32(d *Decoder, p unsafe.Pointer)     { d.decUint32() }
func skipUint16(d *Decoder, p unsafe.Pointer)     { d.decUint16() }
func skipByte(d *Decoder, p unsafe.Pointer)       { d.index++ }
func skipComplex128(d *Decoder, p unsafe.Pointer) { d.decUint64(); d.decUint64() }
func skipBytes(d *Decoder, p unsafe.Pointer) {
	if d.decIsNotNil() {
		l := int(d.decUint32())
		d.index += l
	}
}
func skipPanic(d *Decoder, p unsafe.Pointer) {
	panic("this type is unsupported")
}
