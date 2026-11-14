package layerx

import (
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

// Check must accept the bridge types and private alias ctors; otherwise every
// request carrying an inline keyboard or a DrKLO ctor falls back to the legacy
// parser before the typed handler runs.
func TestCheckMarkupBridgeL228(t *testing.T) {
	var b bin.Buffer
	if err := Put(&b, L228, &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardInlineButtonRow{{
			Buttons: []tg.KeyboardInlineButton{
				{Text: "a", Type: &tg.InlineButtonTypeCallback{Data: []byte{1}}},
			},
		}},
	}); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := Check(&bin.Buffer{Buf: b.Buf}, L228, Lim{}); err != nil {
		t.Fatalf("check markup: %v", err)
	}
}

func TestCheckPrivAliasCreateChat(t *testing.T) {
	var b bin.Buffer
	b.PutID(0x0034a818)
	b.PutInt32(0)
	b.PutVectorHeader(0)
	b.PutString("chat")
	if err := Check(&bin.Buffer{Buf: b.Buf}, L228, Lim{}); err != nil {
		t.Fatalf("check priv createChat: %v", err)
	}
}
