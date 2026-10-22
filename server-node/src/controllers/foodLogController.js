const { ErrNotFound } = require('../errors');
const { ok, created, badRequest, notFound, internalErrorf } = require('../utils/response');

const MEAL_TYPES = ['desayuno', 'almuerzo', 'cena', 'snack'];

function createFoodLogController(foodLog, foods, profiles) {
  return {
    create(req, res) {
      const body = req.body || {};
      if (!body.food_id || !MEAL_TYPES.includes(body.meal_type) || typeof body.servings !== 'number' || body.servings <= 0 || body.servings > 20) {
        badRequest(res, 'food_id, meal_type and servings are required');
        return;
      }

      try {
        foods.getById(body.food_id);
      } catch (err) {
        if (err === ErrNotFound) {
          badRequest(res, "that food doesn't exist");
          return;
        }
        internalErrorf(req, res, 'could not validate the food', err);
        return;
      }

      try {
        const entry = foodLog.create(req.userId, body);
        created(res, entry);
      } catch (err) {
        internalErrorf(req, res, 'could not log that food', err);
      }
    },

    // getDay returns everything logged on a date (default today) plus
    // totals consumed so far and the current target — everything the Home
    // ring needs in one call. Totals are summed from the entries in the
    // same response, never stored separately, so they can never drift from
    // what's listed.
    getDay(req, res) {
      const date = req.query.date || new Date().toISOString().slice(0, 10);

      let entries;
      try {
        entries = foodLog.listForDay(req.userId, date);
      } catch (err) {
        internalErrorf(req, res, 'could not load your food log', err);
        return;
      }

      const consumed = { calories: 0, protein_g: 0, carbs_g: 0, fat_g: 0 };
      for (const e of entries) {
        consumed.calories += e.calories;
        consumed.protein_g += e.protein_g;
        consumed.carbs_g += e.carbs_g;
        consumed.fat_g += e.fat_g;
      }

      let target = null;
      try {
        target = profiles.getLatestNutritionTargets(req.userId);
      } catch (err) {
        if (err !== ErrNotFound) {
          internalErrorf(req, res, 'could not load your targets', err);
          return;
        }
      }

      ok(res, { logged_on: date, entries, consumed, target });
    },

    delete(req, res) {
      const id = Number(req.params.id);
      if (!Number.isInteger(id)) {
        badRequest(res, 'invalid entry id');
        return;
      }

      let owner;
      try {
        owner = foodLog.getOwner(id);
      } catch (err) {
        if (err === ErrNotFound) {
          notFound(res, 'entry not found');
          return;
        }
        internalErrorf(req, res, 'could not load the entry', err);
        return;
      }
      if (owner !== req.userId) {
        notFound(res, 'entry not found');
        return;
      }

      try {
        foodLog.delete(id);
        ok(res, { deleted: true });
      } catch (err) {
        internalErrorf(req, res, 'could not remove the entry', err);
      }
    },
  };
}

module.exports = { createFoodLogController };
