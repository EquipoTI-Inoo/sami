package nutrition

import (
	"math"
	"strings"

	"coachia-backend/models"
)

// restrictionTags maps a keyword a user might type to the tag a compatible
// food must carry. Matching is substring-based so "vegetariano", "vegetariana"
// and "soy vegetariano" all hit the same rule.
var restrictionTags = map[string]string{
	"vegan":      "vegano",
	"vegetarian": "vegetariano",
	"lactosa":    "sin_lactosa",
	"lacteo":     "sin_lactosa",
	"dairy":      "sin_lactosa",
}

type Preferences struct {
	Likes        []string
	Dislikes     []string
	Restrictions []string
}

type mealSlot struct {
	name       string
	categories []string // tried in order; first category with an available food wins
}

type mealDef struct {
	name  string
	ratio float64
	slots []mealSlot
}

var meals = []mealDef{
	{
		name:  "Desayuno",
		ratio: 0.25,
		slots: []mealSlot{
			{"lácteo o proteína", []string{"dairy", "protein"}},
			{"carbohidrato", []string{"carb"}},
			{"fruta", []string{"fruit"}},
		},
	},
	{
		name:  "Almuerzo",
		ratio: 0.40,
		slots: []mealSlot{
			{"proteína", []string{"protein", "legume"}},
			{"carbohidrato", []string{"carb"}},
			{"vegetal", []string{"vegetable"}},
		},
	},
	{
		name:  "Cena",
		ratio: 0.35,
		slots: []mealSlot{
			{"proteína", []string{"protein", "legume"}},
			{"carbohidrato", []string{"carb"}},
			{"vegetal", []string{"vegetable"}},
		},
	},
}

// BuildPlan picks real foods from the catalog to approximate the day's
// nutrition targets, respecting likes/dislikes/restrictions. It is a greedy
// heuristic, not an optimizer — totals are computed from whatever was
// actually picked, so the response is always internally consistent even when
// it doesn't land exactly on target.
func BuildPlan(foods []models.PeruvianFood, target models.NutritionTargets, prefs Preferences) models.MealPlan {
	available := filterFoods(foods, prefs.Dislikes, prefs.Restrictions)
	byCategory := groupByCategory(available)
	usedProtein := map[int64]bool{}

	plan := models.MealPlan{Target: target}

	for _, def := range meals {
		mealCalories := float64(target.Calories) * def.ratio
		mealProtein := float64(target.ProteinG) * def.ratio
		mealCarbs := float64(target.CarbsG) * def.ratio
		mealFat := float64(target.FatG) * def.ratio

		meal := models.Meal{Name: def.name}

		for _, slot := range def.slots {
			food, ok := pickFood(byCategory, slot.categories, prefs.Likes, usedProtein)
			if !ok {
				continue
			}

			var servings float64
			switch food.Category {
			case "protein", "legume":
				servings = safeServings(mealProtein, food.ProteinG, mealCalories, food.Calories)
				usedProtein[food.ID] = true
			case "carb":
				servings = safeServings(mealCarbs, food.CarbsG, mealCalories, food.Calories)
			default: // dairy, vegetable, fruit, fat — a fixed, sensible single portion
				servings = 1
			}

			item := toMealItem(food, servings)
			meal.Items = append(meal.Items, item)
			meal.Calories += item.Calories
			meal.ProteinG += item.ProteinG
			meal.CarbsG += item.CarbsG
			meal.FatG += item.FatG
		}

		// Top up fat only if the meal is meaningfully short of its fat target —
		// protein/carb sources already carry some fat, so this avoids doubling up.
		fatGap := mealFat - float64(meal.FatG)
		if fatGap > 5 {
			if fatFood, ok := pickFood(byCategory, []string{"fat"}, prefs.Likes, map[int64]bool{}); ok {
				servings := safeServings(fatGap, fatFood.FatG, mealCalories, fatFood.Calories)
				item := toMealItem(fatFood, servings)
				meal.Items = append(meal.Items, item)
				meal.Calories += item.Calories
				meal.ProteinG += item.ProteinG
				meal.CarbsG += item.CarbsG
				meal.FatG += item.FatG
			}
		}

		plan.Meals = append(plan.Meals, meal)
		plan.Calories += meal.Calories
		plan.ProteinG += meal.ProteinG
		plan.CarbsG += meal.CarbsG
		plan.FatG += meal.FatG
	}

	return plan
}

