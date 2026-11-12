package gen

import (
	"encoding/json"
	"os"

	"github.com/go-faster/errors"
)

// layerAudit is one layer report entry.
type layerAudit struct {
	N     Layer          `json:"n"`
	Count map[string]int `json:"count"`
	Hooks []string       `json:"hooks,omitempty"`
	Units int            `json:"units"`
}

// auditFile is the audit document.
type auditFile struct {
	Canon  Layer        `json:"canon"`
	Units  int          `json:"units"`
	Bytes  int          `json:"bytes"`
	Lines  int          `json:"lines"`
	Priv   int          `json:"priv"`
	PrivDC int          `json:"privDecodable"`
	Layers []layerAudit `json:"layers"`
}

// WriteAudit writes audit json.
func (l *LayGen) WriteAudit(path string) error {
	l.buildUnits()
	nBytes, nLines := l.Stat()
	doc := auditFile{
		Canon:  l.U.Canon,
		Units:  len(l.custom),
		Bytes:  nBytes,
		Lines:  nLines,
		Priv:   len(l.U.Priv.Items),
		PrivDC: len(l.U.Priv.Named()),
	}
	for _, spec := range l.U.Layers {
		r := l.reports[spec.Meta.N]
		c := map[string]int{}
		for k, v := range r.Count() {
			c[k.String()] = v
		}
		doc.Layers = append(doc.Layers, layerAudit{
			N:     spec.Meta.N,
			Count: c,
			Hooks: r.KindsOf(Hook),
			Units: len(l.layerUnits[spec.Meta.N]),
		})
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshal audit")
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0600)
}
