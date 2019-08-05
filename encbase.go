package gotiny

import (
	"reflect"
	"time"
	"unsafe"
)

type encEng func(*Encoder, unsafe.Pointer) // Encoder

var (
	encEngines = [...]encEng{
		reflect.Invalid:       encIgnore,
		reflect.Bool:          encBool,
		reflect.Int:           encInt,
		reflect.Int8:          encInt8,
		reflect.Int16:         encInt16,
		reflect.Int32:         encInt32,
		reflect.Int64:         encInt64,
		reflect.Uint:          encUint,
		reflect.Uint8:         encUint8,
		reflect.Uint16:        encUint16,
		reflect.Uint32:        encUint32,
		reflect.Uint64:        encUint64,
		reflect.Uintptr:       encUintptr,
		reflect.UnsafePointer: encPointer,
		reflect.Float32:       encFloat32,
		reflect.Float64:       encFloat64,
		reflect.Complex64:     encComplex64,
		reflect.Complex128:    encComplex128,
		reflect.String:        encString,
	}
)

func (e *Encoder) encBool(v bool) {
	if e.boolBit == 0 {
		e.boolPos = len(e.buf)
		e.buf = append(e.buf, 0)
		e.boolBit = 1
	}
	if v {
		e.buf[e.boolPos] |= e.boolBit
	}
	e.boolBit <<= 1
}

func (e *Encoder) encUint64(v uint64) {
	switch {
	case v < 1<<7-1:
		e.buf = append(e.buf, byte(v))
	case v < 1<<14-1:
		e.buf = append(e.buf, byte(v)|0x80, byte(v>>7))
	case v < 1<<21-1:
		e.buf = append(e.buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14))
	case v < 1<<28-1:
		e.buf = append(e.buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14)|0x80, byte(v>>21))
	case v < 1<<35-1:
		e.buf = append(e.buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14)|0x80, byte(v>>21)|0x80, byte(v>>28))
	case v < 1<<42-1:
		e.buf = append(e.buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14)|0x80, byte(v>>21)|0x80, byte(v>>28)|0x80, byte(v>>35))
	case v < 1<<49-1:
		e.buf = append(e.buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14)|0x80, byte(v>>21)|0x80, byte(v>>28)|0x80, byte(v>>35)|0x80, byte(v>>42))
	case v < 1<<56-1:
		e.buf = append(e.buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14)|0x80, byte(v>>21)|0x80, byte(v>>28)|0x80, byte(v>>35)|0x80, byte(v>>42)|0x80, byte(v>>49))
	default:
		e.buf = append(e.buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14)|0x80, byte(v>>21)|0x80, byte(v>>28)|0x80, byte(v>>35)|0x80, byte(v>>42)|0x80, byte(v>>49)|0x80, byte(v>>56))
	}
}

func (e *Encoder) encUint16(v uint16) {
	if v < 1<<7-1 {
		e.buf = append(e.buf, byte(v))
	} else if v < 1<<14-1 {
		e.buf = append(e.buf, byte(v)|0x80, byte(v>>7))
	} else {
		e.buf = append(e.buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14))
	}
}

func (e *Encoder) encUint32(v uint32) {
	switch {
	case v < 1<<7-1:
		e.buf = append(e.buf, byte(v))
	case v < 1<<14-1:
		e.buf = append(e.buf, byte(v)|0x80, byte(v>>7))
	case v < 1<<21-1:
		e.buf = append(e.buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14))
	case v < 1<<28-1:
		e.buf = append(e.buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14)|0x80, byte(v>>21))
	default:
		e.buf = append(e.buf, byte(v)|0x80, byte(v>>7)|0x80, byte(v>>14)|0x80, byte(v>>21)|0x80, byte(v>>28))
	}
}

func (e *Encoder) encLength(v int)    { e.encUint32(uint32(v)) }
func (e *Encoder) encString(s string) { e.encUint32(uint32(len(s))); e.buf = append(e.buf, s...) }
func (e *Encoder) encIsNotNil(v bool) { e.encBool(v) }

func encIgnore(*Encoder, unsafe.Pointer)      {}
func encBool(e *Encoder, p unsafe.Pointer)    { e.encBool(*(*bool)(p)) }
func encInt(e *Encoder, p unsafe.Pointer)     { e.encUint64(int64ToUint64(int64(*(*int)(p)))) }
func encInt8(e *Encoder, p unsafe.Pointer)    { e.buf = append(e.buf, *(*uint8)(p)) }
func encInt16(e *Encoder, p unsafe.Pointer)   { e.encUint16(int16ToUint16(*(*int16)(p))) }
func encInt32(e *Encoder, p unsafe.Pointer)   { e.encUint32(int32ToUint32(*(*int32)(p))) }
func encInt64(e *Encoder, p unsafe.Pointer)   { e.encUint64(int64ToUint64(*(*int64)(p))) }
func encUint8(e *Encoder, p unsafe.Pointer)   { e.buf = append(e.buf, *(*uint8)(p)) }
func encUint16(e *Encoder, p unsafe.Pointer)  { e.encUint16(*(*uint16)(p)) }
func encUint32(e *Encoder, p unsafe.Pointer)  { e.encUint32(*(*uint32)(p)) }
func encUint64(e *Encoder, p unsafe.Pointer)  { e.encUint64(uint64(*(*uint64)(p))) }
func encUint(e *Encoder, p unsafe.Pointer)    { e.encUint64(uint64(*(*uint)(p))) }
func encUintptr(e *Encoder, p unsafe.Pointer) { e.encUint64(uint64(*(*uintptr)(p))) }
func encPointer(e *Encoder, p unsafe.Pointer) { e.encUint64(uint64(*(*uintptr)(p))) }
func encFloat32(e *Encoder, p unsafe.Pointer) { e.encUint32(float32ToUint32(p)) }
func encFloat64(e *Encoder, p unsafe.Pointer) { e.encUint64(float64ToUint64(p)) }
func encString(e *Encoder, p unsafe.Pointer) {
	s := *(*string)(p)
	e.encUint32(uint32(len(s)))
	e.buf = append(e.buf, s...)
}
func encTime(e *Encoder, p unsafe.Pointer)      { e.encUint64(uint64((*time.Time)(p).UnixNano())) }
func encComplex64(e *Encoder, p unsafe.Pointer) { e.encUint64(*(*uint64)(p)) }
func encComplex128(e *Encoder, p unsafe.Pointer) {
	e.encUint64(*(*uint64)(p))
	e.encUint64(*(*uint64)(unsafe.Pointer(uintptr(p) + ptr1Size)))
}

func encBytes(e *Encoder, p unsafe.Pointer) {
	isNotNil := !isNil(p)
	e.encIsNotNil(isNotNil)
	if isNotNil {
		buf := *(*[]byte)(p)
		e.encLength(len(buf))
		e.buf = append(e.buf, buf...)
	}
}
