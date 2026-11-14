# gobd

private fork of [gotd/td](https://github.com/gotd/td) used by the baigram server.

it adds `layerx`, a multi-layer tl runtime that encodes and decodes telegram
objects for layers 228-233, so one server can serve clients speaking different
layers.

## usage

the server consumes this fork through a local replace directive:

```go
replace github.com/gotd/td => ../gotd
```

to use it as a normal dependency:

```console
go get github.com/kugua332334554/gobd
```

minimal client:

```go
client := telegram.NewClient(appID, appHash, telegram.Options{})
err := client.Run(ctx, func(ctx context.Context) error {
	api := client.API()
	// invoke mtproto methods here
	return nil
})
```

## layout

- `bin` - binary encoding and decoding primitives
- `tg` - generated tl types for the canonical layer
- `layerx` - per-layer tables, projections and generated codecs (`lx_*_gen.go`)
- `mtproto` - mtproto 2.0 sessions, rpc and transport glue
- `telegram` - high level client: auth, uploads, downloads, messages
- `_schema` - tl sources: telegram, layers 228-233, private ctors
- `gen` - generator templates, `cmd/gotdgen` - schema code generator
- `examples` - runnable client examples

## development

```console
make generate          # regenerate tg and layerx from _schema
make test              # go test -race ./...
make check_generated   # fails on stale generated code
```

github actions are disabled on this fork; run checks locally.

see `ARCHITECTURE.md` and `CONTRIBUTING.md` for details.
