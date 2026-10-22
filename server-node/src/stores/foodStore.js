const { ErrNotFound } = require('../errors');

function rowToFood(row) {
  return {
    id: row.id,
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
}

class FoodStore {
  constructor(db) {
    this.db = db;
  }

  listAll() {
    const rows = this.db.prepare('SELECT id, name, category, serving_desc, serving_grams, calories, protein_g, carbs_g, fat_g, tags FROM peruvian_foods').all();
    return rows.map(rowToFood);
  }

  getById(id) {
    const row = this.db
      .prepare('SELECT id, name, category, serving_desc, serving_grams, calories, protein_g, carbs_g, fat_g, tags FROM peruvian_foods WHERE id = ?')
      .get(id);
    if (!row) throw ErrNotFound;
    return rowToFood(row);
  }
}

module.exports = { FoodStore };
