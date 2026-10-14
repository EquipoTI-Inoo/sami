package nutrition

import "testing"

func TestCalculateMaintainMale(t *testing.T) {
	// BMR = 10*80 + 6.25*180 - 5*30 + 5 = 800 + 1125 - 150 + 5 = 1780
	// TDEE = 1780 * 1.55 (moderate) = 2759
	targets := Calculate("male", 30, 180, 80, "moderate", "maintain")

	if targets.Calories != 2759 {
		t.Errorf("expected 2759 calories, got %d", targets.Calories)
	}
	if targets.ProteinG != 160 { // 2g/kg * 80kg
		t.Errorf("expected 160g protein, got %d", targets.ProteinG)
	}
}

func TestCalculateCutAppliesDeficit(t *testing.T) {
	maintain := Calculate("female", 28, 165, 65, "light", "maintain")
	cut := Calculate("female", 28, 165, 65, "light", "cut")

	if cut.Calories >= maintain.Calories {
		t.Errorf("expected cut (%d) to be below maintain (%d)", cut.Calories, maintain.Calories)
	}
}

func TestCalculateBulkAppliesSurplus(t *testing.T) {
	maintain := Calculate("male", 25, 175, 70, "active", "maintain")
	bulk := Calculate("male", 25, 175, 70, "active", "bulk")

	if bulk.Calories <= maintain.Calories {
		t.Errorf("expected bulk (%d) to be above maintain (%d)", bulk.Calories, maintain.Calories)
	}
}

func TestCalculateNeverReturnsNegativeCarbs(t *testing.T) {
	// A very light, low-activity cut can push protein+fat kcal close to total —
	// carbs must clamp at zero rather than go negative.
	targets := Calculate("female", 60, 150, 40, "sedentary", "cut")

	if targets.CarbsG < 0 {
		t.Errorf("expected non-negative carbs, got %d", targets.CarbsG)
	}
}

// Regression test for a real bug: the raw Mifflin-St Jeor formula can land
// well under 1000 kcal/day for a small, older, low-activity profile on a
// cut — unsafe to recommend without medical supervision. This exact
// profile computed to 841 kcal before the floor was added.
func TestCalculateNeverGoesBelowSafeFloor(t *testing.T) {
	targets := Calculate("female", 60, 150, 40, "sedentary", "cut")
	if targets.Calories < 1200 {
		t.Errorf("expected at least the 1200 kcal female floor, got %d", targets.Calories)
	}

	maleTargets := Calculate("male", 70, 160, 50, "sedentary", "cut")
	if maleTargets.Calories < 1500 {
		t.Errorf("expected at least the 1500 kcal male floor, got %d", maleTargets.Calories)
	}
}

// Regression test for a real bug: for a high enough body weight relative to
// a tight calorie target, protein (2g/kg of raw body weight) plus the fixed
// 25% fat allocation could together exceed the whole calorie budget,
// flooring carbs at zero while the reported macros silently summed to more
// than the stated calorie target. Macros must always sum to Calories.
func TestCalculateMacrosAlwaysSumToCalories(t *testing.T) {
	cases := []struct {
		sex                 string
		age                 int
		heightCm, weightKg  float64
		activityLevel, goal string
	}{
		{"female", 90, 140, 300, "sedentary", "cut"},
		{"male", 80, 150, 280, "sedentary", "cut"},
		{"female", 60, 150, 40, "sedentary", "cut"},
		{"male", 30, 180, 80, "active", "bulk"},
	}

	for _, tc := range cases {
		targets := Calculate(tc.sex, tc.age, tc.heightCm, tc.weightKg, tc.activityLevel, tc.goal)
		macroCalories := targets.ProteinG*4 + targets.CarbsG*4 + targets.FatG*9
		// Rounding each macro to whole grams can drift the sum by a couple
		// kcal either way — allow a small tolerance, not open-ended drift.
		diff := macroCalories - targets.Calories
		if diff < -5 || diff > 5 {
			t.Errorf("%+v: macros sum to %d kcal, target is %d kcal (diff %d)", tc, macroCalories, targets.Calories, diff)
		}
	}
}
