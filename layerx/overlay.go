package layerx

import "sync"

// Overlay maps private ctor ids.
type Overlay struct {
	mu sync.RWMutex
	m  map[uint32]Sym
}

// NewOverlay creates empty overlay.
func NewOverlay() *Overlay {
	return &Overlay{m: map[uint32]Sym{}}
}

// Map binds private id to sym.
func (o *Overlay) Map(id uint32, s Sym) error {
	if !SymValid(s) {
		return ErrNoPlan
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.m[id] = s
	return nil
}

// Sym resolves private id.
func (o *Overlay) Sym(id uint32) (Sym, bool) {
	if o == nil {
		return 0, false
	}
	o.mu.RLock()
	defer o.mu.RUnlock()
	s, ok := o.m[id]
	return s, ok
}

// SetOverlay attaches overlay.
func (t *Table) SetOverlay(o *Overlay) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ov = o
}

// overlay returns attached overlay.
func (t *Table) overlay() *Overlay {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.ov
}

// symOf resolves id with overlay.
func (t *Table) symOf(l Layer, id uint32) (Sym, bool) {
	if s, ok := SymOf(l, id); ok {
		return s, true
	}
	return t.overlay().Sym(id)
}
