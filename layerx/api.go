// Package layerx is a multi-layer TL projection runtime.
package layerx

import (
	"fmt"

	"github.com/gotd/td/bin"
)

// Layer is TL layer number.
type Layer uint16

// Sym is canonical type id.
type Sym uint32

// putWithID writes layer ctor id.
func putWithID(b *bin.Buffer, l Layer, s Sym) error {
	id, ok := Ctor(l, s)
	if !ok {
		return fmt.Errorf("%w: %s in layer %d", ErrAbsent, SymName(s), l)
	}
	b.PutID(id)
	return nil
}

// typeIDer exposes canonical id.
type typeIDer interface {
	TypeID() uint32
}

// putObj writes one object.
func putObj(b *bin.Buffer, l Layer, v bin.Object) error {
	if v == nil {
		return ErrNil
	}
	if !Known(l) {
		return ErrLayer
	}
	t, ok := v.(typeIDer)
	if !ok {
		return ErrNoPlan
	}
	if s, ok := PrivSym(t.TypeID()); ok {
		return putAt(b, l, s, v)
	}
	s := Sym(t.TypeID())
	if !SymValid(s) {
		return fmt.Errorf("%w: id %08x", ErrNoPlan, uint32(t.TypeID()))
	}
	return putAt(b, l, s, v)
}

// getObj reads one object.
func getObj(b *bin.Buffer, l Layer) (bin.Object, error) {
	id, err := b.Uint32()
	if err != nil {
		return nil, err
	}
	s, ok := SymOf(l, id)
	if !ok {
		return nil, fmt.Errorf("%w: id %08x layer %d", ErrNoPlan, id, l)
	}
	return getAt(b, l, s)
}

// Put encodes v for layer.
func Put(b *bin.Buffer, l Layer, v bin.Object) error {
	return putObj(b, l, v)
}

// Get decodes one object for layer.
func Get(b *bin.Buffer, l Layer) (bin.Object, error) {
	if !Known(l) {
		return nil, ErrLayer
	}
	return getObj(b, l)
}

// PutVec appends vector-encoded items.
func PutVec(b *bin.Buffer, l Layer, items []bin.Object) error {
	b.PutVectorHeader(len(items))
	for _, v := range items {
		if err := putObj(b, l, v); err != nil {
			return err
		}
	}
	return nil
}

// GetVec decodes vector items.
func GetVec(b *bin.Buffer, l Layer) ([]bin.Object, error) {
	n, err := b.VectorHeader()
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, ErrLimit
	}
	out := make([]bin.Object, 0, n%bin.PreallocateLimit)
	for i := 0; i < n; i++ {
		v, err := getObj(b, l)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// skipPrim discards a primitive.
func skipPrim(b *bin.Buffer, name string) error {
	var err error
	switch name {
	case "int", "int32":
		_, err = b.Int32()
	case "int53":
		_, err = b.Int53()
	case "long", "int64":
		_, err = b.Long()
	case "int128":
		_, err = b.Int128()
	case "int256":
		_, err = b.Int256()
	case "double":
		_, err = b.Double()
	case "string":
		_, err = b.String()
	case "bytes":
		_, err = b.Bytes()
	case "Bool", "bool", "true", "false":
		_, err = b.Bool()
	default:
		return fmt.Errorf("%w: skip %s", ErrNoPlan, name)
	}
	return err
}
