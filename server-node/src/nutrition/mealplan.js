// Ported 1:1 from backend/nutrition/mealplan.go — same heuristic, same order.

const restrictionTags = { vegan: 'vegano', vegetarian: 'vegetariano', lactosa: 'sin_lactosa', lacteo: 'sin_lactosa', dairy: 'sin_lactosa' };

const meals = [
  {
    name: 'Desayuno',
    ratio: 0.25,
    slots: [
      { categories: ['dairy', 'protein'] },
      { categories: ['carb'] },
      { categories: ['fruit'] },
    ],
  },
  {
    name: 'Almuerzo',
    ratio: 0.4,
    slots: [
      { categories: ['protein', 'legume'] },
      { categories: ['carb'] },
      { categories: ['vegetable'] },
    ],
  },
  {
    name: 'Cena',
    ratio: 0.35,
    slots: [
      { categories: ['protein', 'legume'] },
      { categories: ['carb'] },
      { categories: ['vegetable'] },
    ],
  },
];

const MAX_SHARE_OF_MEAL_CALORIES = 0.65;

function matchesAny(name, keywords) {
  const lower = name.toLowerCase();
  return (keywords || []).some((k) => {
    const kw = k.trim().toLowerCase();
    return kw !== '' && lower.includes(kw);
  });
}

function hasAllTags(foodTags, required) {
  return required.every((req) => foodTags.includes(req));
}

function requiredTagsFor(restrictions) {
  const tags = [];
  for (const r of restrictions || []) {
    const lower = r.toLowerCase();
    for (const [keyword, tag] of Object.entries(restrictionTags)) {
      if (lower.includes(keyword)) tags.push(tag);
    }
  }
  return tags;
}

function filterFoods(foods, dislikes, restrictions) {
  const requiredTags = requiredTagsFor(restrictions);
  return foods.filter((f) => !matchesAny(f.name, dislikes) && hasAllTags(f.tags, requiredTags));
}

function groupByCategory(foods) {
  const byCategory = {};
  for (const f of foods) {
    (byCategory[f.category] ??= []).push(f);
  }
  return byCategory;
}

function unusedIn(foods, used) {
  return (foods || []).filter((f) => !used.has(f.id));
}

// Tries each category in order and returns the first liked food not already
// used; categories are a priority order, not an open pool — "legume" only
// gets considered once every "protein" item has been used elsewhere in the
// day. Without that, a liked legume (lower protein density, scales to a
// huge portion to hit the same protein target) could outrank an unliked but
// still-available meat, and the day's totals would blow past the target.
function pickFood(byCategory, categories, likes, used) {
  for (const cat of categories) {
    const unused = unusedIn(byCategory[cat], used);
    if (unused.length === 0) continue; // this category is exhausted — fall through

    const liked = unused.find((f) => matchesAny(f.name, likes));
    if (liked) return liked;
    return unused[0]; // no liked match here, but this category still wins over the next
  }

  // Every category in the list is fully used already — repeat rather than
  // leave the slot empty.
  for (const cat of categories) {
    if ((byCategory[cat] || []).length > 0) return byCategory[cat][0];
  }
  return null;
}

// Scales to roughly hit a per-item macro target, rounded to the nearest
// half-serving, clamped to a human-reasonable range. The calorie-share cap
// stops a low-density food (legumes are the clearest case) scaled up to hit
// its one target macro from blowing through the whole meal's calories.
function safeServings(targetMacro, macroPerServing, mealCalories, caloriesPerServing) {
  if (macroPerServing <= 0) return 1;
  let raw = targetMacro / macroPerServing;

  if (caloriesPerServing > 0) {
    const maxByCalories = (mealCalories * MAX_SHARE_OF_MEAL_CALORIES) / caloriesPerServing;
    raw = Math.min(raw, maxByCalories);
  }

  const rounded = Math.round(raw * 2) / 2;
  return Math.max(0.5, Math.min(3, rounded));
}

function toMealItem(food, servings) {
  return {
    food,
    servings,
    total_grams: Math.round(food.serving_grams * servings),
    calories: Math.round(food.calories * servings),
    protein_g: Math.round(food.protein_g * servings),
    carbs_g: Math.round(food.carbs_g * servings),
    fat_g: Math.round(food.fat_g * servings),
  };
}

// Picks real foods from the catalog to approximate the day's nutrition
// targets, respecting likes/dislikes/restrictions. A greedy heuristic, not
// an optimizer — totals are computed from whatever was actually picked, so
// the response is always internally consistent even off-target.
function buildPlan(foods, target, prefs) {
  const available = filterFoods(foods, prefs.dislikes, prefs.restrictions);
  const byCategory = groupByCategory(available);
  const usedProtein = new Set();

  const plan = { target, meals: [], calories: 0, protein_g: 0, carbs_g: 0, fat_g: 0 };

  for (const def of meals) {
    const mealCalories = target.calories * def.ratio;
    const mealProtein = target.protein_g * def.ratio;
    const mealFat = target.fat_g * def.ratio;

    const meal = { name: def.name, items: [], calories: 0, protein_g: 0, carbs_g: 0, fat_g: 0 };

    for (const slot of def.slots) {
      const food = pickFood(byCategory, slot.categories, prefs.likes, usedProtein);
      if (!food) continue;

      let servings;
      if (food.category === 'protein' || food.category === 'legume') {
        servings = safeServings(mealProtein, food.protein_g, mealCalories, food.calories);
        usedProtein.add(food.id);
      } else if (food.category === 'carb') {
        servings = safeServings(target.carbs_g * def.ratio, food.carbs_g, mealCalories, food.calories);
      } else {
        servings = 1; // dairy, vegetable, fruit, fat — a fixed, sensible single portion
      }

      const item = toMealItem(food, servings);
      meal.items.push(item);
      meal.calories += item.calories;
      meal.protein_g += item.protein_g;
      meal.carbs_g += item.carbs_g;
      meal.fat_g += item.fat_g;
    }

    // Top up fat only if the meal is meaningfully short of its fat target —
    // protein/carb sources already carry some fat, so this avoids doubling up.
    const fatGap = mealFat - meal.fat_g;
    if (fatGap > 5) {
      const fatFood = pickFood(byCategory, ['fat'], prefs.likes, new Set());
      if (fatFood) {
        const servings = safeServings(fatGap, fatFood.fat_g, mealCalories, fatFood.calories);
        const item = toMealItem(fatFood, servings);
        meal.items.push(item);
        meal.calories += item.calories;
        meal.protein_g += item.protein_g;
        meal.carbs_g += item.carbs_g;
        meal.fat_g += item.fat_g;
      }
    }

    plan.meals.push(meal);
    plan.calories += meal.calories;
    plan.protein_g += meal.protein_g;
    plan.carbs_g += meal.carbs_g;
    plan.fat_g += meal.fat_g;
  }

  return plan;
}

module.exports = { buildPlan };
