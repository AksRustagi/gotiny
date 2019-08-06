package gotiny

import (
	"reflect"
	"sync"
	"time"
	"unsafe"
)

type gotinyType uint8

const (
	typeIgnore gotinyType = iota
	typeStruct
	typeSlice
	typeArray
	typeMap
	typeBool
	typeInt
	typeInt8
	typeInt16
	typeInt32
	typeInt64
	typeUint
	typeUint8
	typeUint16
	typeUint32
	typeUint64
	typeFloat32
	typeFloat64
	typeBytes
	typeTime
	typeInterface
	typePointer
	typeComplex64
	typeComplex128
	typeCustom // custom serialiser
)

var (
	rt2Node = map[reflect.Type]Scheme{
		reflect.TypeOf((*bool)(nil)).Elem():           Scheme{encodeEngine: encBool, decodeEngine: decBool, Type: typeBool},
		reflect.TypeOf((*int)(nil)).Elem():            Scheme{encodeEngine: encInt, decodeEngine: decInt, Type: typeInt},
		reflect.TypeOf((*int8)(nil)).Elem():           Scheme{encodeEngine: encInt8, decodeEngine: decInt8, Type: typeInt8},
		reflect.TypeOf((*int32)(nil)).Elem():          Scheme{encodeEngine: encInt32, decodeEngine: decInt32, Type: typeInt32},
		reflect.TypeOf((*int16)(nil)).Elem():          Scheme{encodeEngine: encInt16, decodeEngine: decInt16, Type: typeInt16},
		reflect.TypeOf((*int64)(nil)).Elem():          Scheme{encodeEngine: encInt64, decodeEngine: decInt64, Type: typeInt64},
		reflect.TypeOf((*uint)(nil)).Elem():           Scheme{encodeEngine: encUint, decodeEngine: decUint, Type: typeUint},
		reflect.TypeOf((*uint8)(nil)).Elem():          Scheme{encodeEngine: encUint8, decodeEngine: decUint8, Type: typeUint8},
		reflect.TypeOf((*uint16)(nil)).Elem():         Scheme{encodeEngine: encUint16, decodeEngine: decUint16, Type: typeUint16},
		reflect.TypeOf((*uint32)(nil)).Elem():         Scheme{encodeEngine: encUint32, decodeEngine: decUint32, Type: typeUint32},
		reflect.TypeOf((*uint64)(nil)).Elem():         Scheme{encodeEngine: encUint64, decodeEngine: decUint64, Type: typeUint64},
		reflect.TypeOf((*uintptr)(nil)).Elem():        Scheme{encodeEngine: encUintptr, decodeEngine: decUintptr, Type: typeUint64}, // encoded as int
		reflect.TypeOf((*unsafe.Pointer)(nil)).Elem(): Scheme{encodeEngine: encPointer, decodeEngine: decPointer, Type: typeUint64},
		reflect.TypeOf((*float32)(nil)).Elem():        Scheme{encodeEngine: encFloat32, decodeEngine: decFloat32, Type: typeFloat32},
		reflect.TypeOf((*float64)(nil)).Elem():        Scheme{encodeEngine: encFloat64, decodeEngine: decFloat64, Type: typeFloat64},
		reflect.TypeOf((*complex64)(nil)).Elem():      Scheme{encodeEngine: encComplex64, decodeEngine: decComplex64, Type: typeComplex64},
		reflect.TypeOf((*complex128)(nil)).Elem():     Scheme{encodeEngine: encComplex128, decodeEngine: decComplex128, Type: typeComplex128},
		reflect.TypeOf((*[]byte)(nil)).Elem():         Scheme{encodeEngine: encBytes, decodeEngine: decBytes, Type: typeBytes},
		reflect.TypeOf((*string)(nil)).Elem():         Scheme{encodeEngine: encString, decodeEngine: decString, Type: typeBytes},
		reflect.TypeOf((*time.Time)(nil)).Elem():      Scheme{encodeEngine: encTime, decodeEngine: decTime, Type: typeUint64},
		reflect.TypeOf((*struct{})(nil)).Elem():       Scheme{encodeEngine: encIgnore, decodeEngine: decIgnore, Type: typeIgnore},
		reflect.TypeOf(nil):                           Scheme{encodeEngine: encIgnore, decodeEngine: decIgnore, Type: typeIgnore},
	}
	rtLock sync.RWMutex

	type2Empty = map[gotinyType]decEng{
		typeIgnore:     func(d *Decoder, p unsafe.Pointer) {},
		typeBool:       func(d *Decoder, p unsafe.Pointer) { var v bool; decBool(d, unsafe.Pointer(&v)) },
		typeInt:        skipUint64,
		typeInt8:       skipByte,
		typeInt16:      skipUint16,
		typeInt32:      skipUint32,
		typeInt64:      skipUint64,
		typeUint:       skipUint64,
		typeUint8:      skipByte,
		typeUint16:     skipUint16,
		typeUint32:     skipUint32,
		typeUint64:     skipUint64,
		typeFloat32:    skipUint32,
		typeFloat64:    skipUint64,
		typeBytes:      skipBytes,
		typeTime:       skipUint64,
		typeInterface:  skipPanic,
		typePointer:    skipUint64,
		typeComplex64:  skipUint64,
		typeComplex128: skipComplex128,
	}
)

