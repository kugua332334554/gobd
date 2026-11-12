package gen

import (
	"path/filepath"
	"sort"
	"testing"
)

func loadTestUni(t *testing.T) *Uni {
	t.Helper()
	u, err := LoadUni(
		filepath.Join("..", "_schema", "layers", "meta.json"),
		filepath.Join("..", "_schema", "layers", "decide.json"),
		filepath.Join("..", "_schema", "layers", "priv.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestLayWhy(t *testing.T) {
	u := loadTestUni(t)
	l := u.LayerOf(228)
	r := u.Classify(l)
	byRef := map[string]int{}
	byKind := map[DiffKind]int{}
	for key, k := range r.Kinds {
		if k != Relay {
			continue
		}
		for _, ref := range r.Refs[key] {
			rk, ok := r.Kinds[ref]
			if !ok {
				rk = Gone
			}
			if rk == Same {
				continue
			}
			byKind[rk]++
			byRef[ref]++
			break
		}
	}
	t.Logf("first bad ref kinds: %v", byKind)
	type kv struct {
		k string
		n int
	}
	var top []kv
	for k, n := range byRef {
		top = append(top, kv{k, n})
	}
	sort.Slice(top, func(i, j int) bool { return top[i].n > top[j].n })
	for i, e := range top {
		if i >= 25 {
			break
		}
		t.Logf("  %s -> %d", e.k, e.n)
	}
}

func TestLayShapes(t *testing.T) {
	u := loadTestUni(t)
	l := u.LayerOf(228)
	r := u.Classify(l)
	for _, key := range []string{"user", "channel", "message", "chatFull", "username", "pageBlockTable", "chatAdminRights", "keyboardButton", "channels.toggleForum", "payments.getStarGiftWithdrawalUrl", "botVerification", "botVerifierSettings", "bots.setCustomVerification"} {
		can := u.CanIdx.byKey[key]
		lay, ok := l.Idx.Get(key)
		if !ok {
			t.Logf("%s: absent", key)
			continue
		}
		t.Logf("%s: kind=%v\n  can=%s\n  lay=%s", key, r.Kinds[key], defShape(can), defShape(lay))
		a, b := defShape(can), defShape(lay)
		if a != b {
			n := len(a)
			if len(b) < n {
				n = len(b)
			}
			i := 0
			for i < n && a[i] == b[i] {
				i++
			}
			lo := i - 30
			if lo < 0 {
				lo = 0
			}
			t.Logf("  diff at %d: can=%q lay=%q", i, a[lo:min(len(a), i+40)], b[lo:min(len(b), i+40)])
		}
	}
}

func TestLayClassify(t *testing.T) {
	u := loadTestUni(t)
	t.Logf("%s", u)
	for _, l := range u.Layers {
		r := u.Classify(l)
		c := r.Count()
		t.Logf("layer %d: same=%d id=%d body=%d relay=%d none=%d hook=%d",
			l.Meta.N, c[Same], c[Id], c[Body], c[Relay], c[None], c[Hook])
		if c[Hook] > 0 {
			for _, key := range r.KindsOf(Hook) {
				can := u.CanIdx.byKey[key]
				lay, _ := l.Idx.Get(key)
				t.Logf("  hook %s:\n    can=%s\n    lay=%s", key, defShape(can), defShape(lay))
			}
		}
	}
}
