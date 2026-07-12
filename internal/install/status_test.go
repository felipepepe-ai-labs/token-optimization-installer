package install

import "testing"

func TestExtractVersion(t *testing.T) {
	for input, want := range map[string]string{
		"rtk 0.28.2":      "0.28.2",
		"engram v1.17.0":  "1.17.0",
		"version unknown": "",
	} {
		if got := extractVersion(input); got != want {
			t.Fatalf("extractVersion(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	for _, test := range []struct {
		left, right string
		want        int
	}{
		{"1.2.3", "1.2.4", -1},
		{"1.10.0", "1.9.9", 1},
		{"v2.0.0", "2.0.0", 0},
	} {
		if got := compareVersions(test.left, test.right); got != test.want {
			t.Fatalf("compareVersions(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
		}
	}
}

func TestUpdateSelection(t *testing.T) {
	if !(UpdateStatus{}).ShouldSelect() {
		t.Fatal("missing component should be selected")
	}
	if (UpdateStatus{Installed: true, Checked: true}).ShouldSelect() {
		t.Fatal("current component should not be selected")
	}
	if !(UpdateStatus{Installed: true, UpdateAvailable: true}).ShouldSelect() {
		t.Fatal("outdated component should be selected")
	}
}