// Coder provides single thread safe interface to encode and decode objects
// for performance reasons encoders and decoders reusing using channel pool
type Coder struct {
	scheme         Scheme
	originalScheme Scheme
	encodeEngines  []encEng // optimisation param for faster creation of encoder and decoder
	decodeEngines  []decEng
	encoder        chan *Encoder // to reuse existing encoders
	decoder        chan *Decoder // to reuse existing decoders
	length         int
}

// CoderNew creates new scheme oblect
func CoderNew(l int) *Coder {
	return &Coder{
		length: l,
		scheme: Scheme{
			Childs: make([]*Scheme, l),
		},
		encodeEngines: make([]encEng, l),
		decodeEngines: make([]decEng, l),
		encoder:       make(chan *Encoder, 10),
		decoder:       make(chan *Decoder, 10),
	}
}

// New creates new scheme using passed list of objects
func New(is ...interface{}) *Coder {
	l := len(is)
	coder := CoderNew(l)
	for i := 0; i < l; i++ {
		coder.getEngine(i, reflect.TypeOf(is[i]))
	}
	return coder
}

// NewWithPtr creates an encoder that encodes the ps pointing type
func NewWithPtr(ps ...interface{}) *Coder {
	l := len(ps)
	coder := CoderNew(l)
	for i := 0; i < l; i++ {
		rt := reflect.TypeOf(ps[i])
		if rt.Kind() != reflect.Ptr {
			panic("must a pointer type!")
		}
		coder.getEngine(i, rt.Elem())
	}
	return coder
}

// NewWithType creates an scheme using type
func NewWithType(ts ...reflect.Type) *Coder {
	l := len(ts)
	coder := CoderNew(l)
	for i := 0; i < l; i++ {
		coder.getEngine(i, ts[i])
	}
	return coder
}

// GetScheme returns scheme of coder
func (c *Coder) GetScheme() *Scheme {
	return &c.scheme
}

// SetScheme will set scheme for the coder
// Note: scheme will be copied, changes after setting will not apply
func (c *Coder) SetScheme(scheme *Scheme) {
	c.encodeEngines = []encEng{}
	c.decodeEngines = []decEng{}

	if len(c.originalScheme.Childs) != len(scheme.Childs) {
		panic("setting scheme with different number of elements")
	}

	for i, child := range scheme.Childs {
		child.fillEngines(c, c.originalScheme.Childs[i])
		c.encodeEngines = append(c.encodeEngines, child.encodeEngine)
		c.decodeEngines = append(c.decodeEngines, child.decodeEngine)
	}
	c.scheme = *scheme
}

// GetEncoder creates encoder for scheme using cached data on scheme
func (c *Coder) GetEncoder() (enc *Encoder) {
	select {
	case enc = <-c.encoder:
	default:
		enc = &Encoder{
			length:  c.length,
			engines: c.encodeEngines,
		}
	}
	return
}

// PutEncoder put encoder back for reuse
func (c *Coder) PutEncoder(enc *Encoder) {
	select {
	case c.encoder <- enc:
	default:
	}
}

