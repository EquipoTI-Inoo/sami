package nutrition

import (
	"testing"

	"coachia-backend/models"
)

func sampleFoods() []models.PeruvianFood {
	return []models.PeruvianFood{
		{ID: 1, Name: "Pollo (pechuga a la plancha)", Category: "protein", ServingGrams: 150, Calories: 248, ProteinG: 46.5, Tags: []string{}},
		{ID: 2, Name: "Pescado (bonito o jurel, a la plancha)", Category: "protein", ServingGrams: 150, Calories: 210, ProteinG: 39, Tags: []string{}},
		{ID: 3, Name: "Lentejas cocidas", Category: "legume", ServingGrams: 198, Calories: 230, ProteinG: 18, CarbsG: 40, Tags: []string{"vegetariano", "vegano", "sin_lactosa"}},
		{ID: 4, Name: "Arroz blanco cocido", Category: "carb", ServingGrams: 158, Calories: 205, CarbsG: 45, Tags: []string{"vegetariano", "vegano", "sin_lactosa"}},
		{ID: 5, Name: "Lechuga", Category: "vegetable", ServingGrams: 36, Calories: 5, Tags: []string{"vegetariano", "vegano", "sin_lactosa"}},
		{ID: 6, Name: "Leche entera", Category: "dairy", ServingGrams: 240, Calories: 149, ProteinG: 8, CarbsG: 12, FatG: 8, Tags: []string{"vegetariano", "con_lactosa"}},
		{ID: 7, Name: "Plátano", Category: "fruit", ServingGrams: 118, Calories: 105, CarbsG: 27, Tags: []string{"vegetariano", "vegano", "sin_lactosa"}},
		{ID: 8, Name: "Aceite vegetal", Category: "fat", ServingGrams: 14, Calories: 124, FatG: 14, Tags: []string{"vegetariano", "vegano", "sin_lactosa"}},
	}
}

func TestBuildPlanRespectsDislikes(t *testing.T) {
	target := models.NutritionTargets{Calories: 2000, ProteinG: 150, CarbsG: 200, FatG: 60}
	plan := BuildPlan(sampleFoods(), target, Preferences{Dislikes: []string{"pescado"}})

	for _, meal := range plan.Meals {
		for _, item := range meal.Items {
			if item.Food.Name == "Pescado (bonito o jurel, a la plancha)" {
				t.Fatalf("disliked food %q appeared in the plan", item.Food.Name)
			}
		}
	}
}

func TestBuildPlanRespectsVegetarian(t *testing.T) {
	target := models.NutritionTargets{Calories: 2000, ProteinG: 150, CarbsG: 200, FatG: 60}
	plan := BuildPlan(sampleFoods(), target, Preferences{Restrictions: []string{"vegetariano"}})

	for _, meal := range plan.Meals {
		for _, item := range meal.Items {
			if item.Food.Category == "protein" {
				t.Fatalf("meat/fish item %q appeared despite vegetarian restriction", item.Food.Name)
			}
		}
	}
}

func TestBuildPlanTotalsAreNonNegative(t *testing.T) {
	target := models.NutritionTargets{Calories: 2200, ProteinG: 160, CarbsG: 220, FatG: 70}
	plan := BuildPlan(sampleFoods(), target, Preferences{})

	if plan.Calories <= 0 || plan.ProteinG <= 0 || plan.CarbsG <= 0 {
		t.Fatalf("expected positive totals, got calories=%d protein=%d carbs=%d", plan.Calories, plan.ProteinG, plan.CarbsG)
	}
	if len(plan.Meals) != 3 {
		t.Fatalf("expected 3 meals, got %d", len(plan.Meals))
	}
}

// Regression test for a real bug: liking an available-but-low-protein-density
// food (e.g. lentils) used to be able to outrank an unliked but still-unused
// meat for a "protein" slot, because scaling the liked legume to the full
// meal's protein target ballooned it to 3+ cups and dragged the day's
// calories ~35% over target. Category order must win over "likes" when the
// preferred category still has unused food in it.
func TestBuildPlanStaysWithin25PercentOfCalorieTarget(t *testing.T) {
	target := models.NutritionTargets{Calories: 2156, ProteinG: 156, CarbsG: 248, FatG: 60}
	plan := BuildPlan(sampleFoods(), target, Preferences{
		Likes:    []string{"pollo", "arroz", "lentejas"},
		Dislikes: []string{"pescado"},
	})

	upperBound := float64(target.Calories) * 1.25
	if float64(plan.Calories) > upperBound {
		t.Fatalf("plan calories %d exceed 125%% of target %d (upper bound %.0f)", plan.Calories, target.Calories, upperBound)
	}
}

// Regression test: every plan item must report real grams (servings × the
// food's reference serving_grams), not just an ambiguous "× servings"
// multiplier — a user asked for exact quantities instead of multipliers.
func TestBuildPlanItemsReportTotalGrams(t *testing.T) {
	target := models.NutritionTargets{Calories: 2000, ProteinG: 150, CarbsG: 200, FatG: 60}
	plan := BuildPlan(sampleFoods(), target, Preferences{})

	foodsByID := map[int64]models.PeruvianFood{}
	for _, f := range sampleFoods() {
		foodsByID[f.ID] = f
	}

	for _, meal := range plan.Meals {
		for _, item := range meal.Items {
			food := foodsByID[item.Food.ID]
			wantGrams := int(food.ServingGrams * item.Servings)
			// Allow ±1g for rounding from math.Round on either side.
			if diff := item.TotalGrams - wantGrams; diff < -1 || diff > 1 {
				t.Errorf("%s: expected ~%dg for %v servings of a %gg serving, got %dg",
					item.Food.Name, wantGrams, item.Servings, food.ServingGrams, item.TotalGrams)
			}
			if item.TotalGrams <= 0 {
				t.Errorf("%s: expected positive total_grams, got %d", item.Food.Name, item.TotalGrams)
			}
		}
	}
}
