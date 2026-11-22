package layerx

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

// encodeDecodeLayer round-trips v at l and returns the decoded value.
func roundTripLayer(t *testing.T, l Layer, v bin.Object) bin.Object {
	t.Helper()
	var b bin.Buffer
	if err := Put(&b, l, v); err != nil {
		t.Fatalf("put %T at %d: %v", v, l, err)
	}
	decoded, err := Get(&bin.Buffer{Buf: append([]byte(nil), b.Buf...)}, l)
	if err != nil {
		t.Fatalf("get %T at %d: %v", v, l, err)
	}
	return decoded
}

func TestReplyKeyboardVariants228(t *testing.T) {
	poll := &tg.ButtonTypeRequestPoll{Quiz: true}
	poll.Flags.Set(0)
	markup := &tg.ReplyKeyboardMarkup{
		Resize: true,
		Rows: []tg.KeyboardButtonRow{
			{Buttons: []tg.KeyboardButton{
				{Text: "plain", Type: &tg.ButtonTypeDefault{}},
				{Text: "phone", Type: &tg.ButtonTypeRequestPhone{}},
				{Text: "geo", Type: &tg.ButtonTypeRequestGeoLocation{}},
				{Text: "poll", Type: poll},
				{Text: "peer", Type: &tg.ButtonTypeRequestPeer{
					ButtonID:    7,
					PeerType:    &tg.RequestPeerTypeUser{},
					MaxQuantity: 3,
				}},
				{Text: "web", Type: &tg.ButtonTypeSimpleWebView{URL: "https://example.com"}},
			}},
		},
	}
	obj := roundTripLayer(t, L228, markup)
	got, ok := obj.(*tg.ReplyKeyboardMarkup)
	if !ok {
		t.Fatalf("decoded %T, want *tg.ReplyKeyboardMarkup", obj)
	}
	if !got.Resize || len(got.Rows) != 1 || len(got.Rows[0].Buttons) != 6 {
		t.Fatalf("unexpected markup: %+v", got)
	}
	wantTypes := []any{
		&tg.ButtonTypeDefault{},
		&tg.ButtonTypeRequestPhone{},
		&tg.ButtonTypeRequestGeoLocation{},
		&tg.ButtonTypeRequestPoll{},
		&tg.ButtonTypeRequestPeer{},
		&tg.ButtonTypeSimpleWebView{},
	}
	for i, want := range wantTypes {
		if reflect.TypeOf(got.Rows[0].Buttons[i].Type) != reflect.TypeOf(want) {
			t.Fatalf("button %d type = %T, want %T", i, got.Rows[0].Buttons[i].Type, want)
		}
	}
	peer, ok := got.Rows[0].Buttons[4].Type.(*tg.ButtonTypeRequestPeer)
	if !ok || peer.ButtonID != 7 || peer.MaxQuantity != 3 {
		t.Fatalf("request-peer type = %#v", got.Rows[0].Buttons[4].Type)
	}
	web, ok := got.Rows[0].Buttons[5].Type.(*tg.ButtonTypeSimpleWebView)
	if !ok || web.URL != "https://example.com" {
		t.Fatalf("simple-web-view type = %#v", got.Rows[0].Buttons[5].Type)
	}
	quiz, ok := got.Rows[0].Buttons[3].Type.(*tg.ButtonTypeRequestPoll)
	if !ok || !quiz.Quiz || !quiz.Flags.Has(0) {
		t.Fatalf("request-poll type = %#v", got.Rows[0].Buttons[3].Type)
	}

	var b bin.Buffer
	if err := Put(&b, L228, markup); err != nil {
		t.Fatalf("put: %v", err)
	}
	decoded, err := Get(&bin.Buffer{Buf: append([]byte(nil), b.Buf...)}, L228)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	var again bin.Buffer
	if err := Put(&again, L228, decoded); err != nil {
		t.Fatalf("re-put: %v", err)
	}
	if !bytes.Equal(b.Buf, again.Buf) {
		t.Fatalf("round trip changed bytes\nfirst %x\nsecond %x", b.Buf, again.Buf)
	}
}

