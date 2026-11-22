package layerx

import (
	"fmt"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

// Cross-layer obligations: shapes the canonical schema can express but an
// older layer cannot. Each helper either projects a value into the older wire
// form, substitutes a documented fallback, or fails closed. They are only
// reached from putObj/getObj, so callers keep using canonical objects.

// updateUnsupported reports how an update type behaves at layer l:
// drop means "omit it from a container", reject means "refuse to encode".
func updateUnsupported(l Layer, id uint32) (drop, reject bool) {
	switch l {
	case L228, L229:
		switch id {
		case tg.UpdateSentWalletTransactionTypeID,
			tg.UpdateWalletGaslessInfoTypeID,
			tg.UpdateWalletStateTypeID,
			tg.UpdateWalletTonConnectPendingDisconnectTypeID,
			tg.UpdateWalletTonConnectSessionTypeID:
			return false, true
		}
	}
	switch l {
	case L228, L229, L230, L231, L232:
		switch id {
		case tg.UpdateBotWidgetCopyQueryTypeID,
			tg.UpdateBotWidgetQueryTypeID,
			tg.UpdateEditForumTopicTypeID,
			tg.UpdateInlineBotWidgetQueryTypeID,
			tg.UpdateMessageWidgetDataTypeID,
			tg.UpdateTranscribedAudioFileTypeID:
			return true, false
		}
		if l == L228 && id == tg.UpdateEphemeralBotCallbackQueryTypeID {
			return true, false
		}
	}
	return false, false
}

// putUpdatesProjected encodes updates, updatesCombined and updateShort with
// per-layer update filtering. A member the layer cannot carry is skipped, so a
// single unknown update does not discard the whole push.
func putUpdatesProjected(b *bin.Buffer, l Layer, s Sym, v bin.Object) error {
	switch x := v.(type) {
	case *tg.Updates:
		if x == nil {
			return ErrNil
		}
		updates, err := encodeUpdateList(l, x.Updates)
		if err != nil {
			return err
		}
		if err := putWithID(b, l, SymUpdates); err != nil {
			return err
		}
		b.PutVectorHeader(len(updates))
		for _, u := range updates {
			if err := putObj(b, l, u); err != nil {
				return fmt.Errorf("encode update: %w", err)
			}
		}
		b.PutVectorHeader(len(x.Users))
		for _, u := range x.Users {
			if err := putObj(b, l, u); err != nil {
				return fmt.Errorf("encode user: %w", err)
			}
		}
		b.PutVectorHeader(len(x.Chats))
		for _, c := range x.Chats {
			if err := putObj(b, l, c); err != nil {
				return fmt.Errorf("encode chat: %w", err)
			}
		}
		b.PutInt(x.Date)
		b.PutInt(x.Seq)
		return nil
	case *tg.UpdatesCombined:
		if x == nil {
			return ErrNil
		}
		updates, err := encodeUpdateList(l, x.Updates)
		if err != nil {
			return err
		}
		if err := putWithID(b, l, SymUpdatesCombined); err != nil {
			return err
		}
		b.PutVectorHeader(len(updates))
		for _, u := range updates {
			if err := putObj(b, l, u); err != nil {
				return fmt.Errorf("encode update: %w", err)
			}
		}
		b.PutVectorHeader(len(x.Users))
		for _, u := range x.Users {
			if err := putObj(b, l, u); err != nil {
				return fmt.Errorf("encode user: %w", err)
			}
		}
		b.PutVectorHeader(len(x.Chats))
		for _, c := range x.Chats {
			if err := putObj(b, l, c); err != nil {
				return fmt.Errorf("encode chat: %w", err)
			}
		}
		b.PutInt(x.Date)
		b.PutInt(x.SeqStart)
		b.PutInt(x.Seq)
		return nil
	case *tg.UpdateShort:
		if x == nil {
			return ErrNil
		}
		if x.Update == nil {
			return fmt.Errorf("%w: updateShort without update", ErrNil)
		}
		drop, reject := updateUnsupported(l, x.Update.TypeID())
		if drop || reject || !hasCtor(l, x.Update) {
			return fmt.Errorf("%w: update %T in layer %d", ErrNoPlan, x.Update, l)
		}
		if err := putWithID(b, l, SymUpdateShort); err != nil {
			return err
		}
		if err := putObj(b, l, x.Update); err != nil {
			return err
		}
		b.PutInt(x.Date)
		return nil
	default:
		return fmt.Errorf("%w: %T as updates container", ErrMismatch, v)
	}
}

// encodeUpdateList filters updates the layer cannot express: rejected members
// fail the encode, members without a layer ctor are dropped.
func encodeUpdateList(l Layer, in []tg.UpdateClass) ([]tg.UpdateClass, error) {
	out := make([]tg.UpdateClass, 0, len(in))
	for _, u := range in {
		if u == nil {
			return nil, fmt.Errorf("%w: nil update", ErrNil)
		}
		drop, reject := updateUnsupported(l, u.TypeID())
		if reject {
			return nil, fmt.Errorf("%w: update %T in layer %d", ErrAbsent, u, l)
		}
		if drop || !hasCtor(l, u) {
			continue
		}
		out = append(out, u)
	}
	return out, nil
}

// hasCtor reports whether the canonical object has a wire id at l.
func hasCtor(l Layer, v bin.Object) bool {
	t, ok := v.(typeIDer)
	if !ok {
		return false
	}
	_, ok = Ctor(l, Sym(t.TypeID()))
	return ok
}

// putMessageMediaUnsupported writes messageMediaUnsupported, the placeholder
// older layers use for media kinds they do not know.
func putMessageMediaUnsupported(b *bin.Buffer, l Layer) error {
	return putWithID(b, l, SymMessageMediaUnsupported)
}

// putToggleForumProjected encodes channels.toggleForum for layers that predate
// the sidebar layout. The flag has no wire slot there and is omitted.
func putToggleForumProjected(b *bin.Buffer, l Layer, v bin.Object) error {
	req, ok := v.(*tg.ChannelsToggleForumRequest)
	if !ok {
		return fmt.Errorf("%w: %T as channels.toggleForum", ErrMismatch, v)
	}
	if req == nil {
		return ErrNil
	}
	if err := putWithID(b, l, SymChannelsToggleForum); err != nil {
		return err
	}
	if err := putObj(b, l, req.Channel); err != nil {
		return err
	}
	b.PutBool(req.Enabled)
	b.PutBool(req.Tabs)
	return nil
}

// TextWithEntities fields collapse to plain strings in the earliest layers.
// Only the visible text survives; formatting entities are not representable.

func putBotVerification229(b *bin.Buffer, l Layer, v *tg.BotVerification) error {
	if err := putWithID(b, l, SymBotVerification); err != nil {
		return err
	}
	b.PutLong(v.BotID)
	b.PutLong(v.Icon)
	b.PutString(v.Description.Text)
	return nil
}

func getBotVerification229(b *bin.Buffer) (bin.Object, error) {
	botID, err := b.Long()
	if err != nil {
		return nil, fmt.Errorf("decode bot_id: %w", err)
	}
	icon, err := b.Long()
	if err != nil {
		return nil, fmt.Errorf("decode icon: %w", err)
	}
	desc, err := b.String()
	if err != nil {
		return nil, fmt.Errorf("decode description: %w", err)
	}
	return &tg.BotVerification{
		BotID:       botID,
		Icon:        icon,
		Description: tg.TextWithEntities{Text: desc},
	}, nil
}

func putBotVerifierSettings229(b *bin.Buffer, l Layer, v *tg.BotVerifierSettings) error {
	var flags bin.Fields
	if v.Flags.Has(0) {
		flags.Set(0)
	}
	if v.CanModifyCustomDescription {
		flags.Set(1)
	}
	if err := putWithID(b, l, SymBotVerifierSettings); err != nil {
		return err
	}
	if err := flags.Encode(b); err != nil {
		return fmt.Errorf("encode flags: %w", err)
	}
	b.PutLong(v.Icon)
	b.PutString(v.Company)
	if flags.Has(0) {
		b.PutString(v.CustomDescription.Text)
	}
	return nil
}

func getBotVerifierSettings229(b *bin.Buffer) (bin.Object, error) {
	flags, err := b.Uint32()
	if err != nil {
		return nil, fmt.Errorf("decode flags: %w", err)
	}
	icon, err := b.Long()
	if err != nil {
		return nil, fmt.Errorf("decode icon: %w", err)
	}
	company, err := b.String()
	if err != nil {
		return nil, fmt.Errorf("decode company: %w", err)
	}
	out := &tg.BotVerifierSettings{
		Icon:                       icon,
		Company:                    company,
		CanModifyCustomDescription: flags&(1<<1) != 0,
	}
	if flags&(1<<0) != 0 {
		desc, err := b.String()
		if err != nil {
			return nil, fmt.Errorf("decode custom_description: %w", err)
		}
		out.CustomDescription = tg.TextWithEntities{Text: desc}
		out.Flags.Set(0)
	}
	if out.CanModifyCustomDescription {
		out.Flags.Set(1)
	}
	return out, nil
}

func putBotsSetCustomVerification229(b *bin.Buffer, l Layer, v *tg.BotsSetCustomVerificationRequest) error {
	var flags bin.Fields
	hasBot := v.Flags.Has(0) || v.Bot != nil
	hasDesc := v.Flags.Has(2)
	if hasBot {
		flags.Set(0)
	}
	if v.Enabled {
		flags.Set(1)
	}
	if hasDesc {
		flags.Set(2)
	}
	if err := putWithID(b, l, SymBotsSetCustomVerification); err != nil {
		return err
	}
	if err := flags.Encode(b); err != nil {
		return fmt.Errorf("encode flags: %w", err)
	}
	if hasBot {
		if err := putObj(b, l, v.Bot); err != nil {
			return err
		}
	}
	if err := putObj(b, l, v.Peer); err != nil {
		return err
	}
	if hasDesc {
		b.PutString(v.CustomDescription.Text)
	}
	return nil
}

func getBotsSetCustomVerification229(b *bin.Buffer, l Layer) (bin.Object, error) {
	flags, err := b.Uint32()
	if err != nil {
		return nil, fmt.Errorf("decode flags: %w", err)
	}
	out := &tg.BotsSetCustomVerificationRequest{Enabled: flags&(1<<1) != 0}
	if flags&(1<<0) != 0 {
		bot, err := getObj(b, l)
		if err != nil {
			return nil, fmt.Errorf("decode bot: %w", err)
		}
		user, ok := bot.(tg.InputUserClass)
		if !ok {
			return nil, fmt.Errorf("%w: bot %T", ErrMismatch, bot)
		}
		out.Bot = user
		out.Flags.Set(0)
	}
	peer, err := getObj(b, l)
	if err != nil {
		return nil, fmt.Errorf("decode peer: %w", err)
	}
	inputPeer, ok := peer.(tg.InputPeerClass)
	if !ok {
		return nil, fmt.Errorf("%w: peer %T", ErrMismatch, peer)
	}
	out.Peer = inputPeer
	if flags&(1<<2) != 0 {
		desc, err := b.String()
		if err != nil {
			return nil, fmt.Errorf("decode custom_description: %w", err)
		}
		out.CustomDescription = tg.TextWithEntities{Text: desc}
		out.Flags.Set(2)
	}
	if out.Enabled {
		out.Flags.Set(1)
	}
	return out, nil
}
