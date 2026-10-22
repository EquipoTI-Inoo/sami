// Ported 1:1 from backend/nutrition/calculator.go — same constants, same math.

const activityMultipliers = { sedentary: 1.2, light: 1.375, moderate: 1.55, active: 1.725, very_active: 1.9 };

// Express the target as a fraction of TDEE: a 20% deficit to cut, a 10%
// surplus to bulk. Performance gets a small surplus to fuel training
// without the fat-gain pace of a full bulk.
const goalAdjustments = { cut: 0.8, maintain: 1.0, bulk: 1.1, performance: 1.05 };

const PROTEIN_G_PER_KG = 2.0; // evidence-based target for training adults, regardless of goal
const FAT_CALORIE_FRACTION = 0.25;

// No deficit should ever be recommended below these without medical
// supervision — commonly cited clinical floors (e.g. NHS, dietitian
// guidance). A tiny/old/low-activity profile combined with "cut" can
// otherwise compute well under 1000 kcal/day from the raw formula.
const MIN_SAFE_CALORIES_FEMALE = 1200.0;
const MIN_SAFE_CALORIES_MALE = 1500.0;

function calculate(sex, age, heightCm, weightKg, activityLevel, goal) {
  let bmr = 10 * weightKg + 6.25 * heightCm - 5 * age;
  let minSafeCalories = MIN_SAFE_CALORIES_FEMALE;
  if (sex === 'male') {
    bmr += 5;
    minSafeCalories = MIN_SAFE_CALORIES_MALE;
  } else {
    bmr -= 161;
  }

  const tdee = bmr * (activityMultipliers[activityLevel] ?? 1);
  const calories = Math.max(tdee * (goalAdjustments[goal] ?? 1), minSafeCalories);

  let proteinKcal = PROTEIN_G_PER_KG * weightKg * 4;
  let fatKcal = calories * FAT_CALORIE_FRACTION;

  // A high body weight relative to a tight calorie target (e.g. an
  // aggressive cut) can push protein+fat alone past the whole budget —
  // scale both down proportionally rather than let carbs go negative or
  // report macros that don't actually sum to the stated calories.
  if (proteinKcal + fatKcal > calories) {
    const scale = calories / (proteinKcal + fatKcal);
    proteinKcal *= scale;
    fatKcal *= scale;
  }

  const carbKcal = Math.max(0, calories - proteinKcal - fatKcal);

  return {
    calories: Math.round(calories),
    proteinG: Math.round(proteinKcal / 4),
    carbsG: Math.round(carbKcal / 4),
    fatG: Math.round(fatKcal / 9),
  };
}

module.exports = { calculate };
