package training

import (
	"testing"

	"coachia-backend/models"
)

func sampleCatalog() []models.Exercise {
	make := func(id int64, group string) models.Exercise {
		return models.Exercise{ID: id, Name: group, MuscleGroup: group, Location: "both"}
	}
	var catalog []models.Exercise
	id := int64(1)
	for _, group := range []string{"piernas", "pecho", "espalda", "hombros", "brazos", "core"} {
		// Three options per group so round-robin has real variety to pick from.
		for i := 0; i < 3; i++ {
			catalog = append(catalog, make(id, group))
			id++
		}
	}
	return catalog
}

func TestGenerateProducesThreeDays(t *testing.T) {
	program := Generate(sampleCatalog(), "maintain")
	if len(program.Days) != 3 {
		t.Fatalf("expected 3 days, got %d", len(program.Days))
	}
	for _, day := range program.Days {
		if len(day.Exercises) == 0 {
			t.Errorf("day %q has no exercises", day.Name)
		}
	}
}

func TestGenerateRepRangesMatchGoal(t *testing.T) {
	cases := []struct {
		goal                       string
		wantSets, wantMin, wantMax int
	}{
		{"cut", 3, 12, 15},
		{"maintain", 3, 10, 12},
		{"bulk", 4, 6, 10},
		{"performance", 4, 5, 8},
		{"unknown-goal", 3, 10, 12}, // falls back to maintain's scheme
	}

	for _, tc := range cases {
		program := Generate(sampleCatalog(), tc.goal)
		for _, day := range program.Days {
			for _, ex := range day.Exercises {
				if ex.TargetSets != tc.wantSets || ex.TargetRepsMin != tc.wantMin || ex.TargetRepsMax != tc.wantMax {
					t.Fatalf("goal %q: expected %d sets %d-%d reps, got %d sets %d-%d reps",
						tc.goal, tc.wantSets, tc.wantMin, tc.wantMax, ex.TargetSets, ex.TargetRepsMin, ex.TargetRepsMax)
				}
			}
		}
	}
}

// Regression-guard: never crash or produce garbage when the catalog is
// missing a muscle group entirely (e.g. a very narrow equipment filter) —
// that group's slots should just be skipped, not fill with a zero-value
// exercise.
func TestGenerateSkipsMissingMuscleGroups(t *testing.T) {
	catalog := []models.Exercise{{ID: 1, Name: "Sentadilla", MuscleGroup: "piernas", Location: "both"}}

	program := Generate(catalog, "cut")

	for _, day := range program.Days {
		for _, ex := range day.Exercises {
			if ex.ExerciseID != 1 {
				t.Fatalf("expected only exercise id 1 to ever be picked, got %d", ex.ExerciseID)
			}
		}
	}
}

func TestGenerateRotatesThroughAvailableOptions(t *testing.T) {
	program := Generate(sampleCatalog(), "maintain")

	// Day 1 asks for "piernas" twice — with 3 options available, the two
	// picks within the same day must not be the exact same catalog row.
	day1Legs := []int64{}
	for _, ex := range program.Days[0].Exercises[:2] {
		day1Legs = append(day1Legs, ex.ExerciseID)
	}
	if day1Legs[0] == day1Legs[1] {
		t.Fatalf("expected the two leg picks on day 1 to differ, got the same exercise twice: %v", day1Legs)
	}
}
