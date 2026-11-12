package layerx

import (
	"context"
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

func TestPrivTable(t *testing.T) {
	s, ok := PrivSym(0x812c2ae6)
	if !ok {
		t.Fatal("id not mapped")
	}
	if !IsPriv(s) {
		t.Fatal("sym not private")
	}
	if got := PrivName(s); got != "messages.getStatsURL" {
		t.Fatalf("name %q", got)
	}
	if _, ok := PrivSkip(0x555555f6); !ok {
		t.Fatal("skip id not listed")
	}
	if _, ok := PrivSym(0x555555f6); ok {
		t.Fatal("skip id should not map")
	}
}

// privReq is one private request case.
type privReq struct {
	sym Sym
	obj bin.Object
}

func TestPrivRequests(t *testing.T) {
	cases := []privReq{
		{sym: SymPrivGetStatsURL, obj: &PrivGetStatsURL{
			Dark: true, Peer: &tg.InputPeerSelf{}, Params: "p",
		}},
		{sym: SymPrivGetAllChats, obj: &PrivGetAllChats{ExceptIDs: []int64{1, 2}}},
		{sym: SymPrivForwardMessage, obj: &PrivForwardMessage{
			Peer: &tg.InputPeerSelf{}, ID: 7, RandomID: 8,
		}},
		{sym: SymPrivReportReaction, obj: &PrivReportReaction{
			Peer: &tg.InputPeerSelf{}, ID: 3, UserID: &tg.InputUserSelf{},
		}},
		{sym: SymPrivGetWebViewResult, obj: &PrivGetWebViewResult{
			Peer: &tg.InputPeerSelf{}, Bot: &tg.InputUserSelf{}, QueryID: 9,
		}},
		{sym: SymPrivSetWebViewResult, obj: &PrivSetWebViewResult{QueryID: 10}},
		{sym: SymPrivSendWebViewResultMessage, obj: &PrivSendWebViewResultMessage{QueryID: 11}},
		{sym: SymPrivRequestRecurringPayment, obj: &PrivRequestRecurringPayment{
			UserID:              &tg.InputUserSelf{},
			RecurringInitCharge: "100",
			InvoiceMedia:        &tg.InputMediaEmpty{},
		}},
		{sym: SymPrivImportCard, obj: &PrivImportCard{ExportCard: []int{1, 2, 3}}},
		{sym: SymPrivDeleteFolder, obj: &PrivDeleteFolder{FolderID: 4}},
	}
	for _, c := range cases {
		tbl := NewTable()
		if err := tbl.Add(c.sym, func(ctx context.Context, m *Mark) (Out, error) {
			return &tg.BoolTrue{}, nil
		}); err != nil {
			t.Fatal(err)
		}
		for _, l := range Layers() {
			var b bin.Buffer
			if err := Put(&b, l, c.obj); err != nil {
				t.Fatalf("%s layer %d: put: %v", SymName(c.sym), l, err)
			}
			a, err := tbl.Take(l, &b)
			if err != nil {
				t.Fatalf("%s layer %d: take: %v", SymName(c.sym), l, err)
			}
			if a.Mark().Sym() != c.sym {
				t.Fatalf("%s: sym %s", SymName(c.sym), SymName(a.Mark().Sym()))
			}
			if _, ok := a.Mark().Obj().(*PrivGetStatsURL); ok && c.sym == SymPrivGetStatsURL {
				continue
			}
		}
	}
}

func TestPrivSkipView(t *testing.T) {
	tbl := NewTable()
	var got OddView
	tbl.OnOdd(func(v OddView) { got = v })
	var b bin.Buffer
	b.PutID(0x555555f6)
	if _, err := tbl.Take(L228, &b); err == nil {
		t.Fatal("expected error")
	}
	if !got.Priv || got.Name != "audioEncrypted" {
		t.Fatalf("odd view %+v", got)
	}
}