func TestUpdateProjectionDropsUnsupportedMembers(t *testing.T) {
	dropped := &tg.UpdateTranscribedAudioFile{TranscriptionID: 1, Text: "hi"}
	known := &tg.UpdateDeleteMessages{Messages: []int{1}, Pts: 1, PtsCount: 1}

	for _, l := range []Layer{L228, L229, L230, L231, L232} {
		var standalone bin.Buffer
		if err := Put(&standalone, l, dropped); err == nil {
			t.Fatalf("layer %d: standalone dropped update encoded", l)
		}

		var b bin.Buffer
		container := &tg.Updates{Updates: []tg.UpdateClass{dropped, known}, Date: 1, Seq: 1}
		if err := Put(&b, l, container); err != nil {
			t.Fatalf("layer %d: put container: %v", l, err)
		}
		decoded, err := Get(&bin.Buffer{Buf: append([]byte(nil), b.Buf...)}, l)
		if err != nil {
			t.Fatalf("layer %d: get container: %v", l, err)
		}
		got, ok := decoded.(*tg.Updates)
		if !ok || len(got.Updates) != 1 {
			t.Fatalf("layer %d: updates = %#v", l, got)
		}
		if _, ok := got.Updates[0].(*tg.UpdateDeleteMessages); !ok {
			t.Fatalf("layer %d: kept %T", l, got.Updates[0])
		}
	}

	// The canonical layer keeps both members.
	decoded := roundTripLayer(t, L233, &tg.Updates{Updates: []tg.UpdateClass{dropped, known}, Date: 1, Seq: 1})
	got := decoded.(*tg.Updates)
	if len(got.Updates) != 2 {
		t.Fatalf("layer 233 dropped a representable update: %#v", got.Updates)
	}
}

func TestWalletUpdatesFailClosedEarly(t *testing.T) {
	wallet := &tg.UpdateWalletState{State: &tg.WalletStateEmpty{}}
	container := &tg.Updates{Updates: []tg.UpdateClass{wallet}, Date: 1, Seq: 1}
	for _, l := range []Layer{L228, L229} {
		var b bin.Buffer
		err := Put(&b, l, container)
		if !errors.Is(err, ErrAbsent) {
			t.Fatalf("layer %d: err = %v, want ErrAbsent", l, err)
		}
	}
}

func TestWidgetMediaProjectedToUnsupported(t *testing.T) {
	for _, l := range []Layer{L228, L229, L230, L231, L232} {
		var b bin.Buffer
		if err := Put(&b, l, &tg.MessageMediaWidget{}); err != nil {
			t.Fatalf("layer %d: put: %v", l, err)
		}
		decoded, err := Get(&bin.Buffer{Buf: append([]byte(nil), b.Buf...)}, l)
		if err != nil {
			t.Fatalf("layer %d: get: %v", l, err)
		}
		if _, ok := decoded.(*tg.MessageMediaUnsupported); !ok {
			t.Fatalf("layer %d: got %T, want *tg.MessageMediaUnsupported", l, decoded)
		}
	}
}

func TestTextPayloadAdaptersEarlyLayers(t *testing.T) {
	bold := &tg.MessageEntityBold{Offset: 0, Length: 2}
	for _, l := range []Layer{L228, L229} {
		verification := &tg.BotVerification{
			BotID:       1,
			Icon:        2,
			Description: tg.TextWithEntities{Text: "hello", Entities: []tg.MessageEntityClass{bold}},
		}
		got, ok := roundTripLayer(t, l, verification).(*tg.BotVerification)
		if !ok {
			t.Fatalf("layer %d: verification type", l)
		}
		if got.BotID != 1 || got.Icon != 2 || got.Description.Text != "hello" {
			t.Fatalf("layer %d: verification = %#v", l, got)
		}
		if len(got.Description.Entities) != 0 {
			t.Fatalf("layer %d: entities survived a string-only layer", l)
		}

		settings := &tg.BotVerifierSettings{
			CanModifyCustomDescription: true,
			Icon:                       3,
			Company:                    "acme",
			CustomDescription:          tg.TextWithEntities{Text: "custom"},
		}
		settings.Flags.Set(0)
		gotSettings, ok := roundTripLayer(t, l, settings).(*tg.BotVerifierSettings)
		if !ok {
			t.Fatalf("layer %d: settings type", l)
		}
		if !gotSettings.CanModifyCustomDescription || gotSettings.CustomDescription.Text != "custom" {
			t.Fatalf("layer %d: settings = %#v", l, gotSettings)
		}

		req := &tg.BotsSetCustomVerificationRequest{
			Enabled:           true,
			Peer:              &tg.InputPeerSelf{},
			CustomDescription: tg.TextWithEntities{Text: "desc"},
		}
		req.Flags.Set(2)
		gotReq, ok := roundTripLayer(t, l, req).(*tg.BotsSetCustomVerificationRequest)
		if !ok {
			t.Fatalf("layer %d: request type", l)
		}
		if !gotReq.Enabled || gotReq.CustomDescription.Text != "desc" {
			t.Fatalf("layer %d: request = %#v", l, gotReq)
		}
	}
}

