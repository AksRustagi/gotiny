package gotiny

import (
	"encoding/json"
	"unsafe"
)

// Scheme is point of object scheme graph
type Scheme struct {
	Name         string `json:"name,omitempty"`
	encodeEngine encEng
	decodeEngine decEng
	Type         gotinyType `json:"type,omitempty"`
	Childs       []*Scheme  `json:"childs,omitempty"`
	offset       uintptr    // struct offset to fill object
}

// SchemeNew creates new scheme node
func SchemeNew(name string, encodeEngine encEng, decodeEngine decEng) Scheme {
	return Scheme{
		Name:         name,
		encodeEngine: encodeEngine,
		decodeEngine: decodeEngine,
	}
}

// SchemeFromJSON will return scheme created from json data
func SchemeFromJSON(data string) (*Scheme, error) {
	scheme := Scheme{}
	err := json.Unmarshal([]byte(data), &scheme)
	return &scheme, err
}

// AsJSON return scheme in format of json
func (s *Scheme) AsJSON() string {
	res, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(res)
}

func (s *Scheme) setStructEngines(place string) {
	childs := s.Childs
	s.encodeEngine = func(e *Encoder, p unsafe.Pointer) {
		for i := 0; i < len(childs); i++ {
			childs[i].encodeEngine(e, unsafe.Pointer(uintptr(p)+childs[i].offset))
		}
	}
	s.decodeEngine = func(d *Decoder, p unsafe.Pointer) {
		for i := 0; i < len(childs); i++ {
			//fmt.Println("decode child", childs[i].Name, "offset", childs[i].offset)
			childs[i].decodeEngine(d, unsafe.Pointer(uintptr(p)+childs[i].offset))
		}
	}
}

// should add engines wich skips on decode and writes empty value on encode
func (s *Scheme) setEmptyEngines() {
	if s.Type == typeStruct {
		s.setStructEngines("via empty engines")
	} else {
		s.decodeEngine = type2Empty[s.Type]
	}
	// overwrite encode engine with panicing
	s.encodeEngine = func(e *Encoder, p unsafe.Pointer) {
		panic("encodeing empty type from dinamic scheme not supported yet")
	}
}

// find scheme node child from inside current scheme (object representation)
func (s *Scheme) find(childToFind *Scheme) *Scheme {
	for _, child := range s.Childs {
		if child.Name == childToFind.Name { // return element with same child
			return child
		}
	}
	return nil
}

// prepare sets engines using main object scheme
func (s *Scheme) fillEngines(coder *Coder, originalScheme *Scheme) {
	if originalScheme == nil {
		for _, child := range s.Childs {
			child.fillEngines(coder, nil)
		}
		s.setEmptyEngines()
		return
	}

	for _, child := range s.Childs {
		originalChild := originalScheme.find(child)
		child.fillEngines(coder, originalChild)
	}

	if s.Type != originalScheme.Type {
		s.setEmptyEngines()
	} else if s.Type == typeStruct {
		s.setStructEngines("via prepare")
	} else {
		s.encodeEngine = originalScheme.encodeEngine
		s.decodeEngine = originalScheme.decodeEngine
	}
	s.offset = originalScheme.offset
}
