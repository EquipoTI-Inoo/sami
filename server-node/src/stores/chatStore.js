class ChatStore {
  constructor(db) {
    this.db = db;
  }

  append(userId, role, content) {
    this.db.prepare('INSERT INTO chat_messages (user_id, role, content) VALUES (?, ?, ?)').run(userId, role, JSON.stringify(content));
  }

  list(userId) {
    const rows = this.db.prepare('SELECT role, content FROM chat_messages WHERE user_id = ? ORDER BY id ASC').all(userId);
    return rows.map((r) => ({ role: r.role, content: JSON.parse(r.content) }));
  }
}

module.exports = { ChatStore };
