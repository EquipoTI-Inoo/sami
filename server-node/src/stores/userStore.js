const { ErrNotFound, ErrEmailTaken } = require('../errors');

class UserStore {
  constructor(db) {
    this.db = db;
  }

  create(email, passwordHash) {
    try {
      const res = this.db.prepare('INSERT INTO users (email, password_hash) VALUES (?, ?)').run(email, passwordHash);
      return this.findById(Number(res.lastInsertRowid));
    } catch (err) {
      if (String(err.message).toLowerCase().includes('unique constraint')) {
        throw ErrEmailTaken;
      }
      throw err;
    }
  }

  findById(id) {
    const row = this.db.prepare('SELECT id, email, created_at FROM users WHERE id = ?').get(id);
    if (!row) throw ErrNotFound;
    return row;
  }

  findByEmail(email) {
    const row = this.db.prepare('SELECT id, email, created_at FROM users WHERE email = ?').get(email);
    if (!row) throw ErrNotFound;
    return row;
  }

  findByEmailWithPassword(email) {
    const row = this.db.prepare('SELECT id, email, created_at, password_hash FROM users WHERE email = ?').get(email);
    if (!row) throw ErrNotFound;
    return row;
  }
}

module.exports = { UserStore };
