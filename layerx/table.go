package layerx

import (
	"context"
	"sync"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/mt"
)

// Out is a method result.
type Out = bin.Object

// Handler serves one sym.
type Handler func(ctx context.Context, m *Mark) (Out, error)

// Wrap is envelope info.
type Wrap struct {
	Sym   Sym
	Layer Layer
	Meta  bin.Object
}

// OddView describes an unknown method.
type OddView struct {
	Layer Layer
	Ctor  uint32
	Sym   Sym
	Known bool
	Priv  bool
	Name  string
}

// Table routes calls by sym.
type Table struct {
	mu     sync.RWMutex
	h      map[Sym]Handler
	onWrap func(Wrap)
	onOdd  func(OddView)
	ov     *Overlay
}

// NewTable creates empty table.
func NewTable() *Table {
	return &Table{h: map[Sym]Handler{}}
}

// Add registers handler.
func (t *Table) Add(s Sym, h Handler) error {
	if h == nil {
		return ErrNil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.h[s]; ok {
		return ErrAbsent
	}
	t.h[s] = h
	return nil
}

// Has reports registered handler.
func (t *Table) Has(s Sym) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.h[s]
	return ok
}

// OnWrap sets envelope hook.
func (t *Table) OnWrap(f func(Wrap)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onWrap = f
}

// OnOdd sets unknown hook.
func (t *Table) OnOdd(f func(OddView)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onOdd = f
}

// Take decodes a request for layer.
func (t *Table) Take(l Layer, body *bin.Buffer) (*Taken, error) {
	return t.admit(l, body, Opts{}, false)
}

// TakeDef takes with default layer.
func (t *Table) TakeDef(l Layer, body *bin.Buffer) (*Taken, error) {
	return t.Take(l, body)
}

// TakeAny takes using canonical layer.
func (t *Table) TakeAny(body *bin.Buffer) (*Taken, error) {
	return t.Take(Canon, body)
}

// TakeOpt takes with preflight.
func (t *Table) TakeOpt(l Layer, body *bin.Buffer, o Opts) (*Taken, error) {
	return t.admit(l, body, o, true)
}

// TakeDefOpt takes with default layer and preflight.
func (t *Table) TakeDefOpt(l Layer, body *bin.Buffer, o Opts) (*Taken, error) {
	return t.admit(l, body, o, true)
}

// TakeAnyOpt takes with canonical layer and preflight.
func (t *Table) TakeAnyOpt(body *bin.Buffer, o Opts) (*Taken, error) {
	return t.admit(Canon, body, o, true)
}

// admit unwraps and decodes a call.
func (t *Table) admit(l Layer, body *bin.Buffer, o Opts, check bool) (*Taken, error) {
	if !Known(l) {
		return nil, ErrLayer
	}
	body, err := expand(body, o)
	if err != nil {
		return nil, err
	}
	eff, wraps, fixed, err := t.unwrap(body, l)
	if err != nil {
		return nil, err
	}
	if !Known(eff) {
		return nil, ErrLayer
	}
	if check {
		tmp := bin.Buffer{Buf: body.Buf}
		if err := Check(&tmp, eff, o.Lim); err != nil {
			return nil, err
		}
		if len(tmp.Buf) != 0 {
			return nil, ErrLimit
		}
	}
	id, err := body.Uint32()
	if err != nil {
		return nil, err
	}
	s, ok := t.symOf(eff, id)
	if !ok {
		o := OddView{Layer: eff, Ctor: id}
		if n, ok := PrivSkip(id); ok {
			o.Priv = true
			o.Name = n
		}
		return nil, t.odd(o)
	}
	if !t.Has(s) {
		return nil, t.odd(OddView{Layer: eff, Ctor: id, Sym: s, Known: true})
	}
	v, err := getAt(body, eff, s)
	if err != nil {
		return nil, err
	}
	// Per-layer decoders keep the raw flag word local (bit meanings differ
	// between layers), so the canonical Flags bitmap would stay zero and every
	// Flags.Has/Get* accessor in a handler would lie. Rebuild it from the
	// decoded field values, exactly like getObj does for nested objects.
	if f, ok := v.(interface{ SetFlags() }); ok {
		f.SetFlags()
	}
	a := &Taken{
		m:     &Mark{layer: eff, sym: s, ctor: id, fixed: fixed, obj: v},
		wraps: wraps,
	}
	if f := t.wrapHook(); f != nil {
		for _, w := range wraps {
			f(w)
		}
	}
	return a, nil
}