// GetDecoder creates decode for scheme using cached data on scheme
func (c *Coder) GetDecoder() (dec *Decoder) {
	select {
	case dec = <-c.decoder:
	default:
		dec = &Decoder{
			length:  c.length,
			engines: c.decodeEngines,
		}
	}
	return
}

// PutDecoder put decoder back for reuse
func (c *Coder) PutDecoder(dec *Decoder) {
	select {
	case c.decoder <- dec:
	default:
	}
}

// Encode object using entry parameter as a pointer to the value to be encoded
func (c *Coder) Encode(is ...interface{}) []byte {
	enc := c.GetEncoder()
	data := enc.Encode(is...)
	c.PutEncoder(enc)
	return data
}

// EncodePtr the input parameter is the unsafe.Pointer pointer
func (c *Coder) EncodePtr(ps ...unsafe.Pointer) []byte {
	enc := c.GetEncoder()
	data := enc.EncodePtr(ps...)
	c.PutEncoder(enc)
	return data
}

// EncodeValue the input parameter is the reflect.Value
func (c *Coder) EncodeValue(vs ...reflect.Value) []byte {
	enc := c.GetEncoder()
	data := enc.EncodeValue(vs...)
	c.PutEncoder(enc)
	return data
}

// Decode decodes data with new decoder
func (c *Coder) Decode(buf []byte, is ...interface{}) int {
	dec := c.GetDecoder()
	res := dec.Decode(buf, is...)
	c.PutDecoder(dec)
	return res
}

// DecodePtr decodes using ps as an unsafe.Pointer of the variable
func (c *Coder) DecodePtr(buf []byte, is ...unsafe.Pointer) int {
	dec := c.GetDecoder()
	res := dec.DecodePtr(buf, is...)
	c.PutDecoder(dec)
	return res
}

// DecodeValue decodes using vs as an reflect.Value
func (c *Coder) DecodeValue(buf []byte, vs ...reflect.Value) int {
	dec := c.GetDecoder()
	res := dec.DecodeValue(buf, vs...)
	c.PutDecoder(dec)
	return res
}

func (c *Coder) getEngine(index int, rt reflect.Type) {
	rtLock.RLock()
	node, ok := rt2Node[rt]
	rtLock.RUnlock()
	if !ok {
		rtLock.Lock()
		buildSchemeEngine("", rt, &node)
		//var scheme Scheme
		//buildScheme("", rt, &scheme)
		rtLock.Unlock()
	}
	c.scheme.Childs[index] = &node
	c.encodeEngines[index] = node.encodeEngine
	c.decodeEngines[index] = node.decodeEngine
	// keep original scheme to be able to apply multiple schemes on top
	c.originalScheme = c.scheme
}

