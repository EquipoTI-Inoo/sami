const { ErrNotFound } = require('../errors');

function rowToEntry(row) {
  const food = {
    id: row.f_id,
    name: row.name,
    category: row.category,
    serving_desc: row.serving_desc,
    serving_grams: row.serving_grams,
    calories: row.calories,
    protein_g: row.protein_g,
    carbs_g: row.carbs_g,
    fat_g: row.fat_g,
    tags: JSON.parse(row.tags),
  };

  return {
    id: row.id,
    user_id: row.user_id,
    food_id: row.food_id,
    food,
    meal_type: row.meal_type,
    servings: row.servings,
    total_grams: Math.round(food.serving_grams * row.servings),
    logged_on: row.logged_on,
    created_at: row.created_at,
    calories: Math.round(food.calories * row.servings),
    protein_g: Math.round(food.protein_g * row.servings),
    carbs_g: Math.round(food.carbs_g * row.servings),
    fat_g: Math.round(food.fat_g * row.servings),
  };
}

const SELECT_ENTRY = `
  SELECT fl.id, fl.user_id, fl.food_id, fl.meal_type, fl.servings, fl.logged_on, fl.created_at,
         f.id as f_id, f.name, f.category, f.serving_desc, f.serving_grams, f.calories, f.protein_g, f.carbs_g, f.fat_g, f.tags
  FROM food_log_entries fl
  JOIN peruvian_foods f ON f.id = fl.food_id
`;

class FoodLogStore {
  constructor(db) {
    this.db = db;
  }

  // loggedOn defaults to today (UTC) when empty — the common case is
  // logging what you're eating right now.
  create(userId, req) {
    const loggedOn = req.logged_on || new Date().toISOString().slice(0, 10);
    const res = this.db
      .prepare('INSERT INTO food_log_entries (user_id, food_id, meal_type, servings, logged_on) VALUES (?, ?, ?, ?, ?)')
      .run(userId, req.food_id, req.meal_type, req.servings, loggedOn);
    return this.getById(Number(res.lastInsertRowid));
  }

  getById(id) {
    const row = this.db.prepare(`${SELECT_ENTRY} WHERE fl.id = ?`).get(id);
    if (!row) throw ErrNotFound;
    return rowToEntry(row);
  }

  // Macros recomputed fresh from the current peruvian_foods row every time
  // — never stored, so a catalog correction shows up immediately.
  listForDay(userId, loggedOn) {
    const rows = this.db.prepare(`${SELECT_ENTRY} WHERE fl.user_id = ? AND fl.logged_on = ? ORDER BY fl.created_at`).all(userId, loggedOn);
    return rows.map(rowToEntry);
  }

  getOwner(id) {
    const row = this.db.prepare('SELECT user_id FROM food_log_entries WHERE id = ?').get(id);
    if (!row) throw ErrNotFound;
    return row.user_id;
  }

  delete(id) {
    this.db.prepare('DELETE FROM food_log_entries WHERE id = ?').run(id);
  }
}

module.exports = { FoodLogStore };
