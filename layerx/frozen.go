package layerx

import "github.com/gotd/td/bin"

// Frozen is an immutable value.
type Frozen struct {
	sym  Sym
	body []byte
}

// Freeze captures canonical bytes.
func Freeze(v bin.Object) (*Frozen, error) {
	if v == nil {
		return nil, ErrNil
	}
	t, ok := v.(typeIDer)
	if !ok {
		return nil, ErrNoPlan
	}
	var b bin.Buffer
	if err := v.Encode(&b); err != nil {
		return nil, err
	}
	if len(b.Buf) < bin.Word {
		return nil, ErrStale
	}
	body := append([]byte(nil), b.Buf[bin.Word:]...)
	return &Frozen{sym: Sym(t.TypeID()), body: body}, nil
}

// Sym returns frozen sym.
func (f *Frozen) Sym() Sym { return f.sym }

// Write encodes value for layer.
func (f *Frozen) Write(b *bin.Buffer, l Layer) error {
	if !Known(l) {
		return ErrLayer
	}
	if !customAt(l, f.sym) {
		if err := putWithID(b, l, f.sym); err != nil {
			return err
		}
		b.Buf = append(b.Buf, f.body...)
		return nil
	}
	v := newSym(f.sym)
	if v == nil {
		return ErrNoPlan
	}
	d, ok := v.(bin.BareDecoder)
	if !ok {
		return ErrNoPlan
	}
	if err := d.DecodeBare(&bin.Buffer{Buf: f.body}); err != nil {
		return ErrStale
	}
	return putAt(b, l, f.sym, v)
}

// Size returns encoded size.
func (f *Frozen) Size(l Layer) (int, error) {
	var b bin.Buffer
	if err := f.Write(&b, l); err != nil {
		return 0, err
	}
	return len(b.Buf), nil
}
