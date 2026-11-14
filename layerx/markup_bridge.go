package layerx

import (
	"fmt"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

// Legacy layer 228 ctor ids.
//
// See _schema/layers/l228.tl: replyInlineMarkup#48a30254 at line 734,
// the keyboardButton* family at lines 710-729.
const (
	ctor228ReplyInlineMarkup uint32 = 0x48a30254
	ctor228KeyboardButtonRow uint32 = 0x77608b83

	ctor228KeyboardButton              uint32 = 0x7d170cff
	ctor228KeyboardButtonURL           uint32 = 0xd80c25ec
	ctor228KeyboardButtonCallback      uint32 = 0xe62bc960
	ctor228KeyboardButtonRequestPhone  uint32 = 0x417efd8f
	ctor228KeyboardButtonRequestGeo    uint32 = 0xaa40f94d
	ctor228KeyboardButtonSwitchInline  uint32 = 0x991399fc
	ctor228KeyboardButtonGame          uint32 = 0x89c590f9
	ctor228KeyboardButtonBuy           uint32 = 0x3fa53905
	ctor228KeyboardButtonURLAuth       uint32 = 0xf51006f9
	ctor228InputKeyboardButtonURLAuth  uint32 = 0x68013e72
	ctor228KeyboardButtonRequestPoll   uint32 = 0x7a11d782
	ctor228InputKeyboardButtonUser     uint32 = 0x7d5e07c7
	ctor228KeyboardButtonUserProfile   uint32 = 0xc0fd5d09
	ctor228KeyboardButtonWebView       uint32 = 0xe846b1a0
	ctor228KeyboardButtonSimpleWebView uint32 = 0xe15c4370
	ctor228KeyboardButtonRequestPeer   uint32 = 0x5b0f15f5
	ctor228InputKeyboardButtonReqPeer  uint32 = 0x02b78156
	ctor228KeyboardButtonCopy          uint32 = 0xbcc4af10
)

// styleBit is the flags bit of the optional style field.
const styleBit = 10

// putMarkupBridge encodes replyInlineMarkup for the given layer.
//
// L228 uses rows of KeyboardButtonRow, L229+ uses rows of
// KeyboardInlineButtonRow with an extra force_reply flag.
func putMarkupBridge(b *bin.Buffer, l Layer, v bin.Object) error {
	m, ok := v.(*tg.ReplyInlineMarkup)
	if !ok {
		return fmt.Errorf("%w: %T as replyInlineMarkup", ErrMismatch, v)
	}
	if m == nil {
		return ErrNil
	}
	if l == L228 {
		return putMarkup228(b, m)
	}
	return putMarkupInline(b, l, m)
}

// getMarkupBridge decodes replyInlineMarkup for the given layer.
func getMarkupBridge(b *bin.Buffer, l Layer) (bin.Object, error) {
	if l == L228 {
		m, err := getMarkup228(b)
		if err != nil {
			return nil, err
		}
		return m, nil
	}
	m, err := getMarkupInline(b, l)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// putMarkup228 encodes the legacy layout.
func putMarkup228(b *bin.Buffer, m *tg.ReplyInlineMarkup) error {
	b.PutID(ctor228ReplyInlineMarkup)
	// force_reply has no L228 counterpart, dropped on downgrade.
	b.PutVectorHeader(len(m.Rows))
	for i := range m.Rows {
		b.PutID(ctor228KeyboardButtonRow)
		row := &m.Rows[i]
		b.PutVectorHeader(len(row.Buttons))
		for j := range row.Buttons {
			if err := putButton228(b, &row.Buttons[j]); err != nil {
				return fmt.Errorf("encode replyInlineMarkup row %d button %d: %w", i, j, err)
			}
		}
	}
	return nil
}

// putButton228 writes one legacy KeyboardButton.
func putButton228(b *bin.Buffer, btn *tg.KeyboardInlineButton) error {
	if btn == nil {
		return ErrNil
	}
	if btn.Type == nil {
		return fmt.Errorf("%w: keyboardButton type is nil", ErrNil)
	}

	var flags bin.Fields
	styled := btn.Flags.Has(styleBit) || !btn.Style.Zero()
	if styled {
		flags.Set(styleBit)
	}

	var (
		ctor uint32
		tail func(*bin.Buffer) error
	)
	switch t := btn.Type.(type) {
	case *tg.InlineButtonTypeURL:
		ctor = ctor228KeyboardButtonURL
		tail = func(b *bin.Buffer) error {
			b.PutString(t.URL)
			return nil
		}
	case *tg.InlineButtonTypeURLAuth:
		ctor = ctor228KeyboardButtonURLAuth
		if t.FwdText != "" {
			flags.Set(0)
		}
		tail = func(b *bin.Buffer) error {
			if t.FwdText != "" {
				b.PutString(t.FwdText)
			}
			b.PutString(t.URL)
			b.PutInt(t.ButtonID)
			return nil
		}
	case *tg.InlineButtonTypeWebView:
		ctor = ctor228KeyboardButtonWebView
		tail = func(b *bin.Buffer) error {
			b.PutString(t.URL)
			return nil
		}
	case *tg.InlineButtonTypeCallback:
		ctor = ctor228KeyboardButtonCallback
		if t.RequiresPassword {
			flags.Set(0)
		}
		tail = func(b *bin.Buffer) error {
			b.PutBytes(t.Data)
			return nil
		}
	case *tg.InlineButtonTypeGame:
		ctor = ctor228KeyboardButtonGame
	case *tg.InlineButtonTypeBuy:
		ctor = ctor228KeyboardButtonBuy
	case *tg.InlineButtonTypeSwitchInline:
		ctor = ctor228KeyboardButtonSwitchInline
		if t.SamePeer {
			flags.Set(0)
		}
		if t.PeerTypes != nil {
			flags.Set(1)
		}
		tail = func(b *bin.Buffer) error {
			b.PutString(t.Query)
			if t.PeerTypes == nil {
				return nil
			}
			b.PutVectorHeader(len(t.PeerTypes))
			for i, p := range t.PeerTypes {
				if p == nil {
					return fmt.Errorf("%w: peer_types[%d] is nil", ErrNil, i)
				}
				if err := p.Encode(b); err != nil {
					return fmt.Errorf("encode peer_types[%d]: %w", i, err)
				}
			}
			return nil
		}
	case *tg.InlineButtonTypeUserProfile:
		ctor = ctor228KeyboardButtonUserProfile
		tail = func(b *bin.Buffer) error {
			b.PutLong(t.UserID)
			return nil
		}
	case *tg.InlineButtonTypeCopy:
		ctor = ctor228KeyboardButtonCopy
		tail = func(b *bin.Buffer) error {
			b.PutString(t.CopyText)
			return nil
		}
	case *tg.InlineButtonTypeDisabled:
		// L228 has no inline disabled type, degrade to plain button.
		ctor = ctor228KeyboardButton
	default:
		return fmt.Errorf("%w: inline button type %T in layer %d", ErrNoPlan, btn.Type, L228)
	}

	b.PutID(ctor)
	if err := flags.Encode(b); err != nil {
		return fmt.Errorf("encode flags: %w", err)
	}
	if styled {
		if err := btn.Style.Encode(b); err != nil {
			return fmt.Errorf("encode style: %w", err)
		}
	}
	b.PutString(btn.Text)
	if tail != nil {
		if err := tail(b); err != nil {
			return err
		}
	}
	return nil
}

// getMarkup228 decodes the legacy layout.
func getMarkup228(b *bin.Buffer) (*tg.ReplyInlineMarkup, error) {
	n, err := b.VectorHeader()
	if err != nil {
		return nil, fmt.Errorf("decode replyInlineMarkup rows: %w", err)
	}
	if err := vecBound(b, n); err != nil {
		return nil, err
	}

	m := &tg.ReplyInlineMarkup{}
	if n > 0 {
		m.Rows = make([]tg.KeyboardInlineButtonRow, 0, n%bin.PreallocateLimit)
	}
	for i := 0; i < n; i++ {
		if err := readID(b, ctor228KeyboardButtonRow, "keyboardButtonRow"); err != nil {
			return nil, err
		}
		k, err := b.VectorHeader()
		if err != nil {
			return nil, fmt.Errorf("decode keyboardButtonRow buttons: %w", err)
		}
		if err := vecBound(b, k); err != nil {
			return nil, err
		}
		row := tg.KeyboardInlineButtonRow{}
		if k > 0 {
			row.Buttons = make([]tg.KeyboardInlineButton, 0, k%bin.PreallocateLimit)
		}
		for j := 0; j < k; j++ {
			btn, err := getButton228(b, L228)
			if err != nil {
				return nil, fmt.Errorf("decode row %d button %d: %w", i, j, err)
			}
			row.Buttons = append(row.Buttons, btn)
		}
		m.Rows = append(m.Rows, row)
	}
	return m, nil
}

// getButton228 decodes one legacy KeyboardButton.
func getButton228(b *bin.Buffer, l Layer) (tg.KeyboardInlineButton, error) {
	var out tg.KeyboardInlineButton
	id, err := b.Uint32()
	if err != nil {
		return out, err
	}

	var flags bin.Fields
	if err := flags.Decode(b); err != nil {
		return out, fmt.Errorf("decode flags: %w", err)
	}
	out.Flags = flags
	if flags.Has(styleBit) {
		if err := out.Style.Decode(b); err != nil {
			return out, fmt.Errorf("decode style: %w", err)
		}
	}

	// keyboardButtonRequestPoll declares quiz before text.
	if id == ctor228KeyboardButtonRequestPoll && flags.Has(0) {
		if _, err := b.Bool(); err != nil {
			return out, fmt.Errorf("decode quiz: %w", err)
		}
	}

	out.Text, err = b.String()
	if err != nil {
		return out, fmt.Errorf("decode text: %w", err)
	}

	switch id {
	case ctor228KeyboardButton:
		// plain button is the legacy default, canonical sees it as disabled.
		out.Type = &tg.InlineButtonTypeDisabled{}
	case ctor228KeyboardButtonURL:
		v, err := b.String()
		if err != nil {
			return out, fmt.Errorf("decode url: %w", err)
		}
		out.Type = &tg.InlineButtonTypeURL{URL: v}
	case ctor228KeyboardButtonCallback:
		data, err := b.Bytes()
		if err != nil {
			return out, fmt.Errorf("decode data: %w", err)
		}
		cb := &tg.InlineButtonTypeCallback{
			RequiresPassword: flags.Has(0),
			Data:             data,
		}
		if cb.RequiresPassword {
			cb.Flags.Set(0)
		}
		out.Type = cb
	case ctor228KeyboardButtonRequestPhone, ctor228KeyboardButtonRequestGeo,
		ctor228KeyboardButtonRequestPoll, ctor228KeyboardButtonRequestPeer,
		ctor228InputKeyboardButtonReqPeer:
		// reply-only button types have no inline counterpart, use disabled.
		if id == ctor228KeyboardButtonRequestPeer || id == ctor228InputKeyboardButtonReqPeer {
			if err := skipTail228(b, l, "button_id", "peer_type", "max_quantity"); err != nil {
				return out, err
			}
		}
		out.Type = &tg.InlineButtonTypeDisabled{}
	case ctor228KeyboardButtonSwitchInline:
		query, err := b.String()
		if err != nil {
			return out, fmt.Errorf("decode query: %w", err)
		}
		sw := &tg.InlineButtonTypeSwitchInline{
			SamePeer: flags.Has(0),
			Query:    query,
		}
		if sw.SamePeer {
			sw.Flags.Set(0)
		}
		if flags.Has(1) {
			k, err := b.VectorHeader()
			if err != nil {
				return out, fmt.Errorf("decode peer_types: %w", err)
			}
			if err := vecBound(b, k); err != nil {
				return out, err
			}
			sw.PeerTypes = make([]tg.InlineQueryPeerTypeClass, 0, k%bin.PreallocateLimit)
			for i := 0; i < k; i++ {
				p, err := tg.DecodeInlineQueryPeerType(b)
				if err != nil {
					return out, fmt.Errorf("decode peer_types[%d]: %w", i, err)
				}
				sw.PeerTypes = append(sw.PeerTypes, p)
			}
			sw.Flags.Set(1)
		}
		out.Type = sw
	case ctor228KeyboardButtonGame:
		out.Type = &tg.InlineButtonTypeGame{}
	case ctor228KeyboardButtonBuy:
		out.Type = &tg.InlineButtonTypeBuy{}
	case ctor228KeyboardButtonURLAuth:
		auth := &tg.InlineButtonTypeURLAuth{}
		if flags.Has(0) {
			if auth.FwdText, err = b.String(); err != nil {
				return out, fmt.Errorf("decode fwd_text: %w", err)
			}
			auth.Flags.Set(0)
		}
		if auth.URL, err = b.String(); err != nil {
			return out, fmt.Errorf("decode url: %w", err)
		}
		if auth.ButtonID, err = b.Int(); err != nil {
			return out, fmt.Errorf("decode button_id: %w", err)
		}
		out.Type = auth
	case ctor228InputKeyboardButtonURLAuth:
		// input-only variant, bot and request_write_access are dropped.
		// request_write_access is ?true, so it carries no bytes.
		auth := &tg.InlineButtonTypeURLAuth{}
		if flags.Has(1) {
			if auth.FwdText, err = b.String(); err != nil {
				return out, fmt.Errorf("decode fwd_text: %w", err)
			}
			auth.Flags.Set(0)
		}
		if auth.URL, err = b.String(); err != nil {
			return out, fmt.Errorf("decode url: %w", err)
		}
		if err := skipObj(b, l); err != nil {
			return out, fmt.Errorf("decode bot: %w", err)
		}
		out.Type = auth
	case ctor228KeyboardButtonUserProfile:
		uid, err := b.Long()
		if err != nil {
			return out, fmt.Errorf("decode user_id: %w", err)
		}
		out.Type = &tg.InlineButtonTypeUserProfile{UserID: uid}
	case ctor228InputKeyboardButtonUser:
		// InputUser cannot be resolved to a long here, drop the button.
		if err := skipObj(b, l); err != nil {
			return out, fmt.Errorf("decode user_id: %w", err)
		}
		out.Type = &tg.InlineButtonTypeDisabled{}
	case ctor228KeyboardButtonWebView:
		v, err := b.String()
		if err != nil {
			return out, fmt.Errorf("decode url: %w", err)
		}
		out.Type = &tg.InlineButtonTypeWebView{URL: v}
	case ctor228KeyboardButtonSimpleWebView:
		// L233 has no simple webview, keep the url and drop the distinction.
		v, err := b.String()
		if err != nil {
			return out, fmt.Errorf("decode url: %w", err)
		}
		out.Type = &tg.InlineButtonTypeWebView{URL: v}
	case ctor228KeyboardButtonCopy:
		v, err := b.String()
		if err != nil {
			return out, fmt.Errorf("decode copy_text: %w", err)
		}
		out.Type = &tg.InlineButtonTypeCopy{CopyText: v}
	default:
		return out, fmt.Errorf("%w: keyboardButton id %08x in layer %d", ErrNoPlan, id, l)
	}
	return out, nil
}

// skipTail228 skips a request_peer tail, currently only used while decoding
// button types that get degraded to disabled.
func skipTail228(b *bin.Buffer, l Layer, what ...string) error {
	for _, name := range what {
		switch name {
		case "button_id", "max_quantity":
			if _, err := b.Int(); err != nil {
				return fmt.Errorf("decode %s: %w", name, err)
			}
		default:
			if err := skipObj(b, l); err != nil {
				return fmt.Errorf("decode %s: %w", name, err)
			}
		}
	}
	return nil
}

// putMarkupInline encodes the L229-L233 layout.
func putMarkupInline(b *bin.Buffer, l Layer, m *tg.ReplyInlineMarkup) error {
	if err := putWithID(b, l, SymReplyInlineMarkup); err != nil {
		return err
	}
	var flags bin.Fields
	if m.ForceReply || m.Flags.Has(5) {
		flags.Set(5)
	}
	if err := flags.Encode(b); err != nil {
		return fmt.Errorf("encode flags: %w", err)
	}
	b.PutVectorHeader(len(m.Rows))
	for i := range m.Rows {
		if err := putWithID(b, l, SymKeyboardInlineButtonRow); err != nil {
			return err
		}
		row := &m.Rows[i]
		b.PutVectorHeader(len(row.Buttons))
		for j := range row.Buttons {
			if err := putInlineButton(b, l, &row.Buttons[j]); err != nil {
				return fmt.Errorf("encode replyInlineMarkup row %d button %d: %w", i, j, err)
			}
		}
	}
	return nil
}

// putInlineButton writes one KeyboardInlineButton.
func putInlineButton(b *bin.Buffer, l Layer, btn *tg.KeyboardInlineButton) error {
	if btn == nil {
		return ErrNil
	}
	if btn.Type == nil {
		return fmt.Errorf("%w: keyboardInlineButton type is nil", ErrNil)
	}
	if err := putWithID(b, l, SymKeyboardInlineButton); err != nil {
		return err
	}
	var flags bin.Fields
	styled := btn.Flags.Has(styleBit) || !btn.Style.Zero()
	if styled {
		flags.Set(styleBit)
	}
	if err := flags.Encode(b); err != nil {
		return fmt.Errorf("encode flags: %w", err)
	}
	if styled {
		if err := btn.Style.Encode(b); err != nil {
			return fmt.Errorf("encode style: %w", err)
		}
	}
	b.PutString(btn.Text)
	if err := btn.Type.Encode(b); err != nil {
		return fmt.Errorf("encode inline button type: %w", err)
	}
	return nil
}

// getMarkupInline decodes the L229-L233 layout.
func getMarkupInline(b *bin.Buffer, l Layer) (*tg.ReplyInlineMarkup, error) {
	var flags bin.Fields
	if err := flags.Decode(b); err != nil {
		return nil, fmt.Errorf("decode replyInlineMarkup flags: %w", err)
	}
	m := &tg.ReplyInlineMarkup{
		Flags:      flags,
		ForceReply: flags.Has(5),
	}
	n, err := b.VectorHeader()
	if err != nil {
		return nil, fmt.Errorf("decode replyInlineMarkup rows: %w", err)
	}
	if err := vecBound(b, n); err != nil {
		return nil, err
	}
	if n > 0 {
		m.Rows = make([]tg.KeyboardInlineButtonRow, 0, n%bin.PreallocateLimit)
	}
	for i := 0; i < n; i++ {
		id, ok := Ctor(l, SymKeyboardInlineButtonRow)
		if !ok {
			return nil, fmt.Errorf("%w: %s in layer %d", ErrAbsent, SymName(SymKeyboardInlineButtonRow), l)
		}
		if err := readID(b, id, "keyboardInlineButtonRow"); err != nil {
			return nil, err
		}
		k, err := b.VectorHeader()
		if err != nil {
			return nil, fmt.Errorf("decode keyboardInlineButtonRow buttons: %w", err)
		}
		if err := vecBound(b, k); err != nil {
			return nil, err
		}
		row := tg.KeyboardInlineButtonRow{}
		if k > 0 {
			row.Buttons = make([]tg.KeyboardInlineButton, 0, k%bin.PreallocateLimit)
		}
		for j := 0; j < k; j++ {
			btn, err := getInlineButton(b, l)
			if err != nil {
				return nil, fmt.Errorf("decode row %d button %d: %w", i, j, err)
			}
			row.Buttons = append(row.Buttons, btn)
		}
		m.Rows = append(m.Rows, row)
	}
	return m, nil
}

// getInlineButton decodes one KeyboardInlineButton.
func getInlineButton(b *bin.Buffer, l Layer) (tg.KeyboardInlineButton, error) {
	var out tg.KeyboardInlineButton
	id, ok := Ctor(l, SymKeyboardInlineButton)
	if !ok {
		return out, fmt.Errorf("%w: %s in layer %d", ErrAbsent, SymName(SymKeyboardInlineButton), l)
	}
	if err := readID(b, id, "keyboardInlineButton"); err != nil {
		return out, err
	}
	var flags bin.Fields
	if err := flags.Decode(b); err != nil {
		return out, fmt.Errorf("decode flags: %w", err)
	}
	out.Flags = flags
	if flags.Has(styleBit) {
		if err := out.Style.Decode(b); err != nil {
			return out, fmt.Errorf("decode style: %w", err)
		}
	}
	var err error
	if out.Text, err = b.String(); err != nil {
		return out, fmt.Errorf("decode text: %w", err)
	}
	if out.Type, err = tg.DecodeInlineButtonType(b); err != nil {
		return out, fmt.Errorf("decode inline button type: %w", err)
	}
	return out, nil
}

// readID reads a ctor id and checks it.
func readID(b *bin.Buffer, want uint32, what string) error {
	id, err := b.Uint32()
	if err != nil {
		return err
	}
	if id != want {
		return fmt.Errorf("%w: %s id %08x, want %08x", ErrMismatch, what, id, want)
	}
	return nil
}

// vecBound rejects vectors longer than the remaining buffer.
func vecBound(b *bin.Buffer, n int) error {
	if n < 0 || n > len(b.Buf) {
		return ErrLimit
	}
	return nil
}
