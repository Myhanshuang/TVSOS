package sandbox

import "testing"

func TestBudgetsMergeBehavior(t *testing.T) {
		tomlText := `
[objectives]
mode = "primary"
primary = "U2"
tolerance = 0.2
budgets = { U4_max = 0.3 }
`
	parsed, err := ParseExperimentParams(tomlText)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	t.Logf("parsed budgets: %v", parsed.Objectives.Budgets)
	t.Logf("parsed mode=%s primary=%s tol=%v", parsed.Objectives.NormalizedMode(), parsed.Objectives.NormalizedPrimary(), parsed.Objectives.Tolerance)
	t.Logf("effective budgets: %v", parsed.Objectives.EffectiveBudgets())

	// 旧版 TOML 兼容
	oldText := `
[objectives]
weights = { C2 = 1.0, U2 = 0.5 }
soft_constraints = { F1_gini_max = 0.4 }
`
	p2, err := ParseExperimentParams(oldText)
	if err != nil {
		t.Fatalf("parse old: %v", err)
	}
	t.Logf("old parsed: mode=%s primary=%s effective=%v", p2.Objectives.NormalizedMode(), p2.Objectives.NormalizedPrimary(), p2.Objectives.EffectiveBudgets())

	// 显式关闭 F1
	p3, err := ParseExperimentParams("[objectives]\nbudgets = { F1_gini_max = -1 }\n")
	if err != nil {
		t.Fatalf("parse disable: %v", err)
	}
	t.Logf("disable F1 effective: %v", p3.Objectives.EffectiveBudgets())
	if _, ok := p3.Objectives.EffectiveBudgets()["F1_gini_max"]; ok {
		t.Fatal("F1 预算应被显式关闭")
	}
}

