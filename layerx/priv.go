package layerx

import (
	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

// Client ids of DrKLO private requests.
const (
	privGetStatsURLID              uint32 = 0x812c2ae6
	privGetAllChatsID              uint32 = 0x875f74be
	privForwardMessageID           uint32 = 0x33963bf9
	privReportReactionID           uint32 = 0x61422a48
	privGetWebViewResultID         uint32 = 0x22b6c214
	privSetWebViewResultID         uint32 = 0xe41cd11d
	privSendWebViewResultMessageID uint32 = 0xddcf50eb
	privRequestRecurringPaymentID  uint32 = 0x146e958d
	privImportCardID               uint32 = 0x4fe196fe
	privDeleteFolderID             uint32 = 0x1c295881
)

// PrivGetStatsURL is messages.getStatsURL.
type PrivGetStatsURL struct {
	Flags  uint32
	Dark   bool
	Peer   tg.InputPeerClass
	Params string
}

// TypeID returns client id.
func (*PrivGetStatsURL) TypeID() uint32 { return privGetStatsURLID }

// TypeName returns TL name.
func (*PrivGetStatsURL) TypeName() string { return "messages.getStatsURL" }

// Encode implements bin.Encoder.
func (p *PrivGetStatsURL) Encode(b *bin.Buffer) error {
	b.PutID(privGetStatsURLID)
	return p.encodePriv(b, Canon)
}

// EncodeBare implements bin.BareEncoder.
func (p *PrivGetStatsURL) EncodeBare(b *bin.Buffer) error { return p.encodePriv(b, Canon) }

// Decode implements bin.Decoder.
func (p *PrivGetStatsURL) Decode(b *bin.Buffer) error {
	if err := b.ConsumeID(privGetStatsURLID); err != nil {
		return err
	}
	return p.decodePriv(b, Canon)
}

// DecodeBare implements bin.BareDecoder.
func (p *PrivGetStatsURL) DecodeBare(b *bin.Buffer) error { return p.decodePriv(b, Canon) }

func (p *PrivGetStatsURL) encodePriv(b *bin.Buffer, l Layer) error {
	p.Flags = 0
	if p.Dark {
		p.Flags |= 1
	}
	b.PutUint32(p.Flags)
	if err := putObj(b, l, p.Peer); err != nil {
		return err
	}
	b.PutString(p.Params)
	return nil
}

func (p *PrivGetStatsURL) decodePriv(b *bin.Buffer, l Layer) error {
	flags, err := b.Uint32()
	if err != nil {
		return err
	}
	p.Flags = flags
	p.Dark = flags&1 != 0
	peer, err := getObj(b, l)
	if err != nil {
		return err
	}
	p.Peer = peer.(tg.InputPeerClass)
	p.Params, err = b.String()
	return err
}

// PrivGetAllChats is messages.getAllChats.
type PrivGetAllChats struct {
	ExceptIDs []int64
}

// TypeID returns client id.
func (*PrivGetAllChats) TypeID() uint32 { return privGetAllChatsID }

// TypeName returns TL name.
func (*PrivGetAllChats) TypeName() string { return "messages.getAllChats" }

// Encode implements bin.Encoder.
func (p *PrivGetAllChats) Encode(b *bin.Buffer) error {
	b.PutID(privGetAllChatsID)
	return p.encodePriv(b, Canon)
}

// EncodeBare implements bin.BareEncoder.
func (p *PrivGetAllChats) EncodeBare(b *bin.Buffer) error { return p.encodePriv(b, Canon) }

// Decode implements bin.Decoder.
func (p *PrivGetAllChats) Decode(b *bin.Buffer) error {
	if err := b.ConsumeID(privGetAllChatsID); err != nil {
		return err
	}
	return p.decodePriv(b, Canon)
}

// DecodeBare implements bin.BareDecoder.
func (p *PrivGetAllChats) DecodeBare(b *bin.Buffer) error { return p.decodePriv(b, Canon) }

func (p *PrivGetAllChats) encodePriv(b *bin.Buffer, l Layer) error {
	b.PutVectorHeader(len(p.ExceptIDs))
	for _, v := range p.ExceptIDs {
		b.PutLong(v)
	}
	return nil
}

func (p *PrivGetAllChats) decodePriv(b *bin.Buffer, l Layer) error {
	n, err := b.VectorHeader()
	if err != nil {
		return err
	}
	p.ExceptIDs = p.ExceptIDs[:0]
	for i := 0; i < n; i++ {
		v, err := b.Long()
		if err != nil {
			return err
		}
		p.ExceptIDs = append(p.ExceptIDs, v)
	}
	return nil
}

// PrivForwardMessage is messages.forwardMessage.
type PrivForwardMessage struct {
	Peer     tg.InputPeerClass
	ID       int
	RandomID int64
}

// TypeID returns client id.
func (*PrivForwardMessage) TypeID() uint32 { return privForwardMessageID }

// TypeName returns TL name.
func (*PrivForwardMessage) TypeName() string { return "messages.forwardMessage" }

// Encode implements bin.Encoder.
func (p *PrivForwardMessage) Encode(b *bin.Buffer) error {
	b.PutID(privForwardMessageID)
	return p.encodePriv(b, Canon)
}

// EncodeBare implements bin.BareEncoder.
func (p *PrivForwardMessage) EncodeBare(b *bin.Buffer) error { return p.encodePriv(b, Canon) }

// Decode implements bin.Decoder.
func (p *PrivForwardMessage) Decode(b *bin.Buffer) error {
	if err := b.ConsumeID(privForwardMessageID); err != nil {
		return err
	}
	return p.decodePriv(b, Canon)
}

// DecodeBare implements bin.BareDecoder.
func (p *PrivForwardMessage) DecodeBare(b *bin.Buffer) error { return p.decodePriv(b, Canon) }

func (p *PrivForwardMessage) encodePriv(b *bin.Buffer, l Layer) error {
	if err := putObj(b, l, p.Peer); err != nil {
		return err
	}
	b.PutInt(p.ID)
	b.PutLong(p.RandomID)
	return nil
}

func (p *PrivForwardMessage) decodePriv(b *bin.Buffer, l Layer) error {
	peer, err := getObj(b, l)
	if err != nil {
		return err
	}
	p.Peer = peer.(tg.InputPeerClass)
	if p.ID, err = b.Int(); err != nil {
		return err
	}
	p.RandomID, err = b.Long()
	return err
}

// PrivReportReaction is messages.reportReaction.
type PrivReportReaction struct {
	Peer   tg.InputPeerClass
	ID     int
	UserID tg.InputUserClass
}

// TypeID returns client id.
func (*PrivReportReaction) TypeID() uint32 { return privReportReactionID }

// TypeName returns TL name.
func (*PrivReportReaction) TypeName() string { return "messages.reportReaction" }

// Encode implements bin.Encoder.
func (p *PrivReportReaction) Encode(b *bin.Buffer) error {
	b.PutID(privReportReactionID)
	return p.encodePriv(b, Canon)
}

// EncodeBare implements bin.BareEncoder.
func (p *PrivReportReaction) EncodeBare(b *bin.Buffer) error { return p.encodePriv(b, Canon) }

// Decode implements bin.Decoder.
func (p *PrivReportReaction) Decode(b *bin.Buffer) error {
	if err := b.ConsumeID(privReportReactionID); err != nil {
		return err
	}
	return p.decodePriv(b, Canon)
}

// DecodeBare implements bin.BareDecoder.
func (p *PrivReportReaction) DecodeBare(b *bin.Buffer) error { return p.decodePriv(b, Canon) }

func (p *PrivReportReaction) encodePriv(b *bin.Buffer, l Layer) error {
	if err := putObj(b, l, p.Peer); err != nil {
		return err
	}
	b.PutInt(p.ID)
	return putObj(b, l, p.UserID)
}

func (p *PrivReportReaction) decodePriv(b *bin.Buffer, l Layer) error {
	peer, err := getObj(b, l)
	if err != nil {
		return err
	}
	p.Peer = peer.(tg.InputPeerClass)
	if p.ID, err = b.Int(); err != nil {
		return err
	}
	u, err := getObj(b, l)
	if err != nil {
		return err
	}
	p.UserID = u.(tg.InputUserClass)
	return nil
}

// PrivGetWebViewResult is messages.getWebViewResult.
type PrivGetWebViewResult struct {
	Peer    tg.InputPeerClass
	Bot     tg.InputUserClass
	QueryID int64
}

// TypeID returns client id.
func (*PrivGetWebViewResult) TypeID() uint32 { return privGetWebViewResultID }

// TypeName returns TL name.
func (*PrivGetWebViewResult) TypeName() string { return "messages.getWebViewResult" }

// Encode implements bin.Encoder.
func (p *PrivGetWebViewResult) Encode(b *bin.Buffer) error {
	b.PutID(privGetWebViewResultID)
	return p.encodePriv(b, Canon)
}

// EncodeBare implements bin.BareEncoder.
func (p *PrivGetWebViewResult) EncodeBare(b *bin.Buffer) error { return p.encodePriv(b, Canon) }

// Decode implements bin.Decoder.
func (p *PrivGetWebViewResult) Decode(b *bin.Buffer) error {
	if err := b.ConsumeID(privGetWebViewResultID); err != nil {
		return err
	}
	return p.decodePriv(b, Canon)
}

// DecodeBare implements bin.BareDecoder.
func (p *PrivGetWebViewResult) DecodeBare(b *bin.Buffer) error { return p.decodePriv(b, Canon) }

func (p *PrivGetWebViewResult) encodePriv(b *bin.Buffer, l Layer) error {
	if err := putObj(b, l, p.Peer); err != nil {
		return err
	}
	if err := putObj(b, l, p.Bot); err != nil {
		return err
	}
	b.PutLong(p.QueryID)
	return nil
}

func (p *PrivGetWebViewResult) decodePriv(b *bin.Buffer, l Layer) error {
	peer, err := getObj(b, l)
	if err != nil {
		return err
	}
	p.Peer = peer.(tg.InputPeerClass)
	bot, err := getObj(b, l)
	if err != nil {
		return err
	}
	p.Bot = bot.(tg.InputUserClass)
	p.QueryID, err = b.Long()
	return err
}

// PrivSetWebViewResult is messages.setWebViewResult.
type PrivSetWebViewResult struct {
	QueryID int64
}

// TypeID returns client id.
func (*PrivSetWebViewResult) TypeID() uint32 { return privSetWebViewResultID }

// TypeName returns TL name.
func (*PrivSetWebViewResult) TypeName() string { return "messages.setWebViewResult" }

// Encode implements bin.Encoder.
func (p *PrivSetWebViewResult) Encode(b *bin.Buffer) error {
	b.PutID(privSetWebViewResultID)
	return p.encodePriv(b, Canon)
}

// EncodeBare implements bin.BareEncoder.
func (p *PrivSetWebViewResult) EncodeBare(b *bin.Buffer) error { return p.encodePriv(b, Canon) }

// Decode implements bin.Decoder.
func (p *PrivSetWebViewResult) Decode(b *bin.Buffer) error {
	if err := b.ConsumeID(privSetWebViewResultID); err != nil {
		return err
	}
	return p.decodePriv(b, Canon)
}

// DecodeBare implements bin.BareDecoder.
func (p *PrivSetWebViewResult) DecodeBare(b *bin.Buffer) error { return p.decodePriv(b, Canon) }

func (p *PrivSetWebViewResult) encodePriv(b *bin.Buffer, l Layer) error {
	b.PutLong(p.QueryID)
	return nil
}

func (p *PrivSetWebViewResult) decodePriv(b *bin.Buffer, l Layer) error {
	v, err := b.Long()
	p.QueryID = v
	return err
}

// PrivSendWebViewResultMessage is messages.sendWebViewResultMessage.
type PrivSendWebViewResultMessage struct {
	QueryID int64
}

// TypeID returns client id.
func (*PrivSendWebViewResultMessage) TypeID() uint32 { return privSendWebViewResultMessageID }

// TypeName returns TL name.
func (*PrivSendWebViewResultMessage) TypeName() string {
	return "messages.sendWebViewResultMessage"
}

// Encode implements bin.Encoder.
func (p *PrivSendWebViewResultMessage) Encode(b *bin.Buffer) error {
	b.PutID(privSendWebViewResultMessageID)
	return p.encodePriv(b, Canon)
}

// EncodeBare implements bin.BareEncoder.
func (p *PrivSendWebViewResultMessage) EncodeBare(b *bin.Buffer) error { return p.encodePriv(b, Canon) }

// Decode implements bin.Decoder.
func (p *PrivSendWebViewResultMessage) Decode(b *bin.Buffer) error {
	if err := b.ConsumeID(privSendWebViewResultMessageID); err != nil {
		return err
	}
	return p.decodePriv(b, Canon)
}

// DecodeBare implements bin.BareDecoder.
func (p *PrivSendWebViewResultMessage) DecodeBare(b *bin.Buffer) error { return p.decodePriv(b, Canon) }

func (p *PrivSendWebViewResultMessage) encodePriv(b *bin.Buffer, l Layer) error {
	b.PutLong(p.QueryID)
	return nil
}

func (p *PrivSendWebViewResultMessage) decodePriv(b *bin.Buffer, l Layer) error {
	v, err := b.Long()
	p.QueryID = v
	return err
}

// PrivRequestRecurringPayment is payments.requestRecurringPayment.
type PrivRequestRecurringPayment struct {
	UserID              tg.InputUserClass
	RecurringInitCharge string
	InvoiceMedia        tg.InputMediaClass
}

// TypeID returns client id.
func (*PrivRequestRecurringPayment) TypeID() uint32 { return privRequestRecurringPaymentID }

// TypeName returns TL name.
func (*PrivRequestRecurringPayment) TypeName() string {
	return "payments.requestRecurringPayment"
}

// Encode implements bin.Encoder.
func (p *PrivRequestRecurringPayment) Encode(b *bin.Buffer) error {
	b.PutID(privRequestRecurringPaymentID)
	return p.encodePriv(b, Canon)
}

// EncodeBare implements bin.BareEncoder.
func (p *PrivRequestRecurringPayment) EncodeBare(b *bin.Buffer) error {
	return p.encodePriv(b, Canon)
}

// Decode implements bin.Decoder.
func (p *PrivRequestRecurringPayment) Decode(b *bin.Buffer) error {
	if err := b.ConsumeID(privRequestRecurringPaymentID); err != nil {
		return err
	}
	return p.decodePriv(b, Canon)
}

// DecodeBare implements bin.BareDecoder.
func (p *PrivRequestRecurringPayment) DecodeBare(b *bin.Buffer) error {
	return p.decodePriv(b, Canon)
}

func (p *PrivRequestRecurringPayment) encodePriv(b *bin.Buffer, l Layer) error {
	if err := putObj(b, l, p.UserID); err != nil {
		return err
	}
	b.PutString(p.RecurringInitCharge)
	return putObj(b, l, p.InvoiceMedia)
}

func (p *PrivRequestRecurringPayment) decodePriv(b *bin.Buffer, l Layer) error {
	u, err := getObj(b, l)
	if err != nil {
		return err
	}
	p.UserID = u.(tg.InputUserClass)
	if p.RecurringInitCharge, err = b.String(); err != nil {
		return err
	}
	m, err := getObj(b, l)
	if err != nil {
		return err
	}
	p.InvoiceMedia = m.(tg.InputMediaClass)
	return nil
}

// PrivImportCard is contacts.importCard.
type PrivImportCard struct {
	ExportCard []int
}

// TypeID returns client id.
func (*PrivImportCard) TypeID() uint32 { return privImportCardID }

// TypeName returns TL name.
func (*PrivImportCard) TypeName() string { return "contacts.importCard" }

// Encode implements bin.Encoder.
func (p *PrivImportCard) Encode(b *bin.Buffer) error {
	b.PutID(privImportCardID)
	return p.encodePriv(b, Canon)
}

// EncodeBare implements bin.BareEncoder.
func (p *PrivImportCard) EncodeBare(b *bin.Buffer) error { return p.encodePriv(b, Canon) }

// Decode implements bin.Decoder.
func (p *PrivImportCard) Decode(b *bin.Buffer) error {
	if err := b.ConsumeID(privImportCardID); err != nil {
		return err
	}
	return p.decodePriv(b, Canon)
}

// DecodeBare implements bin.BareDecoder.
func (p *PrivImportCard) DecodeBare(b *bin.Buffer) error { return p.decodePriv(b, Canon) }

func (p *PrivImportCard) encodePriv(b *bin.Buffer, l Layer) error {
	b.PutVectorHeader(len(p.ExportCard))
	for _, v := range p.ExportCard {
		b.PutInt(v)
	}
	return nil
}

func (p *PrivImportCard) decodePriv(b *bin.Buffer, l Layer) error {
	n, err := b.VectorHeader()
	if err != nil {
		return err
	}
	p.ExportCard = p.ExportCard[:0]
	for i := 0; i < n; i++ {
		v, err := b.Int()
		if err != nil {
			return err
		}
		p.ExportCard = append(p.ExportCard, v)
	}
	return nil
}

// PrivDeleteFolder is folders.deleteFolder.
type PrivDeleteFolder struct {
	FolderID int
}

// TypeID returns client id.
func (*PrivDeleteFolder) TypeID() uint32 { return privDeleteFolderID }

// TypeName returns TL name.
func (*PrivDeleteFolder) TypeName() string { return "folders.deleteFolder" }

// Encode implements bin.Encoder.
func (p *PrivDeleteFolder) Encode(b *bin.Buffer) error {
	b.PutID(privDeleteFolderID)
	return p.encodePriv(b, Canon)
}

// EncodeBare implements bin.BareEncoder.
func (p *PrivDeleteFolder) EncodeBare(b *bin.Buffer) error { return p.encodePriv(b, Canon) }

// Decode implements bin.Decoder.
func (p *PrivDeleteFolder) Decode(b *bin.Buffer) error {
	if err := b.ConsumeID(privDeleteFolderID); err != nil {
		return err
	}
	return p.decodePriv(b, Canon)
}

// DecodeBare implements bin.BareDecoder.
func (p *PrivDeleteFolder) DecodeBare(b *bin.Buffer) error { return p.decodePriv(b, Canon) }

func (p *PrivDeleteFolder) encodePriv(b *bin.Buffer, l Layer) error {
	b.PutInt(p.FolderID)
	return nil
}

func (p *PrivDeleteFolder) decodePriv(b *bin.Buffer, l Layer) error {
	v, err := b.Int()
	p.FolderID = v
	return err
}
