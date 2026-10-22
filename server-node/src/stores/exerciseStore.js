const { ErrNotFound } = require('../errors');

function rowToExercise(row) {
  return { id: row.id, name: row.name, muscle_group: row.muscle_group, equipment: JSON.parse(row.equipment), location: row.location };
}

class ExerciseStore {
  constructor(db) {
    this.db = db;
  }

  // A location of "gym" or "home" also includes "both"; empty returns the
  // whole catalog — the filter the PRD calls out as new versus Lyftr.
  listAll(location) {
    let query = 'SELECT id, name, muscle_group, equipment, location FROM exercises';
    const args = [];
    if (location) {
      query += ` WHERE location = ? OR location = 'both'`;
      args.push(location);
    }
    query += ' ORDER BY muscle_group, name';
    return this.db.prepare(query).all(...args).map(rowToExercise);
  }

  getById(id) {
    const row = this.db.prepare('SELECT id, name, muscle_group, equipment, location FROM exercises WHERE id = ?').get(id);
    if (!row) throw ErrNotFound;
    return rowToExercise(row);
  }
}

module.exports = { ExerciseStore };
