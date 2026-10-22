const { ErrNotFound } = require('../errors');

class WorkoutStore {
  constructor(db) {
    this.db = db;
  }

  countCompletedSince(userId, since) {
    const row = this.db.prepare('SELECT COUNT(1) as c FROM workouts WHERE user_id = ? AND completed_at IS NOT NULL AND started_at >= ?').get(userId, since);
    return row.c;
  }

  countCompleted(userId) {
    const row = this.db.prepare('SELECT COUNT(1) as c FROM workouts WHERE user_id = ? AND completed_at IS NOT NULL').get(userId);
    return row.c;
  }

  create(userId) {
    const res = this.db.prepare('INSERT INTO workouts (user_id) VALUES (?)').run(userId);
    return this.getById(Number(res.lastInsertRowid));
  }

  // Lightweight ownership check used before any write to a workout, so one
  // user's request can never touch another's session.
  getWorkoutOwner(workoutId) {
    const row = this.db.prepare('SELECT user_id FROM workouts WHERE id = ?').get(workoutId);
    if (!row) throw ErrNotFound;
    return row.user_id;
  }

  // Resolves straight through to the owning user, for authorizing a set log
  // without a round trip through the workout first.
  getWorkoutExerciseOwner(workoutExerciseId) {
    const row = this.db
      .prepare('SELECT w.user_id as user_id, w.id as workout_id FROM workout_exercises we JOIN workouts w ON w.id = we.workout_id WHERE we.id = ?')
      .get(workoutExerciseId);
    if (!row) throw ErrNotFound;
    return { userId: row.user_id, workoutId: row.workout_id };
  }

  addExercise(workoutId, exerciseId) {
    const { c: nextOrder } = this.db.prepare('SELECT COUNT(1) as c FROM workout_exercises WHERE workout_id = ?').get(workoutId);
    const res = this.db
      .prepare('INSERT INTO workout_exercises (workout_id, exercise_id, order_index) VALUES (?, ?, ?)')
      .run(workoutId, exerciseId, nextOrder);
    return this._getWorkoutExerciseById(Number(res.lastInsertRowid));
  }

  _getWorkoutExerciseById(id) {
    const row = this.db.prepare('SELECT id, workout_id, exercise_id, order_index FROM workout_exercises WHERE id = ?').get(id);
    if (!row) throw ErrNotFound;
    return row;
  }

  logSet(workoutExerciseId, reps, weightKg) {
    const { c } = this.db.prepare('SELECT COUNT(1) as c FROM workout_sets WHERE workout_exercise_id = ?').get(workoutExerciseId);
    const nextSetNumber = c + 1;
    const res = this.db
      .prepare('INSERT INTO workout_sets (workout_exercise_id, set_number, reps, weight_kg) VALUES (?, ?, ?, ?)')
      .run(workoutExerciseId, nextSetNumber, reps, weightKg ?? null);
    return this.db
      .prepare('SELECT id, workout_exercise_id, set_number, reps, weight_kg, completed_at FROM workout_sets WHERE id = ?')
      .get(Number(res.lastInsertRowid));
  }

  complete(workoutId) {
    this.db.prepare(`UPDATE workouts SET completed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`).run(workoutId);
    return this.getById(workoutId);
  }

  // Loads a workout with its exercises (including catalog details) and sets
  // fully nested — the shape Gym Mode and the workout detail view both need.
  getById(workoutId) {
    const w = this.db.prepare('SELECT id, user_id, started_at, completed_at, notes FROM workouts WHERE id = ?').get(workoutId);
    if (!w) throw ErrNotFound;

    const exerciseRows = this.db
      .prepare(
        `SELECT we.id as we_id, we.workout_id, we.exercise_id, we.order_index,
                e.id as e_id, e.name, e.muscle_group, e.equipment, e.location
         FROM workout_exercises we
         JOIN exercises e ON e.id = we.exercise_id
         WHERE we.workout_id = ?
         ORDER BY we.order_index`
      )
      .all(workoutId);

    const exercises = exerciseRows.map((row) => ({
      id: row.we_id,
      workout_id: row.workout_id,
      exercise_id: row.exercise_id,
      exercise: { id: row.e_id, name: row.name, muscle_group: row.muscle_group, equipment: JSON.parse(row.equipment), location: row.location },
      order_index: row.order_index,
      sets: [],
    }));

    const setRows = this.db
      .prepare(
        `SELECT ws.id, ws.workout_exercise_id, ws.set_number, ws.reps, ws.weight_kg, ws.completed_at
         FROM workout_sets ws
         JOIN workout_exercises we ON we.id = ws.workout_exercise_id
         WHERE we.workout_id = ?
         ORDER BY ws.workout_exercise_id, ws.set_number`
      )
      .all(workoutId);

    const setsByExercise = {};
    for (const set of setRows) {
      (setsByExercise[set.workout_exercise_id] ??= []).push(set);
    }
    for (const ex of exercises) {
      ex.sets = setsByExercise[ex.id] ?? [];
    }

    return { ...w, exercise_count: exercises.length, exercises };
  }

  // Summaries (no nested exercises/sets, just a count) newest first.
  listForUser(userId) {
    const rows = this.db
      .prepare(
        `SELECT w.id, w.user_id, w.started_at, w.completed_at, w.notes,
                (SELECT COUNT(1) FROM workout_exercises we WHERE we.workout_id = w.id) as exercise_count
         FROM workouts w WHERE w.user_id = ? ORDER BY w.started_at DESC`
      )
      .all(userId);
    return rows;
  }
}

module.exports = { WorkoutStore };
