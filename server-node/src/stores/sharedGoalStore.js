const { ErrNotFound } = require('../errors');

class SharedGoalStore {
  constructor(db) {
    this.db = db;
  }

  create(friendshipId, createdBy, req) {
    const res = this.db
      .prepare('INSERT INTO shared_goals (friendship_id, created_by, goal_type, description, target_value, target_date) VALUES (?, ?, ?, ?, ?, ?)')
      .run(friendshipId, createdBy, req.goal_type, req.description, req.target_value ?? null, req.target_date ?? null);
    return this.getById(Number(res.lastInsertRowid));
  }

  getById(id) {
    const row = this.db
      .prepare('SELECT id, friendship_id, created_by, goal_type, description, target_value, target_date, status, created_at, completed_at FROM shared_goals WHERE id = ?')
      .get(id);
    if (!row) throw ErrNotFound;
    return row;
  }

  listForFriendship(friendshipId) {
    return this.db
      .prepare(
        `SELECT id, friendship_id, created_by, goal_type, description, target_value, target_date, status, created_at, completed_at
         FROM shared_goals WHERE friendship_id = ? ORDER BY created_at DESC`
      )
      .all(friendshipId);
  }

  complete(id) {
    this.db.prepare(`UPDATE shared_goals SET status = 'completed', completed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`).run(id);
    return this.getById(id);
  }
}

module.exports = { SharedGoalStore };
