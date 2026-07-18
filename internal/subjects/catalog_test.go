package subjects

import (
	"testing"
)

func TestLookup_PushSwapSeed(t *testing.T) {
	t.Parallel()
	if got := Lookup("42next-push_swap"); got != 193464 {
		t.Fatalf("Lookup(42next-push_swap) = %d, want 193464", got)
	}
	if got := Lookup("missing"); got != 0 {
		t.Fatalf("Lookup(missing) = %d, want 0", got)
	}
}

func TestParse_SkipsInvalid(t *testing.T) {
	t.Parallel()
	idx, err := Parse([]byte(`{"ok": 1, "": 2, "bad": 0, "neg": -1}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(idx) != 1 || idx["ok"] != 1 {
		t.Fatalf("got %#v", idx)
	}
}

func TestMerge(t *testing.T) {
	t.Parallel()
	dst := Index{"a": 1, "b": 2}
	added, updated := Merge(dst, Index{"b": 9, "c": 3, "": 1, "d": 0})
	if added != 1 || updated != 1 {
		t.Fatalf("added=%d updated=%d", added, updated)
	}
	if dst["b"] != 9 || dst["c"] != 3 || dst["a"] != 1 {
		t.Fatalf("dst=%#v", dst)
	}
}

func TestEmbedded_IsCopy(t *testing.T) {
	t.Parallel()
	cp := Embedded()
	cp["42next-push_swap"] = 1
	if Lookup("42next-push_swap") != 193464 {
		t.Fatal("Embedded mutou o catálogo embutido")
	}
}

func TestMergeAbsent(t *testing.T) {
	t.Parallel()
	dst := Index{"a": 1}
	added := MergeAbsent(dst, Index{"a": 9, "b": 2})
	if added != 1 || dst["a"] != 1 || dst["b"] != 2 {
		t.Fatalf("dst=%#v added=%d", dst, added)
	}
}

func TestMatchSlug(t *testing.T) {
	t.Parallel()
	if got := MatchSlug("42next-push_swap"); got != "42next-push_swap" {
		t.Fatalf("got %q", got)
	}
	if got := MatchSlug("missing-xyz"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestCompletionNames(t *testing.T) {
	t.Parallel()
	names := CompletionNames()
	if len(names) < 100 {
		t.Fatalf("poucos nomes: %d", len(names))
	}
	var hasSlug, hasShort bool
	for _, n := range names {
		if n == "42next-push_swap" {
			hasSlug = true
		}
		if n == "push_swap" {
			hasShort = true
		}
	}
	if !hasSlug || !hasShort {
		t.Fatalf("slug=%v short=%v", hasSlug, hasShort)
	}
}
