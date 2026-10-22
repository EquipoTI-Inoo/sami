const { ErrNotFound } = require('../errors');
const { ok, created, badRequest, notFound, internalErrorf } = require('../utils/response');

function parseId(param) {
  const n = Number(param);
  return Number.isInteger(n) ? n : null;
}

function createWorkoutController(workouts, exercises) {
  // authorizeWorkout writes the response itself on failure (404, matching
  // the "not found" a non-owner would see) so every handler below can just
  // check the return value and return.
  function authorizeWorkout(req, res, workoutId, userId) {
    let owner;
    try {
      owner = workouts.getWorkoutOwner(workoutId);
    } catch (err) {
      if (err === ErrNotFound) {
        notFound(res, 'workout not found');
        return false;
      }
      internalErrorf(req, res, 'could not load the workout', err);
      return false;
    }
    if (owner !== userId) {
      notFound(res, 'workout not found');
      return false;
    }
    return true;
  }

  return {
    start(req, res) {
      try {
        const workout = workouts.create(req.userId);
        created(res, workout);
      } catch (err) {
        internalErrorf(req, res, 'could not start the workout', err);
      }
    },

    list(req, res) {
      try {
        ok(res, workouts.listForUser(req.userId));
      } catch (err) {
        internalErrorf(req, res, 'could not load your workout history', err);
      }
    },

    get(req, res) {
      const workoutId = parseId(req.params.id);
      if (workoutId === null) {
        badRequest(res, 'invalid workout id');
        return;
      }

      let workout;
      try {
        workout = workouts.getById(workoutId);
      } catch (err) {
        if (err === ErrNotFound) {
          notFound(res, 'workout not found');
          return;
        }
        internalErrorf(req, res, 'could not load the workout', err);
        return;
      }
      if (workout.user_id !== req.userId) {
        notFound(res, 'workout not found');
        return;
      }

      ok(res, workout);
    },

    // addExercise appends one exercise to an in-progress workout — Gym Mode
    // builds the exercise list incrementally as the person picks what's
    // next, rather than requiring a full plan up front.
    addExercise(req, res) {
      const workoutId = parseId(req.params.id);
      if (workoutId === null) {
        badRequest(res, 'invalid workout id');
        return;
      }
      if (!authorizeWorkout(req, res, workoutId, req.userId)) return;

      const exerciseId = req.body ? req.body.exercise_id : null;
      if (!exerciseId) {
        badRequest(res, 'exercise_id is required');
        return;
      }

      try {
        exercises.getById(exerciseId);
      } catch (err) {
        if (err === ErrNotFound) {
          badRequest(res, 'exercise does not exist');
          return;
        }
        internalErrorf(req, res, 'could not validate the exercise', err);
        return;
      }

      try {
        const workoutExercise = workouts.addExercise(workoutId, exerciseId);
        created(res, workoutExercise);
      } catch (err) {
        internalErrorf(req, res, 'could not add the exercise', err);
      }
    },

    logSet(req, res) {
      const workoutExerciseId = parseId(req.params.weId);
      if (workoutExerciseId === null) {
        badRequest(res, 'invalid workout exercise id');
        return;
      }

      let owner, workoutId;
      try {
        ({ userId: owner, workoutId } = workouts.getWorkoutExerciseOwner(workoutExerciseId));
      } catch (err) {
        if (err === ErrNotFound) {
          notFound(res, 'workout exercise not found');
          return;
        }
        internalErrorf(req, res, 'could not load the workout exercise', err);
        return;
      }
      if (owner !== req.userId || String(workoutId) !== req.params.id) {
        notFound(res, 'workout exercise not found');
        return;
      }

      const { reps, weight_kg } = req.body || {};
      if (!Number.isInteger(reps) || reps < 1 || reps > 1000) {
        badRequest(res, 'reps is required');
        return;
      }
      if (weight_kg != null && (typeof weight_kg !== 'number' || weight_kg < 0 || weight_kg > 1000)) {
        badRequest(res, 'reps is required');
        return;
      }

      try {
        const set = workouts.logSet(workoutExerciseId, reps, weight_kg);
        created(res, set);
      } catch (err) {
        internalErrorf(req, res, 'could not log the set', err);
      }
    },

    complete(req, res) {
      const workoutId = parseId(req.params.id);
      if (workoutId === null) {
        badRequest(res, 'invalid workout id');
        return;
      }
      if (!authorizeWorkout(req, res, workoutId, req.userId)) return;

      try {
        const workout = workouts.complete(workoutId);
        ok(res, workout);
      } catch (err) {
        internalErrorf(req, res, 'could not complete the workout', err);
      }
    },
  };
}

module.exports = { createWorkoutController };
