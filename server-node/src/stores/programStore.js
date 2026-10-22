const { ErrNotFound } = require('../errors');

class ProgramStore {
  constructor(db) {
    this.db = db;
  }

  // Persists a generated program — program + days + exercises — in one
  // transaction, so a partial write can never leave a program with some
  // days missing their exercises.
  create(userId, goal, location, days) {
    this.db.exec('BEGIN');
    try {
      const res = this.db.prepare('INSERT INTO programs (user_id, goal, location) VALUES (?, ?, ?)').run(userId, goal, location);
      const programId = Number(res.lastInsertRowid);

      days.forEach((day, dayIndex) => {
        const dayRes = this.db.prepare('INSERT INTO program_days (program_id, day_index, name) VALUES (?, ?, ?)').run(programId, dayIndex, day.name);
        const dayId = Number(dayRes.lastInsertRowid);

        day.exercises.forEach((pick, orderIndex) => {
          this.db
            .prepare(
              `INSERT INTO program_exercises (program_day_id, exercise_id, order_index, target_sets, target_reps_min, target_reps_max)
               VALUES (?, ?, ?, ?, ?, ?)`
            )
            .run(dayId, pick.exerciseId, orderIndex, pick.targetSets, pick.targetRepsMin, pick.targetRepsMax);
        });
      });

      this.db.exec('COMMIT');
      return this.getById(programId);
    } catch (err) {
      this.db.exec('ROLLBACK');
      throw err;
    }
  }

  getLatest(userId) {
    const row = this.db.prepare('SELECT id FROM programs WHERE user_id = ? ORDER BY created_at DESC LIMIT 1').get(userId);
    if (!row) throw ErrNotFound;
    return this.getById(row.id);
  }

  getById(programId) {
    const p = this.db.prepare('SELECT id, user_id, goal, location, created_at FROM programs WHERE id = ?').get(programId);
    if (!p) throw ErrNotFound;

    const days = this.db
      .prepare('SELECT id, day_index, name FROM program_days WHERE program_id = ? ORDER BY day_index')
      .all(programId)
      .map((d) => ({ id: d.id, day_index: d.day_index, name: d.name, exercises: [] }));

    const exerciseRows = this.db
      .prepare(
        `SELECT pe.id as pe_id, pe.program_day_id, pe.exercise_id, pe.order_index, pe.target_sets, pe.target_reps_min, pe.target_reps_max,
                e.id as e_id, e.name, e.muscle_group, e.equipment, e.location
         FROM program_exercises pe
         JOIN program_days pd ON pd.id = pe.program_day_id
         JOIN exercises e ON e.id = pe.exercise_id
         WHERE pd.program_id = ?
         ORDER BY pe.program_day_id, pe.order_index`
      )
      .all(programId);

    const exercisesByDay = {};
    for (const row of exerciseRows) {
      (exercisesByDay[row.program_day_id] ??= []).push({
        id: row.pe_id,
        exercise_id: row.exercise_id,
        exercise: { id: row.e_id, name: row.name, muscle_group: row.muscle_group, equipment: JSON.parse(row.equipment), location: row.location },
        order_index: row.order_index,
        target_sets: row.target_sets,
        target_reps_min: row.target_reps_min,
        target_reps_max: row.target_reps_max,
      });
    }
    for (const day of days) {
      if (exercisesByDay[day.id]) day.exercises = exercisesByDay[day.id];
    }

    return { ...p, days };
  }
}

module.exports = { ProgramStore };
