const { ErrNotFound } = require('../errors');

function rowToChange(row) {
  return { ...row, payload: JSON.parse(row.payload) };
}

class PendingChangeStore {
  constructor(db) {
    this.db = db;
  }

  create(userId, changeType, payload, reason) {
    const res = this.db
      .prepare('INSERT INTO pending_changes (user_id, change_type, payload, reason) VALUES (?, ?, ?, ?)')
      .run(userId, changeType, JSON.stringify(payload), reason);
    return this.getById(Number(res.lastInsertRowid));
  }

  getById(id) {
    const row = this.db
      .prepare('SELECT id, user_id, change_type, payload, reason, status, created_at, resolved_at FROM pending_changes WHERE id = ?')
      .get(id);
    if (!row) throw ErrNotFound;
    return rowToChange(row);
  }

  listPending(userId) {
    const rows = this.db
      .prepare(
        `SELECT id, user_id, change_type, payload, reason, status, created_at, resolved_at
         FROM pending_changes WHERE user_id = ? AND status = 'pending' ORDER BY created_at DESC`
      )
      .all(userId);
    return rows.map(rowToChange);
  }

  resolve(id, status) {
    this.db.prepare(`UPDATE pending_changes SET status = ?, resolved_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`).run(status, id);
    return this.getById(id);
  }
}

module.exports = { PendingChangeStore };
