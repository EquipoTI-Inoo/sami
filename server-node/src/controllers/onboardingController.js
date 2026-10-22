const { ErrNotFound } = require('../errors');
const { ok, created, badRequest, notFound, internalError } = require('../utils/response');
const { applyOnboarding } = require('./shared');

const SEX_VALUES = ['male', 'female'];
const ACTIVITY_VALUES = ['sedentary', 'light', 'moderate', 'active', 'very_active'];
const GOAL_VALUES = ['cut', 'maintain', 'bulk', 'performance'];
const LOCATION_VALUES = ['gym', 'home', 'both'];

function validateOnboardingRequest(req) {
  if (!Number.isInteger(req.age) || req.age < 13 || req.age > 100) return false;
  if (!SEX_VALUES.includes(req.sex)) return false;
  if (typeof req.height_cm !== 'number' || req.height_cm < 100 || req.height_cm > 250) return false;
  if (typeof req.weight_kg !== 'number' || req.weight_kg < 30 || req.weight_kg > 300) return false;
  if (req.body_fat_pct != null && (typeof req.body_fat_pct !== 'number' || req.body_fat_pct < 3 || req.body_fat_pct > 70)) return false;
  if (!ACTIVITY_VALUES.includes(req.activity_level)) return false;
  if (!GOAL_VALUES.includes(req.goal)) return false;
  if (!LOCATION_VALUES.includes(req.training_location)) return false;
  return true;
}

function createOnboardingController(profiles, exercises, programs) {
  return {
    submit(req, res) {
      const body = req.body || {};
      if (!validateOnboardingRequest(body)) {
        badRequest(res, 'missing or invalid onboarding fields');
        return;
      }

      try {
        const { profile, targets } = applyOnboarding(profiles, exercises, programs, req.userId, body);
        created(res, { profile, targets });
      } catch (err) {
        internalError(res, 'could not save your profile');
      }
    },

    getProfile(req, res) {
      try {
        const profile = profiles.getProfile(req.userId);
        ok(res, profile);
      } catch (err) {
        if (err === ErrNotFound) {
          notFound(res, 'no profile yet — complete onboarding first');
          return;
        }
        internalError(res, 'could not load profile');
      }
    },

    getNutritionTargets(req, res) {
      try {
        const targets = profiles.getLatestNutritionTargets(req.userId);
        ok(res, targets);
      } catch (err) {
        if (err === ErrNotFound) {
          notFound(res, 'no nutrition targets yet — complete onboarding first');
          return;
        }
        internalError(res, 'could not load nutrition targets');
      }
    },
  };
}

module.exports = { createOnboardingController };