func toMealItem(food models.PeruvianFood, servings float64) models.MealItem {
	return models.MealItem{
		Food:       food,
		Servings:   servings,
		TotalGrams: int(math.Round(food.ServingGrams * servings)),
		Calories:   int(math.Round(food.Calories * servings)),
		ProteinG:   int(math.Round(food.ProteinG * servings)),
		CarbsG:     int(math.Round(food.CarbsG * servings)),
		FatG:       int(math.Round(food.FatG * servings)),
	}
}

// safeServings scales to roughly hit a per-item macro target, rounded to the
// nearest half-serving, clamped to a human-reasonable range.
// maxShareOfMealCalories caps how much of a single meal's calorie budget one
// item may fill on its own. Without this, a low-density food (legumes are the
// clearest case — lots of carbs, not much protein per cup) scaled up to hit
// its one target macro can blow through the whole meal's calories by itself.
const maxShareOfMealCalories = 0.65

func safeServings(targetMacro, macroPerServing, mealCalories, caloriesPerServing float64) float64 {
	if macroPerServing <= 0 {
		return 1
	}
	raw := targetMacro / macroPerServing

	if caloriesPerServing > 0 {
		maxByCalories := (mealCalories * maxShareOfMealCalories) / caloriesPerServing
		raw = math.Min(raw, maxByCalories)
	}

	rounded := math.Round(raw*2) / 2
	return math.Max(0.5, math.Min(3, rounded))
}

func groupByCategory(foods []models.PeruvianFood) map[string][]models.PeruvianFood {
	byCategory := make(map[string][]models.PeruvianFood)
	for _, f := range foods {
		byCategory[f.Category] = append(byCategory[f.Category], f)
	}
	return byCategory
}

// pickFood tries each category in order and returns the first liked food not
// already used; categories are a priority order, not an open pool — "legume"
// only gets considered once every "protein" item has been used elsewhere in
// the day. Without that, a liked legume (lower protein density, so it scales
// to a huge portion to hit the same protein target) could outrank an unliked
// but still-available meat, and the day's totals would blow past the target.
func pickFood(byCategory map[string][]models.PeruvianFood, categories []string, likes []string, used map[int64]bool) (models.PeruvianFood, bool) {
	for _, cat := range categories {
		unused := unusedIn(byCategory[cat], used)
		if len(unused) == 0 {
			continue // this category is exhausted — fall through to the next
		}
		for _, f := range unused {
			if matchesAny(f.Name, likes) {
				return f, true
			}
		}
		return unused[0], true // no liked match here, but this category still wins over the next
	}

	// Every category in the list is fully used already — repeat rather than
	// leave the slot empty.
	for _, cat := range categories {
		if len(byCategory[cat]) > 0 {
			return byCategory[cat][0], true
		}
	}
	return models.PeruvianFood{}, false
}

func unusedIn(foods []models.PeruvianFood, used map[int64]bool) []models.PeruvianFood {
	var result []models.PeruvianFood
	for _, f := range foods {
		if !used[f.ID] {
			result = append(result, f)
		}
	}
	return result
}

func filterFoods(foods []models.PeruvianFood, dislikes, restrictions []string) []models.PeruvianFood {
	requiredTags := requiredTagsFor(restrictions)

	var result []models.PeruvianFood
	for _, f := range foods {
		if matchesAny(f.Name, dislikes) {
			continue
		}
		if !hasAllTags(f.Tags, requiredTags) {
			continue
		}
		result = append(result, f)
	}
	return result
}

func requiredTagsFor(restrictions []string) []string {
	var tags []string
	for _, r := range restrictions {
		r = strings.ToLower(r)
		for keyword, tag := range restrictionTags {
			if strings.Contains(r, keyword) {
				tags = append(tags, tag)
			}
		}
	}
	return tags
}

func hasAllTags(foodTags, required []string) bool {
	for _, req := range required {
		found := false
		for _, t := range foodTags {
			if t == req {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func matchesAny(name string, keywords []string) bool {
	name = strings.ToLower(name)
	for _, k := range keywords {
		k = strings.ToLower(strings.TrimSpace(k))
		if k != "" && strings.Contains(name, k) {
			return true
		}
	}
	return false
}
