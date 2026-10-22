const { ErrNotFound } = require('../errors');
const { ok, badRequest, notFound, conflict, internalErrorf } = require('../utils/response');

function createPendingChangeController(profiles, changes) {
  function resolve(req, res, status) {
    const id = Number(req.params.id);
    if (!Number.isInteger(id)) {
      badRequest(res, 'invalid change id');
      return;
    }

    let change;
    try {
      change = changes.getById(id);
    } catch (err) {
      if (err === ErrNotFound) {
        notFound(res, 'pending change not found');
        return;
      }
      internalErrorf(req, res, 'could not load pending change', err);
      return;
    }
    if (change.user_id !== req.userId) {
      notFound(res, 'pending change not found');
      return;
    }
    if (change.status !== 'pending') {
      conflict(res, 'this change was already resolved');
      return;
    }

    // Approving a nutrition_target change writes nutrition_targets right
    // here — nothing in the chat tool-call path touches that table
    // directly ("autonomía cero sin confirmación explícita").
    if (status === 'approved' && change.change_type === 'nutrition_target') {
      const payload = change.payload;
      try {
        profiles.insertNutritionTargets(req.userId, {
          calories: payload.calories,
          proteinG: payload.protein_g,
          carbsG: payload.carbs_g,
          fatG: payload.fat_g,
          updatedBy: 'ai',
          reason: change.reason,
        });
      } catch (err) {
        internalErrorf(req, res, 'could not apply the change', err);
        return;
      }
    }

    try {
      const resolved = changes.resolve(id, status);
      ok(res, resolved);
    } catch (err) {
      internalErrorf(req, res, 'could not save your decision', err);
    }
  }

  return {
    listPending(req, res) {
      try {
        ok(res, changes.listPending(req.userId));
      } catch (err) {
        internalErrorf(req, res, 'could not load pending changes', err);
      }
    },
    approve(req, res) {
      resolve(req, res, 'approved');
    },
    reject(req, res) {
      resolve(req, res, 'rejected');
    },
  };
}

module.exports = { createPendingChangeController };
