package nutrition

import "math"

// activityMultipliers follows the standard Mifflin-St Jeor activity factors.
var activityMultipliers = map[string]float64{
	"sedentary":   1.2,
	"light":       1.375,
	"moderate":    1.55,
	"active":      1.725,
	"very_active": 1.9,
}

// goalAdjustments express the target as a fraction of TDEE: a 20% deficit to
// cut, a 10% surplus to bulk. Performance gets a small surplus to fuel training
// without the fat-gain pace of a full bulk.
var goalAdjustments = map[string]float64{
	"cut":         0.80,
	"maintain":    1.00,
	"bulk":        1.10,
	"performance": 1.05,
}

const (
	proteinGPerKg      = 2.0 // evidence-based target for training adults, regardless of goal
	fatCalorieFraction = 0.25

	// No deficit should ever be recommended below these without medical
	// supervision — commonly cited clinical floors (e.g. NHS, dietitian
	// guidance). A tiny/old/low-activity profile combined with "cut" can
	// otherwise compute well under 1000 kcal/day from the raw formula.
	minSafeCaloriesFemale = 1200.0
	minSafeCaloriesMale   = 1500.0
)

type Targets struct {
	Calories int
	ProteinG int
	CarbsG   int
	FatG     int
}

// Calculate derives daily calorie and macro targets from Mifflin-St Jeor BMR,
// an activity multiplier, and a goal-based surplus/deficit. weightKg and
// heightCm are the user's current values from onboarding.
func Calculate(sex string, age int, heightCm, weightKg float64, activityLevel, goal string) Targets {
	bmr := 10*weightKg + 6.25*heightCm - 5*float64(age)
	minSafeCalories := minSafeCaloriesFemale
	if sex == "male" {
		bmr += 5
		minSafeCalories = minSafeCaloriesMale
	} else {
		bmr -= 161
	}

	tdee := bmr * activityMultipliers[activityLevel]
	calories := math.Max(tdee*goalAdjustments[goal], minSafeCalories)

	proteinKcal := proteinGPerKg * weightKg * 4
	fatKcal := calories * fatCalorieFraction

	// A high body weight relative to a tight calorie target (e.g. an
	// aggressive cut) can push protein+fat alone past the whole budget —
	// scale both down proportionally rather than let carbs go negative or
	// report macros that don't actually sum to the stated calories.
	if proteinKcal+fatKcal > calories {
		scale := calories / (proteinKcal + fatKcal)
		proteinKcal *= scale
		fatKcal *= scale
	}

	carbKcal := math.Max(0, calories-proteinKcal-fatKcal)

	return Targets{
		Calories: int(math.Round(calories)),
		ProteinG: int(math.Round(proteinKcal / 4)),
		CarbsG:   int(math.Round(carbKcal / 4)),
		FatG:     int(math.Round(fatKcal / 9)),
	}
}
