const { ErrNotFound } = require('../errors');

function marshalStrings(v) {
  return JSON.stringify(v || []);
}

function unmarshalStrings(s) {
  if (!s) return [];
  return JSON.parse(s);
}

class ProfileStore {
  constructor(db) {
    this.db = db;
  }

  upsertProfile(userId, req) {
    const equipment = marshalStrings(req.equipment);
    const likes = marshalStrings(req.likes);
    const dislikes = marshalStrings(req.dislikes);
    const restrictions = marshalStrings(req.restrictions);

    this.db
      .prepare(
        `
      INSERT INTO user_profile
        (user_id, age, sex, height_cm, body_fat_pct, activity_level, goal, training_location, equipment, likes, dislikes, restrictions, updated_at)
      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
      ON CONFLICT(user_id) DO UPDATE SET
        age = excluded.age,
        sex = excluded.sex,
        height_cm = excluded.height_cm,
        body_fat_pct = excluded.body_fat_pct,
        activity_level = excluded.activity_level,
        goal = excluded.goal,
        training_location = excluded.training_location,
        equipment = excluded.equipment,
        likes = excluded.likes,
        dislikes = excluded.dislikes,
        restrictions = excluded.restrictions,
        updated_at = excluded.updated_at
    `
      )
      .run(
        userId,
        req.age,
        req.sex,
        req.height_cm,
        req.body_fat_pct ?? null,
        req.activity_level,
        req.goal,
        req.training_location,
        equipment,
        likes,
        dislikes,
        restrictions
      );
  }

  getProfile(userId) {
    const row = this.db
      .prepare(
        `SELECT user_id, age, sex, height_cm, body_fat_pct, activity_level, goal, training_location, equipment, likes, dislikes, restrictions, created_at, updated_at
         FROM user_profile WHERE user_id = ?`
      )
      .get(userId);
    if (!row) throw ErrNotFound;

    return {
      user_id: row.user_id,
      age: row.age,
      sex: row.sex,
      height_cm: row.height_cm,
      body_fat_pct: row.body_fat_pct,
      activity_level: row.activity_level,
      goal: row.goal,
      training_location: row.training_location,
      equipment: unmarshalStrings(row.equipment),
      likes: unmarshalStrings(row.likes),
      dislikes: unmarshalStrings(row.dislikes),
      restrictions: unmarshalStrings(row.restrictions),
      created_at: row.created_at,
      updated_at: row.updated_at,
    };
  }

  insertBodyMeasurement(userId, input) {
    const today = new Date().toISOString().slice(0, 10);
    const res = this.db
      .prepare(
        `INSERT INTO body_measurements (user_id, logged_on, weight_kg, waist_cm, hip_cm, arm_cm, body_fat_pct, note)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
      )
      .run(
        userId,
        today,
        input.weightKg,
        input.waistCm ?? null,
        input.hipCm ?? null,
        input.armCm ?? null,
        input.bodyFatPct ?? null,
        input.note ?? null
      );
    return this.getBodyMeasurementById(Number(res.lastInsertRowid));
  }

  getBodyMeasurementById(id) {
    const row = this.db
      .prepare(
        `SELECT id, user_id, logged_on, weight_kg, waist_cm, hip_cm, arm_cm, body_fat_pct, note, created_at
         FROM body_measurements WHERE id = ?`
      )
      .get(id);
    if (!row) throw ErrNotFound;
    return row;
  }

  getLatestBodyMeasurement(userId) {
    const row = this.db
      .prepare(
        `SELECT id, user_id, logged_on, weight_kg, waist_cm, hip_cm, arm_cm, body_fat_pct, note, created_at
         FROM body_measurements WHERE user_id = ? ORDER BY logged_on DESC, id DESC LIMIT 1`
      )
      .get(userId);
    if (!row) throw ErrNotFound;
    return row;
  }

  listRecentBodyMeasurements(userId, limit) {
    return this.db
      .prepare(
        `SELECT id, user_id, logged_on, weight_kg, waist_cm, hip_cm, arm_cm, body_fat_pct, note, created_at
         FROM body_measurements WHERE user_id = ? ORDER BY logged_on DESC, id DESC LIMIT ?`
      )
      .all(userId, limit);
  }

  insertNutritionTargets(userId, t) {
    const res = this.db
      .prepare(
        `INSERT INTO nutrition_targets (user_id, calories, protein_g, carbs_g, fat_g, updated_by, reason)
         VALUES (?, ?, ?, ?, ?, ?, ?)`
      )
      .run(userId, t.calories, t.proteinG, t.carbsG, t.fatG, t.updatedBy, t.reason);
    return this.getNutritionTargetsById(Number(res.lastInsertRowid));
  }

  getNutritionTargetsById(id) {
    const row = this.db
      .prepare(`SELECT id, user_id, calories, protein_g, carbs_g, fat_g, updated_by, reason, created_at FROM nutrition_targets WHERE id = ?`)
      .get(id);
    if (!row) throw ErrNotFound;
    return row;
  }

  getLatestNutritionTargets(userId) {
    const row = this.db
      .prepare(
        `SELECT id, user_id, calories, protein_g, carbs_g, fat_g, updated_by, reason, created_at
         FROM nutrition_targets WHERE user_id = ? ORDER BY created_at DESC LIMIT 1`
      )
      .get(userId);
    if (!row) throw ErrNotFound;
    return row;
  }

  getNutritionTargetsHistory(userId) {
    return this.db
      .prepare(
        `SELECT id, user_id, calories, protein_g, carbs_g, fat_g, updated_by, reason, created_at
         FROM nutrition_targets WHERE user_id = ? ORDER BY created_at DESC`
      )
      .all(userId);
  }
}

module.exports = { ProfileStore };
