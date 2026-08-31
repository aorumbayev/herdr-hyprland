package lasttab

import "testing"

// Hyprland's `workspace previous` jumps to the workspace you were on before
// the current one. Here: the tab focused before the current tab, per herdr
// workspace.
func TestLastReturnsTheTabFocusedBeforeTheCurrentOne(t *testing.T) {
	s := State{}
	s = Observe(s, "w1", "w1:t1")
	s = Observe(s, "w1", "w1:t2")

	got, ok := Last(s, "w1")
	if !ok || got != "w1:t1" {
		t.Fatalf("Last = %q, %v; want %q, true", got, ok, "w1:t1")
	}
}

func TestRefocusingTheCurrentTabChangesNothing(t *testing.T) {
	s := State{}
	s = Observe(s, "w1", "w1:t1")
	s = Observe(s, "w1", "w1:t2")
	s = Observe(s, "w1", "w1:t2")

	got, _ := Last(s, "w1")
	if got != "w1:t1" {
		t.Fatalf("Last = %q; want %q", got, "w1:t1")
	}
}

func TestFocusHistoryMovesForward(t *testing.T) {
	s := State{}
	s = Observe(s, "w1", "w1:t1")
	s = Observe(s, "w1", "w1:t2")
	s = Observe(s, "w1", "w1:t3")

	got, _ := Last(s, "w1")
	if got != "w1:t2" {
		t.Fatalf("Last = %q; want %q", got, "w1:t2")
	}
}

func TestNoHistoryMeansNoLastTab(t *testing.T) {
	s := State{}
	s = Observe(s, "w1", "w1:t1")

	if _, ok := Last(s, "w1"); ok {
		t.Fatal("Last = ok on a workspace with a single focused tab; want not ok")
	}
	if _, ok := Last(s, "w2"); ok {
		t.Fatal("Last = ok on a never-seen workspace; want not ok")
	}
}

func TestWorkspacesKeepSeparateHistories(t *testing.T) {
	s := State{}
	s = Observe(s, "w1", "w1:t1")
	s = Observe(s, "w1", "w1:t2")
	s = Observe(s, "w2", "w2:t9")
	s = Observe(s, "w2", "w2:t8")

	if got, _ := Last(s, "w1"); got != "w1:t1" {
		t.Fatalf("Last(w1) = %q; want %q", got, "w1:t1")
	}
	if got, _ := Last(s, "w2"); got != "w2:t9" {
		t.Fatalf("Last(w2) = %q; want %q", got, "w2:t9")
	}
}
