package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompleteSubjectProjects(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	got, dir := completeSubjectProjects(nil, nil, "push")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive=%v", dir)
	}
	foundShort := false
	for _, name := range got {
		if name == "push_swap" {
			foundShort = true
		}
		if !strings.HasPrefix(strings.ToLower(name), "push") {
			t.Fatalf("não filtra prefixo: %q", name)
		}
	}
	if !foundShort {
		t.Fatalf("esperava alias push_swap, got %v", got)
	}

	gotSlug, _ := completeSubjectProjects(nil, nil, "42next-push")
	foundSlug := false
	for _, name := range gotSlug {
		if name == "42next-push_swap" {
			foundSlug = true
		}
	}
	if !foundSlug {
		t.Fatalf("esperava slug 42next-push_swap, got %v", gotSlug)
	}

	got2, _ := completeSubjectProjects(nil, []string{"already"}, "push")
	if len(got2) != 0 {
		t.Fatalf("com args preenchidos não deve sugerir: %v", got2)
	}
}
