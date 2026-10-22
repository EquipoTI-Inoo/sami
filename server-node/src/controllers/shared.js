// Shared helpers used by more than one controller — ported from the Go
// backend's onboarding.go/programs.go, which keep these as free functions in
// the same package for exactly this reason (single source of truth for
// "what onboarding does" / "what generating a program does").

const { calculate } = require('../nutrition/calculator');
const { generate } = require('../training/program');

function generateProgramForProfile(exercises, programs, userId, goal, trainingLocation) {
  const queryLocation = trainingLocation === 'both' ? '' : trainingLocation;
  const catalog = exercises.listAll(queryLocation);

  const plan = generate(catalog, goal);
  if (plan.days.length === 0) return null;

  return programs.create(userId, goal, trainingLocation, plan.days);
}

// applyOnboarding is the single place onboarding data gets written — the
// direct REST endpoint and the chat tool-call path both go through it.
function applyOnboarding(profiles, exercises, programs, userId, req) {
  profiles.upsertProfile(userId, req);

  profiles.insertBodyMeasurement(userId, { weightKg: req.weight_kg, bodyFatPct: req.body_fat_pct });

  const calc = calculate(req.sex, req.age, req.height_cm, req.weight_kg, req.activity_level, req.goal);
  const targets = profiles.insertNutritionTargets(userId, {
    calories: calc.calories,
    proteinG: calc.proteinG,
    carbsG: calc.carbsG,
    fatG: calc.fatG,
    updatedBy: 'user',
    reason: 'onboarding',
  });

  // Best-effort: a catalog/program hiccup must never block onboarding
  // itself — it can always be regenerated later from Workouts.
  try {
    generateProgramForProfile(exercises, programs, userId, req.goal, req.training_location);
  } catch (err) {
    console.error(`applyOnboarding: could not generate initial program for user ${userId}:`, err);
  }

  const profile = profiles.getProfile(userId);
  return { profile, targets };
}

function hasRequiredOnboardingFields(req) {
  return Boolean(
    req &&
      req.age > 0 &&
      req.sex &&
      req.height_cm > 0 &&
      req.weight_kg > 0 &&
      req.activity_level &&
      req.goal &&
      req.training_location
  );
}

// startOfWeek returns the UTC timestamp for the most recent Monday (ISO week
// start), as a string comparable against the TEXT timestamp columns used
// throughout the schema.
function startOfWeek(date) {
  const d = new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()));
  let weekday = d.getUTCDay();
  if (weekday === 0) weekday = 7; // ISO: Monday=1 .. Sunday=7
  d.setUTCDate(d.getUTCDate() - (weekday - 1));
  return d.toISOString().replace('Z', '').slice(0, 23) + 'Z';
}

// authorizeFriendship resolves friendshipId to the other participant's user
// id, but only when it's accepted and userId is actually one of its two
// sides. Returns null (after writing the 404 response itself) on any
// failure — the same "not found" a non-participant would see, so this never
// confirms a friendship id's existence to someone outside it.
function authorizeFriendship(res, friendships, friendshipId, userId, notFound, internalErrorf, req) {
  const { ErrNotFound } = require('../errors');
  let friendship;
  try {
    friendship = friendships.getById(friendshipId);
  } catch (err) {
    if (err === ErrNotFound) {
      notFound(res, 'friendship not found');
      return null;
    }
    internalErrorf(req, res, 'could not load the friendship', err);
    return null;
  }
  if (friendship.status !== 'accepted') {
    notFound(res, 'friendship not found');
    return null;
  }

  if (userId === friendship.requester_id) return { friendship, otherUserId: friendship.recipient_id };
  if (userId === friendship.recipient_id) return { friendship, otherUserId: friendship.requester_id };
  notFound(res, 'friendship not found');
  return null;
}

module.exports = { generateProgramForProfile, applyOnboarding, hasRequiredOnboardingFields, startOfWeek, authorizeFriendship };