func buildSchemeEngine(name string, rt reflect.Type, nodePtr *Scheme) {
	node, ok := rt2Node[rt]
	if ok {
		node.Name = name
		*nodePtr = node
		return
	}

	node = Scheme{Name: name}
	if encodeEngine, decodeEngine := implementOtherSerializer(rt); encodeEngine != nil {
		node.encodeEngine = encodeEngine
		node.decodeEngine = decodeEngine
		node.Type = typeCustom
		rt2Node[rt] = node // put node to cache
		*nodePtr = node
		return
	}

	kind := rt.Kind()
	switch kind {
	case reflect.Ptr:
		et := rt.Elem()
		var eNode Scheme
		node.encodeEngine = func(e *Encoder, p unsafe.Pointer) {
			isNotNil := !isNil(p)
			e.encIsNotNil(isNotNil)
			if isNotNil {
				eNode.encodeEngine(e, *(*unsafe.Pointer)(p))
			}
		}
		node.decodeEngine = func(d *Decoder, p unsafe.Pointer) {
			if d.decIsNotNil() {
				if isNil(p) {
					*(*unsafe.Pointer)(p) = unsafe.Pointer(reflect.New(et).Elem().UnsafeAddr())
				}
				eNode.decodeEngine(d, *(*unsafe.Pointer)(p))
			} else if !isNil(p) {
				*(*unsafe.Pointer)(p) = nil
			}
		}

		node.Type = typePointer
		node.Childs = []*Scheme{&eNode}
		rt2Node[rt] = node
		buildSchemeEngine("", et, &eNode)
	case reflect.Array:
		et, l := rt.Elem(), rt.Len()
		var eNode Scheme
		size := et.Size()
		node.encodeEngine = func(e *Encoder, p unsafe.Pointer) {
			for i := 0; i < l; i++ {
				eNode.encodeEngine(e, unsafe.Pointer(uintptr(p)+uintptr(i)*size))
			}
		}
		node.decodeEngine = func(d *Decoder, p unsafe.Pointer) {
			for i := 0; i < l; i++ {
				eNode.decodeEngine(d, unsafe.Pointer(uintptr(p)+uintptr(i)*size))
			}
		}

		node.Type = typeArray
		node.Childs = []*Scheme{&eNode}
		rt2Node[rt] = node
		buildSchemeEngine("", et, &eNode)
	case reflect.Slice:
		et := rt.Elem()
		size := et.Size()
		var eNode Scheme
		node.encodeEngine = func(e *Encoder, p unsafe.Pointer) {
			isNotNil := !isNil(p)
			e.encIsNotNil(isNotNil)
			if isNotNil {
				header := (*reflect.SliceHeader)(p)
				l := header.Len
				e.encLength(l)
				for i := 0; i < l; i++ {
					eNode.encodeEngine(e, unsafe.Pointer(header.Data+uintptr(i)*size))
				}
			}
		}
		node.decodeEngine = func(d *Decoder, p unsafe.Pointer) {
			header := (*reflect.SliceHeader)(p)
			if d.decIsNotNil() {
				l := d.decLength()
				if isNil(p) || header.Cap < l {
					*header = reflect.SliceHeader{Data: reflect.MakeSlice(rt, l, l).Pointer(), Len: l, Cap: l}
				} else {
					header.Len = l
				}
				for i := 0; i < l; i++ {
					eNode.decodeEngine(d, unsafe.Pointer(header.Data+uintptr(i)*size))
				}
			} else if !isNil(p) {
				*header = reflect.SliceHeader{}
			}
		}
		node.Type = typeSlice
		node.Childs = []*Scheme{&eNode}
		rt2Node[rt] = node
		buildSchemeEngine("", et, &eNode)
	case reflect.Map:
		var kNode, eNode Scheme
		kt, vt := rt.Key(), rt.Elem()
		skt, svt := reflect.SliceOf(kt), reflect.SliceOf(vt)
		node.encodeEngine = func(e *Encoder, p unsafe.Pointer) {
			isNotNil := !isNil(p)
			e.encIsNotNil(isNotNil)
			if isNotNil {
				v := reflect.NewAt(rt, p).Elem()
				e.encLength(v.Len())
				keys := v.MapKeys()
				for i := 0; i < len(keys); i++ {
					val := v.MapIndex(keys[i])
					kNode.encodeEngine(e, getUnsafePointer(&keys[i]))
					eNode.encodeEngine(e, getUnsafePointer(&val))
				}
			}
		}
		node.decodeEngine = func(d *Decoder, p unsafe.Pointer) {
			if d.decIsNotNil() {
				l := d.decLength()
				var v reflect.Value
				if isNil(p) {
					v = reflect.MakeMapWithSize(rt, l)
					*(*unsafe.Pointer)(p) = unsafe.Pointer(v.Pointer())
				} else {
					v = reflect.NewAt(rt, p).Elem()
				}
				keys, vals := reflect.MakeSlice(skt, l, l), reflect.MakeSlice(svt, l, l)
				for i := 0; i < l; i++ {
					key, val := keys.Index(i), vals.Index(i)
					kNode.decodeEngine(d, unsafe.Pointer(key.UnsafeAddr()))
					eNode.decodeEngine(d, unsafe.Pointer(val.UnsafeAddr()))
					v.SetMapIndex(key, val)
				}
			} else if !isNil(p) {
				*(*unsafe.Pointer)(p) = nil
			}
		}
		node.Type = typeMap
		node.Childs = []*Scheme{&kNode, &eNode}
		rt2Node[rt] = node
		buildSchemeEngine("", kt, &kNode)
		buildSchemeEngine("", vt, &eNode)
	case reflect.Struct:
		/*names, fields, offs := getFieldType(rt, 0)
		nf := len(fields)
		fNodes := make([]*Scheme, nf)
		node.encodeEngine = func(e *Encoder, p unsafe.Pointer) {
			for i := 0; i < len(fNodes) && i < len(offs); i++ {
				fmt.Println("enc offset", fNodes[i].Name, i, offs[i])
				fNodes[i].encodeEngine(e, unsafe.Pointer(uintptr(p)+offs[i]))
			}
		}
		node.decodeEngine = func(d *Decoder, p unsafe.Pointer) {
			for i := 0; i < len(fNodes) && i < len(offs); i++ {
				fmt.Println("dec offset", fNodes[i].Name, i, offs[i])
				fNodes[i].decodeEngine(d, unsafe.Pointer(uintptr(p)+offs[i]))
			}
		}
		node.Childs = fNodes
		node.Type = typeStruct
		rt2Node[rt] = node
		for i := 0; i < nf; i++ {
			fNodes[i] = &Scheme{}
			buildSchemeEngine(names[i], fields[i], fNodes[i])
		}*/

		names, fields, offs := getFieldType(rt, 0)
		nf := len(fields)
		fNodes := make([]*Scheme, nf)

		node.Childs = fNodes
		node.Type = typeStruct
		node.setStructEngines("initial")
		rt2Node[rt] = node
		for i := 0; i < nf; i++ {
			fNodes[i] = &Scheme{}
			buildSchemeEngine(names[i], fields[i], fNodes[i])
			fNodes[i].offset = offs[i]
		}

	case reflect.Interface:
		if rt.NumMethod() > 0 {
			node.encodeEngine = func(e *Encoder, p unsafe.Pointer) {
				isNotNil := !isNil(p)
				e.encIsNotNil(isNotNil)
				if isNotNil {
					v := reflect.ValueOf(*(*interface {
						M()
					})(p))
					et := v.Type()
					e.encString(getNameOfType(et))

					var interfaceNode Scheme
					buildSchemeEngine("", et, &interfaceNode)
					interfaceNode.encodeEngine(e, getUnsafePointer(&v))
				}
			}
		} else {
			node.encodeEngine = func(e *Encoder, p unsafe.Pointer) {
				isNotNil := !isNil(p)
				e.encIsNotNil(isNotNil)
				if isNotNil {
					v := reflect.ValueOf(*(*interface{})(p))
					et := v.Type()
					e.encString(getNameOfType(et))

					var interfaceNode Scheme
					buildSchemeEngine("", et, &interfaceNode)
					interfaceNode.encodeEngine(e, getUnsafePointer(&v))
				}
			}
		}
		node.decodeEngine = func(d *Decoder, p unsafe.Pointer) {
			if d.decIsNotNil() {
				name := ""
				decString(d, unsafe.Pointer(&name))
				et, has := name2type[name]
				if !has {
					panic("unknown typ:" + name)
				}
				v := reflect.NewAt(rt, p).Elem()
				var ev reflect.Value
				if v.IsNil() || v.Elem().Type() != et {
					ev = reflect.New(et).Elem()
				} else {
					ev = v.Elem()
				}
				var interfaceNode Scheme
				buildSchemeEngine("", et, &interfaceNode)
				interfaceNode.decodeEngine(d, getUnsafePointer(&ev))
				v.Set(ev)
			} else if !isNil(p) {
				*(*unsafe.Pointer)(p) = nil
			}
		}
		node.Type = typeInterface
		rt2Node[rt] = node
	case reflect.Chan, reflect.Func:
		panic("not support " + rt.String() + " type")
	default:
		node.encodeEngine = encEngines[kind]
		node.decodeEngine = decEngines[kind]
		rt2Node[rt] = node
	}
	*nodePtr = node
}

// UnusedUnixNanoEncodeTimeType removes unused time
func UnusedUnixNanoEncodeTimeType() {
	delete(rt2Node, reflect.TypeOf((*time.Time)(nil)).Elem())
	delete(rt2Node, reflect.TypeOf((*time.Time)(nil)).Elem())
}