func TestToggleForumSidebarProjection(t *testing.T) {
	withSidebar := &tg.ChannelsToggleForumRequest{
		Channel: &tg.InputChannelEmpty{},
		Enabled: true,
		Tabs:    true,
		Sidebar: true,
	}
	withoutSidebar := &tg.ChannelsToggleForumRequest{
		Channel: &tg.InputChannelEmpty{},
		Enabled: true,
		Tabs:    true,
	}
	for _, l := range []Layer{L228, L229, L230, L231, L232} {
		var a, b bin.Buffer
		if err := Put(&a, l, withSidebar); err != nil {
			t.Fatalf("layer %d: put sidebar: %v", l, err)
		}
		if err := Put(&b, l, withoutSidebar); err != nil {
			t.Fatalf("layer %d: put no-sidebar: %v", l, err)
		}
		if !bytes.Equal(a.Buf, b.Buf) {
			t.Fatalf("layer %d: sidebar changed the wire bytes", l)
		}
		decoded, err := Get(&bin.Buffer{Buf: append([]byte(nil), a.Buf...)}, l)
		if err != nil {
			t.Fatalf("layer %d: get: %v", l, err)
		}
		req, ok := decoded.(*tg.ChannelsToggleForumRequest)
		if !ok || req.Sidebar {
			t.Fatalf("layer %d: decoded = %#v, want sidebar=false", l, decoded)
		}
	}
}

func TestLegacyInputButtonAdapters(t *testing.T) {
	reply := &tg.ReplyKeyboardMarkup{
		Rows: []tg.KeyboardButtonRow{{Buttons: []tg.KeyboardButton{{
			Text: "peer",
			Type: &tg.InputButtonTypeRequestPeer{
				NameRequested:     true,
				UsernameRequested: true,
				ButtonID:          9,
				PeerType:          &tg.RequestPeerTypeUser{},
				MaxQuantity:       4,
			},
		}}}},
	}
	got := roundTripLayer(t, L228, reply).(*tg.ReplyKeyboardMarkup)
	peer, ok := got.Rows[0].Buttons[0].Type.(*tg.InputButtonTypeRequestPeer)
	if !ok || !peer.NameRequested || !peer.UsernameRequested || peer.ButtonID != 9 || peer.MaxQuantity != 4 {
		t.Fatalf("input request-peer = %#v", got.Rows[0].Buttons[0].Type)
	}

	inline := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardInlineButtonRow{{Buttons: []tg.KeyboardInlineButton{{
			Text: "profile",
			Type: &tg.InputInlineButtonTypeUserProfile{UserID: &tg.InputUserSelf{}},
		}}}},
	}
	gotInline := roundTripLayer(t, L228, inline).(*tg.ReplyInlineMarkup)
	profile, ok := gotInline.Rows[0].Buttons[0].Type.(*tg.InputInlineButtonTypeUserProfile)
	if !ok {
		t.Fatalf("input user-profile = %#v", gotInline.Rows[0].Buttons[0].Type)
	}
	if _, ok := profile.UserID.(*tg.InputUserSelf); !ok {
		t.Fatalf("input user-profile user = %T", profile.UserID)
	}
}
