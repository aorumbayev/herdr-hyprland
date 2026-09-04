package movetab

import "testing"

func TestPlanFindsTabByNumberInWorkspace(t *testing.T) {
	tabs := []Tab{
		{ID: "w1:t1", WorkspaceID: "w1", Number: 1},
		{ID: "w2:t1", WorkspaceID: "w2", Number: 1},
	}
	got := Resolve(tabs, "w1", "1")
	if got.TabID != "w1:t1" || got.NewTabLabel != "" {
		t.Fatalf("Plan = %+v; want existing w1:t1", got)
	}
}

func TestPlanCreatesWhenNumberMissing(t *testing.T) {
	tabs := []Tab{{ID: "w1:t2", WorkspaceID: "w1", Number: 2}}
	got := Resolve(tabs, "w1", "1")
	if got.TabID != "" || got.NewTabLabel != "1" {
		t.Fatalf("Plan = %+v; want new tab labeled 1", got)
	}
}

func TestPlanIgnoresStaleNumberFromOtherWorkspace(t *testing.T) {
	tabs := []Tab{{ID: "w2:t1", WorkspaceID: "w2", Number: 1}}
	got := Resolve(tabs, "w1", "1")
	if got.NewTabLabel != "1" {
		t.Fatalf("Plan = %+v; want new tab in w1", got)
	}
}