func (t *Table) wrapHook() func(Wrap) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.onWrap
}

// expand unwraps gzip.
func expand(body *bin.Buffer, o Opts) (*bin.Buffer, error) {
	id, err := body.PeekID()
	if err != nil {
		return nil, err
	}
	if id != mt.GzipPackedTypeID {
		return body, nil
	}
	if o.Unzip == nil {
		return nil, ErrNoZip
	}
	body.Uint32()
	data, err := body.Bytes()
	if err != nil {
		return nil, err
	}
	raw, err := o.Unzip(data)
	if err != nil {
		return nil, err
	}
	if len(raw) > o.Lim.Default().Bytes {
		return nil, ErrLimit
	}
	return &bin.Buffer{Buf: raw}, nil
}

// unwrap strips envelopes.
func (t *Table) unwrap(body *bin.Buffer, l Layer) (Layer, []Wrap, bool, error) {
	eff := l
	fixed := false
	var wraps []Wrap
	for i := 0; i < wrapMax; i++ {
		id, err := body.PeekID()
		if err != nil {
			return eff, wraps, fixed, err
		}
		s, ok := t.symOf(l, id)
		if !ok {
			// Unknown terminal, handled by admit.
			return eff, wraps, fixed, nil
		}
		w := wrapOf(s)
		if w < 0 {
			return eff, wraps, fixed, nil
		}
		body.Skip(bin.Word)
		lay, has, err := wrapSkip(w, body, l)
		if err != nil {
			return eff, wraps, fixed, err
		}
		if has {
			eff = Layer(lay)
			fixed = true
		}
		wraps = append(wraps, Wrap{Sym: s, Layer: eff})
	}
	return eff, wraps, fixed, ErrLimit
}

func (t *Table) odd(v OddView) error {
	t.mu.RLock()
	f := t.onOdd
	t.mu.RUnlock()
	if f != nil {
		f(v)
	}
	return ErrOdd
}

// Serve invokes the handler.
func (t *Table) Serve(ctx context.Context, a *Taken) (Out, error) {
	if a == nil || a.m == nil {
		return nil, ErrNil
	}
	t.mu.RLock()
	h, ok := t.h[a.m.sym]
	t.mu.RUnlock()
	if !ok {
		return nil, ErrOdd
	}
	return h(ctx, a.m)
}

// Mark is immutable call identity.
type Mark struct {
	layer Layer
	sym   Sym
	ctor  uint32
	fixed bool
	obj   bin.Object
}

// Layer returns call layer.
func (m *Mark) Layer() Layer { return m.layer }

// Sym returns call sym.
func (m *Mark) Sym() Sym { return m.sym }

// Ctor returns wire id.
func (m *Mark) Ctor() uint32 { return m.ctor }

// Fixed reports explicit layer.
func (m *Mark) Fixed() bool { return m.fixed }

// Obj returns decoded request.
func (m *Mark) Obj() bin.Object { return m.obj }

// PutRes encodes result for call layer.
func (m *Mark) PutRes(b *bin.Buffer, v bin.Object) error {
	if v == nil {
		return ErrNil
	}
	t, ok := v.(typeIDer)
	if !ok {
		return ErrNoPlan
	}
	if !resAny(m.sym) && !resOK(m.sym, t.TypeID()) {
		return ErrMismatch
	}
	return Put(b, m.layer, v)
}

// PutResVec encodes vector result.
func (m *Mark) PutResVec(b *bin.Buffer, items []bin.Object) error {
	b.PutVectorHeader(len(items))
	for _, v := range items {
		if err := m.PutRes(b, v); err != nil {
			return err
		}
	}
	return nil
}

// Taken is an admitted call.
type Taken struct {
	m     *Mark
	wraps []Wrap
}

// Mark returns call identity.
func (t *Taken) Mark() *Mark { return t.m }

// Layer returns call layer.
func (t *Taken) Layer() Layer { return t.m.layer }

// Proof reports explicit evidence.
func (t *Taken) Proof() bool { return t.m.fixed }

// Prep reports preflight ran.
func (t *Taken) Prep() bool { return false }

// Wraps returns envelope chain.
func (t *Taken) Wraps() []Wrap { return t.wraps }

// Wrap returns envelope by index.
func (t *Taken) Wrap(i int) (Wrap, bool) {
	if i < 0 || i >= len(t.wraps) {
		return Wrap{}, false
	}
	return t.wraps[i], true
}
