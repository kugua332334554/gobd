// Package td implements MTProto encoding and decoding.
package td

//go:generate go run github.com/gotd/td/cmd/gotdgen --doc "https://core.telegram.org/" --clean --server --handlers --mapping --slices --package tg --target tg --schema _schema/telegram.tl
//go:generate go run github.com/gotd/td/cmd/gotdgen --doc "https://core.telegram.org/" --clean --package e2e --target tg/e2e --schema _schema/encrypted.tl

//go:generate go run github.com/gotd/td/cmd/gotdgen --clean --client=false --package tgtrace --target tgtrace --schema _schema/trace.tl

//go:generate go run github.com/gotd/td/cmd/gotdgen --clean --package mt --target mt --client=false --schema _schema/mt.tl

//go:generate go run github.com/gotd/td/cmd/gotdgen --schema _schema/telegram.tl --layers-meta _schema/layers/meta.json --layers-decide _schema/layers/decide.json --layers-priv _schema/layers/priv.json --layers-target layerx --layers-package layerx --layers-audit _schema/layers/audit.json --layers-only
