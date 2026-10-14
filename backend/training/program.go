package training

import "coachia-backend/models"

type ExercisePick struct {
	ExerciseID    int64
	TargetSets    int
	TargetRepsMin int
	TargetRepsMax int
}

type DayPlan struct {
	Name      string
	Exercises []ExercisePick
}

type GeneratedProgram struct {
	Days []DayPlan
}

// dayTemplates define a classic 3-day full-body split (legs+push one day,
// pull+shoulders the next, full-body+arms the third) — balanced push/pull/
// legs coverage across the week without needing a training-frequency
// question we don't currently ask at onboarding.
var dayTemplates = []struct {
	Name         string
	MuscleGroups []string
}{
	{"Día 1: Piernas y empuje", []string{"piernas", "piernas", "pecho", "pecho", "core"}},
	{"Día 2: Tirón y hombros", []string{"espalda", "espalda", "hombros", "hombros", "core"}},
	{"Día 3: Full body y brazos", []string{"piernas", "espalda", "pecho", "brazos", "brazos", "core"}},
}

type repScheme struct {
	Sets, RepsMin, RepsMax int
}

// repRangesByGoal follows standard strength-training rep range conventions:
// higher reps/moderate load for a cut (metabolic stress, joint-friendly
// under fatigue), lower reps/heavier load for bulk (hypertrophy-biased),
// lowest reps for performance (strength/power bias). No absolute weight is
// ever prescribed — that depends on the person's own strength, which this
// app has no way to know until they log real sets.
var repRangesByGoal = map[string]repScheme{
	"cut":         {Sets: 3, RepsMin: 12, RepsMax: 15},
	"maintain":    {Sets: 3, RepsMin: 10, RepsMax: 12},
	"bulk":        {Sets: 4, RepsMin: 6, RepsMax: 10},
	"performance": {Sets: 4, RepsMin: 5, RepsMax: 8},
}

// Generate builds a deterministic 3-day program from the given exercise
// catalog (already filtered by the caller to the person's location), picking
// exercises by muscle group round-robin so repeated slots across days pull
// different catalog rows when enough variety exists. Goals outside
// repRangesByGoal fall back to the "maintain" scheme.
func Generate(catalog []models.Exercise, goal string) GeneratedProgram {
	scheme, ok := repRangesByGoal[goal]
	if !ok {
		scheme = repRangesByGoal["maintain"]
	}

	byGroup := map[string][]models.Exercise{}
	for _, e := range catalog {
		byGroup[e.MuscleGroup] = append(byGroup[e.MuscleGroup], e)
	}
	cursor := map[string]int{}

	var days []DayPlan
	for _, tmpl := range dayTemplates {
		var picks []ExercisePick
		for _, group := range tmpl.MuscleGroups {
			options := byGroup[group]
			if len(options) == 0 {
				continue
			}
			idx := cursor[group] % len(options)
			cursor[group]++
			ex := options[idx]
			picks = append(picks, ExercisePick{
				ExerciseID:    ex.ID,
				TargetSets:    scheme.Sets,
				TargetRepsMin: scheme.RepsMin,
				TargetRepsMax: scheme.RepsMax,
			})
		}
		if len(picks) == 0 {
			continue
		}
		days = append(days, DayPlan{Name: tmpl.Name, Exercises: picks})
	}

	return GeneratedProgram{Days: days}
}
