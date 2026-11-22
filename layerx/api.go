// Package layerx is a multi-layer TL projection runtime.
package layerx

import (
	"fmt"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
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
	// bridge types without a generated plan, e.g. replyInlineMarkup.
	if s == SymReplyInlineMarkup && !customAt(l, s) {
		return putMarkupBridge(b, l, v)
	}
	// L228 reply keyboards: the button ctor changed shape in newer layers
	// (type:ButtonType), so the generated encoder cannot downgrade it.
	if s == SymReplyKeyboardMarkup && l == L228 {
		return putKeyboardMarkup228(b, v)
	}
	// Containers carrying update trees: members that are absent from the
	// target layer are dropped (or rejected) instead of failing the whole push.
	if s == SymUpdates || s == SymUpdatesCombined || s == SymUpdateShort {
		return putUpdatesProjected(b, l, s, v)
	}
	// Media and layout shapes introduced after a layer cannot be represented
	// there; substitute the documented fallback.
	if s == SymMessageMediaWidget && l < L233 {
		return putMessageMediaUnsupported(b, l)
	}
	if s == SymChannelsToggleForum && l < L233 {
		return putToggleForumProjected(b, l, v)
	}
	// Text payloads typed as TextWithEntities in canonical form are plain
	// strings in the earliest layers.
	if l <= L229 {
		switch s {
		case SymBotVerification:
			if x, ok := v.(*tg.BotVerification); ok {
				return putBotVerification229(b, l, x)
			}
		case SymBotVerifierSettings:
			if x, ok := v.(*tg.BotVerifierSettings); ok {
				return putBotVerifierSettings229(b, l, x)
			}
		case SymBotsSetCustomVerification:
			if x, ok := v.(*tg.BotsSetCustomVerificationRequest); ok {
				return putBotsSetCustomVerification229(b, l, x)
			}
		}
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
		if s, ok = PrivSym(id); !ok {
			return nil, fmt.Errorf("%w: id %08x layer %d", ErrNoPlan, id, l)
		}
	}
	// bridge types without a generated plan, e.g. replyInlineMarkup.
	if s == SymReplyInlineMarkup && !customAt(l, s) {
		return getMarkupBridge(b, l)
	}
	if s == SymReplyKeyboardMarkup && l == L228 {
		return getKeyboardMarkup228(b)
	}
	if l <= L229 {
		switch s {
		case SymBotVerification:
			return getBotVerification229(b)
		case SymBotVerifierSettings:
			return getBotVerifierSettings229(b)
		case SymBotsSetCustomVerification:
			return getBotsSetCustomVerification229(b, l)
		}
	}
	v, err := getAt(b, l, s)
	if err != nil {
		return nil, err
	}
	// Generated per-layer decoders keep the raw flag word local (bit meanings
	// differ between layers), so the canonical Flags bitmap would stay empty
	// and every Get*/Flags.Has accessor would lie. Rebuild it from the decoded
	// field values.
	if f, ok := v.(interface{ SetFlags() }); ok {
		f.SetFlags()
	}
	return v, nil
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
