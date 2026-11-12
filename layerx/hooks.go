package layerx

import (
	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

// putTextStr writes plain text.
func putTextStr(b *bin.Buffer, v tg.TextWithEntities) error {
	b.PutString(v.Text)
	return nil
}

// getTextStr reads plain text.
func getTextStr(b *bin.Buffer) (tg.TextWithEntities, error) {
	s, err := b.String()
	if err != nil {
		return tg.TextWithEntities{}, err
	}
	return tg.TextWithEntities{Text: s}, nil
}
