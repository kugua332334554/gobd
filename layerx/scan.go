package layerx

import (
	"fmt"

	"github.com/gotd/td/bin"
)

// Scan kinds.
const (
	scPrim = iota
	scFlags
	scObj
	scVecPrim
	scVecObj
	scVecVecPrim
	scVecVecObj
)

// Prim kinds.
const (
	pNone = iota
	pInt
	pLong
	pInt53
	pInt128
	pInt256
	pDouble
	pString
	pBytes
	pBool
)

// scanField is one wire slot.
type scanField struct {
	kind uint8
	prim uint8
	bit  int8
	word uint8
	sym  uint32
}

// scanState tracks bounds.
type scanState struct {
	lim   Lim
	agg   int
	depth int
}

// Check validates wire without allocating.
func Check(b *bin.Buffer, l Layer, lim Lim) error {
	lim = lim.Default()
	if len(b.Buf) > lim.Bytes {
		return ErrLimit
	}
	st := &scanState{lim: lim}
	return scanObj(b, l, st)
}

// scanObj scans one object.
func scanObj(b *bin.Buffer, l Layer, st *scanState) error {
	if st.depth >= st.lim.Depth {
		return ErrLimit
	}
	id, err := b.Uint32()
	if err != nil {
		return err
	}
	s, ok := SymOf(l, id)
	if !ok {
		// Private alias ids have no scan table; consume them with the real
		// decoder so preflight stays accurate for DrKLO-only ctors.
		ps, pok := PrivSym(id)
		if !pok {
			return fmt.Errorf("%w: id %08x", ErrNoPlan, id)
		}
		_, err := getAt(b, l, ps)
		return err
	}
	return scanSym(b, l, st, s)
}

// scanSym scans one body.
func scanSym(b *bin.Buffer, l Layer, st *scanState, s Sym) error {
	// bridge types have no generated scan table; decode to consume them.
	if s == SymReplyInlineMarkup {
		st.depth++
		defer func() { st.depth-- }()
		_, err := getMarkupBridge(b, l)
		return err
	}
	fields := scanFor(l, s)
	if fields == nil {
		return fmt.Errorf("%w: %s", ErrNoPlan, SymName(s))
	}
	st.depth++
	defer func() { st.depth-- }()
	var fl [2]uint32
	for _, f := range fields {
		st.agg++
		if st.agg > st.lim.Agg {
			return ErrLimit
		}
		if f.bit >= 0 {
			if f.word >= 2 {
				return ErrLimit
			}
			if fl[f.word]&(1<<uint(f.bit)) == 0 {
				continue
			}
		}
		switch f.kind {
		case scFlags:
			if f.word >= 2 {
				return ErrLimit
			}
			v, err := b.Uint32()
			if err != nil {
				return err
			}
			fl[f.word] = v
		case scPrim:
			if err := scanPrim(b, f.prim); err != nil {
				return err
			}
		case scObj:
			if err := scanObj(b, l, st); err != nil {
				return err
			}
		case scVecPrim:
			n, err := scanHead(b, st)
			if err != nil {
				return err
			}
			for i := 0; i < n; i++ {
				if err := scanPrim(b, f.prim); err != nil {
					return err
				}
			}
		case scVecObj:
			n, err := scanHead(b, st)
			if err != nil {
				return err
			}
			for i := 0; i < n; i++ {
				if err := scanObj(b, l, st); err != nil {
					return err
				}
			}
		case scVecVecPrim:
			n, err := scanHead(b, st)
			if err != nil {
				return err
			}
			for i := 0; i < n; i++ {
				m, err := scanHead(b, st)
				if err != nil {
					return err
				}
				for j := 0; j < m; j++ {
					if err := scanPrim(b, f.prim); err != nil {
						return err
					}
				}
			}
		case scVecVecObj:
			n, err := scanHead(b, st)
			if err != nil {
				return err
			}
			for i := 0; i < n; i++ {
				m, err := scanHead(b, st)
				if err != nil {
					return err
				}
				for j := 0; j < m; j++ {
					if err := scanObj(b, l, st); err != nil {
						return err
					}
				}
			}
		default:
			return ErrNoPlan
		}
	}
	return nil
}

// scanHead reads a vector header.
func scanHead(b *bin.Buffer, st *scanState) (int, error) {
	n, err := b.VectorHeader()
	if err != nil {
		return 0, err
	}
	if n < 0 || n > st.lim.Vec {
		return 0, ErrLimit
	}
	if n > len(b.Buf) {
		return 0, ErrLimit
	}
	st.agg += n
	if st.agg > st.lim.Agg {
		return 0, ErrLimit
	}
	return n, nil
}

// scanPrim skips a primitive.
func scanPrim(b *bin.Buffer, p uint8) error {
	var err error
	switch p {
	case pInt:
		_, err = b.Int32()
	case pLong:
		_, err = b.Long()
	case pInt53:
		_, err = b.Int53()
	case pInt128:
		_, err = b.Int128()
	case pInt256:
		_, err = b.Int256()
	case pDouble:
		_, err = b.Double()
	case pString:
		_, err = b.String()
	case pBytes:
		_, err = b.Bytes()
	case pBool:
		_, err = b.Bool()
	default:
		return ErrNoPlan
	}
	return err
}
