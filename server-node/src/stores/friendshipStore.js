const { ErrNotFound } = require('../errors');

function rowToFriendship(row) {
  return {
    id: row.id,
    requester_id: row.requester_id,
    recipient_id: row.recipient_id,
    status: row.status,
    created_at: row.created_at,
    resolved_at: row.resolved_at,
    friend_id: row.friend_id ?? 0,
    friend_email: row.friend_email ?? '',
  };
}

class FriendshipStore {
  constructor(db) {
    this.db = db;
  }

  create(requesterId, recipientId) {
    const res = this.db.prepare('INSERT INTO friendships (requester_id, recipient_id) VALUES (?, ?)').run(requesterId, recipientId);
    return this.getById(Number(res.lastInsertRowid));
  }

  getById(id) {
    const row = this.db.prepare('SELECT id, requester_id, recipient_id, status, created_at, resolved_at FROM friendships WHERE id = ?').get(id);
    if (!row) throw ErrNotFound;
    return rowToFriendship(row);
  }

  // Any existing friendship (any status) between the two users, regardless
  // of who requested — used to block duplicate/overlapping requests.
  findBetween(userA, userB) {
    const row = this.db
      .prepare(
        `SELECT id, requester_id, recipient_id, status, created_at, resolved_at
         FROM friendships
         WHERE (requester_id = ? AND recipient_id = ?) OR (requester_id = ? AND recipient_id = ?)
         ORDER BY created_at DESC LIMIT 1`
      )
      .get(userA, userB, userB, userA);
    if (!row) throw ErrNotFound;
    return rowToFriendship(row);
  }

  // Pending requests sent TO userId, enriched with the requester's email.
  listIncomingPending(userId) {
    const rows = this.db
      .prepare(
        `SELECT f.id, f.requester_id, f.recipient_id, f.status, f.created_at, f.resolved_at, u.id as friend_id, u.email as friend_email
         FROM friendships f
         JOIN users u ON u.id = f.requester_id
         WHERE f.recipient_id = ? AND f.status = 'pending'
         ORDER BY f.created_at DESC`
      )
      .all(userId);
    return rows.map(rowToFriendship);
  }

  // Accepted friendships for userId, enriched with the OTHER participant's
  // id/email regardless of which side sent the request.
  listAccepted(userId) {
    const rows = this.db
      .prepare(
        `SELECT f.id, f.requester_id, f.recipient_id, f.status, f.created_at, f.resolved_at,
                u.id as friend_id, u.email as friend_email
         FROM friendships f
         JOIN users u ON u.id = CASE WHEN f.requester_id = ? THEN f.recipient_id ELSE f.requester_id END
         WHERE f.status = 'accepted' AND (f.requester_id = ? OR f.recipient_id = ?)
         ORDER BY f.resolved_at DESC`
      )
      .all(userId, userId, userId);
    return rows.map(rowToFriendship);
  }

  resolve(id, status) {
    this.db.prepare(`UPDATE friendships SET status = ?, resolved_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`).run(status, id);
    return this.getById(id);
  }
}

module.exports = { FriendshipStore };
