const { ErrNotFound } = require('../errors');
const { ok, notFound, internalError, internalErrorf } = require('../utils/response');
const { buildPlan } = require('../nutrition/mealplan');

function createNutritionController(profiles, foods) {
  return {
    getMealPlan(req, res) {
      let profile;
      try {
        profile = profiles.getProfile(req.userId);
      } catch (err) {
        if (err === ErrNotFound) {
          notFound(res, 'no profile yet — complete onboarding first');
          return;
        }
        internalError(res, 'could not load profile');
        return;
      }

      let targets;
      try {
        targets = profiles.getLatestNutritionTargets(req.userId);
      } catch (err) {
        if (err === ErrNotFound) {
          notFound(res, 'no nutrition targets yet — complete onboarding first');
          return;
        }
        internalError(res, 'could not load nutrition targets');
        return;
      }

      let catalog;
      try {
        catalog = foods.listAll();
      } catch (err) {
        internalError(res, 'could not load food catalog');
        return;
      }

      const plan = buildPlan(catalog, targets, { likes: profile.likes, dislikes: profile.dislikes, restrictions: profile.restrictions });
      ok(res, plan);
    },

    // getTargetsHistory powers the "dashboard de ajustes" timeline — every
    // row in nutrition_targets is append-only, so this is simply the full
    // history, newest first, with no separate audit table needed.
    getTargetsHistory(req, res) {
      try {
        ok(res, profiles.getNutritionTargetsHistory(req.userId));
      } catch (err) {
        internalErrorf(req, res, 'could not load your plan history', err);
      }
    },

    // listFoods returns the whole curated catalog — the frontend's food-log
    // search picker filters it client-side, same approach as the exercise
    // picker, since the catalog is small (tens, not thousands, of rows).
    listFoods(req, res) {
      try {
        ok(res, foods.listAll());
      } catch (err) {
        internalErrorf(req, res, 'could not load the food catalog', err);
      }
    },
  };
}

module.exports = { createNutritionController };
