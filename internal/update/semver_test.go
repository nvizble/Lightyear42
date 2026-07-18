package update

import "testing"

func TestIsNewer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		current, latest string
		newer           bool
		wantErr         bool
	}{
		{"v1.0.0", "v1.0.1", true, false},
		{"1.0.1", "v1.0.1", false, false},
		{"v1.0.2", "v1.0.1", false, false},
		{"dev", "v1.0.2", false, true},
		{"v1.0.0", "not-a-version", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.current+"->"+tt.latest, func(t *testing.T) {
			t.Parallel()
			got, err := IsNewer(tt.current, tt.latest)
			if tt.wantErr {
				if err == nil {
					t.Fatal("esperava erro")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.newer {
				t.Fatalf("newer = %v, want %v", got, tt.newer)
			}
		})
	}
}

func TestIsReleaseVersion(t *testing.T) {
	t.Parallel()

	if !IsReleaseVersion("v1.0.2") || !IsReleaseVersion("1.0.2") {
		t.Fatal("release válido rejeitado")
	}
	if IsReleaseVersion("dev") || IsReleaseVersion("") {
		t.Fatal("dev deveria ser inválido")
	}
}
