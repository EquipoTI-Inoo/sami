// Ported 1:1 from backend/training/program.go — same templates, same scheme.

const dayTemplates = [
  { name: 'Día 1: Piernas y empuje', muscleGroups: ['piernas', 'piernas', 'pecho', 'pecho', 'core'] },
  { name: 'Día 2: Tirón y hombros', muscleGroups: ['espalda', 'espalda', 'hombros', 'hombros', 'core'] },
  { name: 'Día 3: Full body y brazos', muscleGroups: ['piernas', 'espalda', 'pecho', 'brazos', 'brazos', 'core'] },
];

const repRangesByGoal = {
  cut: { sets: 3, repsMin: 12, repsMax: 15 },
  maintain: { sets: 3, repsMin: 10, repsMax: 12 },
  bulk: { sets: 4, repsMin: 6, repsMax: 10 },
  performance: { sets: 4, repsMin: 5, repsMax: 8 },
};

// Builds a deterministic 3-day program from the given exercise catalog
// (already filtered by the caller to the person's location), picking
// exercises by muscle group round-robin so repeated slots across days pull
// different catalog rows when enough variety exists. No absolute weight is
// ever prescribed — that depends on the person's own strength, which this
// app has no way to know until they log real sets.
function generate(catalog, goal) {
  const scheme = repRangesByGoal[goal] ?? repRangesByGoal.maintain;

  const byGroup = {};
  for (const e of catalog) {
    (byGroup[e.muscle_group] ??= []).push(e);
  }
  const cursor = {};

  const days = [];
  for (const tmpl of dayTemplates) {
    const picks = [];
    for (const group of tmpl.muscleGroups) {
      const options = byGroup[group] ?? [];
      if (options.length === 0) continue;
      const idx = (cursor[group] ?? 0) % options.length;
      cursor[group] = (cursor[group] ?? 0) + 1;
      const ex = options[idx];
      picks.push({ exerciseId: ex.id, targetSets: scheme.sets, targetRepsMin: scheme.repsMin, targetRepsMax: scheme.repsMax });
    }
    if (picks.length === 0) continue;
    days.push({ name: tmpl.name, exercises: picks });
  }

  return { days };
}

module.exports = { generate };
