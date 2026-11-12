package gen

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-faster/errors"
)

// Decision is one reviewed difference.
type Decision struct {
	Key   string `json:"key"`
	Do    string `json:"do"`
	Field string `json:"field,omitempty"`
	Conv  string `json:"conv,omitempty"`
	Val   string `json:"val,omitempty"`
}

// Decide is decide.json.
type Decide struct {
	Ver   int        `json:"ver"`
	Items []Decision `json:"items"`

	byLayer map[Layer]map[string][]Decision
}

// LoadDecide reads decisions, missing file is empty.
func LoadDecide(path string) (*Decide, error) {
	d := &Decide{Ver: 1}
	if path == "" {
		d.build()
		return d, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			d.build()
			return d, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, d); err != nil {
		return nil, errors.Wrap(err, "decide.json")
	}
	d.build()
	return d, nil
}

func (d *Decide) build() {
	d.byLayer = map[Layer]map[string][]Decision{}
	for _, it := range d.Items {
		l, key, ok := splitDecideKey(it.Key)
		if !ok {
			continue
		}
		m := d.byLayer[l]
		if m == nil {
			m = map[string][]Decision{}
			d.byLayer[l] = m
		}
		m[key] = append(m[key], it)
	}
}

func splitDecideKey(key string) (Layer, string, bool) {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "l") {
		return 0, "", false
	}
	n, err := strconv.Atoi(parts[0][1:])
	if err != nil {
		return 0, "", false
	}
	return Layer(n), parts[1], true
}

// At returns decisions for one definition.
func (d *Decide) At(l Layer, key string) []Decision {
	if d == nil {
		return nil
	}
	return d.byLayer[l][key]
}

// Skip reports unavailable definition.
func (d *Decide) Skip(l Layer, key string) bool {
	for _, it := range d.At(l, key) {
		if it.Do == "skip" {
			return true
		}
	}
	return false
}

// Conv returns field conversion.
func (d *Decide) Conv(l Layer, key, field string) (string, bool) {
	for _, it := range d.At(l, key) {
		if it.Do == "hook" && it.Field == field && it.Conv != "" {
			return it.Conv, true
		}
	}
	return "", false
}

// Mask reports silent drop.
func (d *Decide) Mask(l Layer, key, field string) bool {
	for _, it := range d.At(l, key) {
		if it.Do == "mask" && (it.Field == field || it.Field == "") {
			return true
		}
	}
	return false
}

// unknown reports unsupported action.
func (d *Decide) unknown() error {
	for _, it := range d.Items {
		switch it.Do {
		case "keep", "fill", "mask", "hook", "skip":
		default:
			return fmt.Errorf("decide %s: unknown action %q", it.Key, it.Do)
		}
	}
	return nil
}
