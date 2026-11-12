package layerx_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/layerx"
	"github.com/gotd/td/tg"
)

// downstream exercises the server contract.
func TestDownstream(t *testing.T) {
	tbl := layerx.NewTable()
	methods := map[layerx.Sym]layerx.Handler{
		layerx.SymHelpGetConfig: func(ctx context.Context, m *layerx.Mark) (layerx.Out, error) {
			return &tg.Config{Date: 1, Expires: 2}, nil
		},
		layerx.SymUsersGetUsers: func(ctx context.Context, m *layerx.Mark) (layerx.Out, error) {
			return &tg.User{ID: 1, FirstName: "cat"}, nil
		},
		layerx.SymMessagesGetHistory: func(ctx context.Context, m *layerx.Mark) (layerx.Out, error) {
			return &tg.MessagesMessages{Messages: []tg.MessageClass{}}, nil
		},
	}
	for s, h := range methods {
		if err := tbl.Add(s, h); err != nil {
			t.Fatal(err)
		}
	}

	reqs := map[string]func() bin.Object{
		"config": func() bin.Object { return &tg.HelpGetConfigRequest{} },
		"users": func() bin.Object {
			return &tg.UsersGetUsersRequest{ID: []tg.InputUserClass{&tg.InputUserSelf{}}}
		},
		"history": func() bin.Object {
			return &tg.MessagesGetHistoryRequest{Peer: &tg.InputPeerSelf{}}
		},
	}
	for name, mk := range reqs {
		for _, l := range layerx.Layers() {
			var inner bin.Buffer
			if err := layerx.Put(&inner, l, mk()); err != nil {
				t.Fatalf("%s layer %d: put: %v", name, l, err)
			}
			var outer bin.Buffer
			outer.PutID(tg.InvokeWithLayerRequestTypeID)
			outer.PutInt(int(l))
			outer.Buf = append(outer.Buf, inner.Buf...)
			a, err := tbl.TakeOpt(layerx.Canon, &outer, layerx.Opts{})
			if err != nil {
				t.Fatalf("%s layer %d: take: %v", name, l, err)
			}
			if a.Layer() != l {
				t.Fatalf("%s: layer %d != %d", name, a.Layer(), l)
			}
			out, err := tbl.Serve(context.Background(), a)
			if err != nil {
				t.Fatalf("%s layer %d: serve: %v", name, l, err)
			}
			var res bin.Buffer
			if err := a.Mark().PutRes(&res, out); err != nil {
				t.Fatalf("%s layer %d: putres: %v", name, l, err)
			}
			frozen, err := layerx.Freeze(out)
			if err != nil {
				t.Fatal(err)
			}
			var fan bin.Buffer
			if err := frozen.Write(&fan, l); err != nil {
				t.Fatalf("%s layer %d: fanout: %v", name, l, err)
			}
			if !bytes.Equal(res.Buf, fan.Buf) {
				t.Fatalf("%s layer %d: result bytes differ", name, l)
			}
		}
	}
}
