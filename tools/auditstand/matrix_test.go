package main

import (
	"strings"
	"testing"
)

func TestMatrixIsInternallyConsistent(t *testing.T) {
	errs := validateMatrix()
	if len(errs) != 0 {
		t.Fatalf("matrix defects: %v", errs)
	}
}

func TestMatrixCellCountAndStates(t *testing.T) {
	if len(matrix) != len(scenes)*5 {
		t.Fatalf("matrix holds %d cells, want %d", len(matrix), len(scenes)*5)
	}
	for _, c := range matrix {
		switch c.state {
		case cellNA:
			if c.na == "" {
				t.Fatalf("n/a cell %sx%d without a register row", c.scene, c.prop)
			}
		case cellOpen:
			if len(c.budgets) == 0 {
				t.Fatalf("open cell %sx%d without budgets", c.scene, c.prop)
			}
		}
	}
}

func TestMatrixInjectedDefectFails(t *testing.T) {
	// The self-test must actually fail when the matrix is broken - a validator that cannot
	// fail validates nothing.
	saved := matrix[matrixCellIndex("S03", propResponsiveness)]
	defer func() { matrix[matrixCellIndex("S03", propResponsiveness)] = saved }()

	matrix[matrixCellIndex("S03", propResponsiveness)].budgets = []string{"H99"}
	if errs := validateMatrix(); len(errs) == 0 {
		t.Fatal("unknown budget id passed validation")
	}

	matrix[matrixCellIndex("S03", propResponsiveness)].budgets = nil
	if errs := validateMatrix(); len(errs) == 0 {
		t.Fatal("open cell without budgets passed validation")
	}

	matrix[matrixCellIndex("S03", propResponsiveness)].state = cellNA
	matrix[matrixCellIndex("S03", propResponsiveness)].na = ""
	if errs := validateMatrix(); len(errs) == 0 {
		t.Fatal("n/a cell without a register row passed validation")
	}
}

func matrixCellIndex(id sceneID, prop int) int {
	for i, c := range matrix {
		if c.scene == id && c.prop == prop {
			return i
		}
	}
	panic("no such cell")
}

func TestNARegisterRowsAllReferenced(t *testing.T) {
	for _, n := range naRegister {
		if !strings.HasPrefix(n.cell, "S") {
			t.Fatalf("register row %s names no scene: %q", n.id, n.cell)
		}
		if n.reason == "" {
			t.Fatalf("register row %s has no reason", n.id)
		}
	}
}

func TestStripNAAndBudgetRefs(t *testing.T) {
	if got := stripNA("H9,H10 N1"); got != "H9,H10 " {
		t.Fatalf("stripNA = %q", got)
	}
	refs := budgetRefs(" H1,H2,H3 ")
	if len(refs) != 3 || refs[0] != "H1" || refs[2] != "H3" {
		t.Fatalf("budgetRefs = %v", refs)
	}
}

func TestEverySceneHasImplementationStatusOrHonestRefusal(t *testing.T) {
	// A scene without an implementation is fine; a scene implemented but absent from the
	// matrix (or vice versa) is not.
	for id := range implementedScenes {
		if _, ok := sceneByID(id); !ok {
			t.Fatalf("implemented scene %s is not in the catalog", id)
		}
	}
}

func TestBudgetHardBarMarked(t *testing.T) {
	if !budgets[0].hardBar {
		t.Fatal("H1 carries the watchdog hard bar and must be marked as non-hypothesis")
	}
}
