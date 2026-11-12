package layerx

// Lim bounds decoding work.
type Lim struct {
	Bytes int
	Vec   int
	Agg   int
	Depth int
}

// Default returns conservative limits.
func (l Lim) Default() Lim {
	if l.Bytes <= 0 {
		l.Bytes = 1 << 20
	}
	if l.Vec <= 0 {
		l.Vec = 1 << 16
	}
	if l.Agg <= 0 {
		l.Agg = 1 << 18
	}
	if l.Depth <= 0 {
		l.Depth = 32
	}
	return l
}

// Opts controls admission.
type Opts struct {
	Lim   Lim
	Unzip Unzip
}

// Unzip expands compressed payloads.
type Unzip func([]byte) ([]byte, error)
