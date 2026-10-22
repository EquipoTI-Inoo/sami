const { ErrNotFound } = require('../errors');
const { ok, created, badRequest, notFound, internalErrorf } = require('../utils/response');
const { startOfWeek, authorizeFriendship } = require('./shared');

const GOAL_TYPES = ['workouts_per_week', 'weight_target', 'custom'];

function createSharedGoalController(friendships, goals, users, workouts, profiles) {
  // computeProgress reports each side of the friendship's own relevant
  // metric for the goal's type — a shared goal's "progress" is read, never
  // a stored number, so it's always consistent with the real workouts/check-ins.
  function computeProgress(goal, friendship) {
    const progress = {};

    for (const userId of [friendship.requester_id, friendship.recipient_id]) {
      const user = users.findById(userId);
      const p = { user_id: userId, email: user.email };

      if (goal.goal_type === 'workouts_per_week') {
        p.workouts_this_week = workouts.countCompletedSince(userId, startOfWeek(new Date()));
      } else if (goal.goal_type === 'weight_target') {
        try {
          const latest = profiles.getLatestBodyMeasurement(userId);
          p.current_weight_kg = latest.weight_kg;
        } catch (err) {
          if (err !== ErrNotFound) throw err;
        }
      }

      progress[String(userId)] = p;
    }

    return progress;
  }

  return {
    create(req, res) {
      const friendshipId = Number(req.params.id);
      if (!Number.isInteger(friendshipId)) {
        badRequest(res, 'invalid friendship id');
        return;
      }
      if (!authorizeFriendship(res, friendships, friendshipId, req.userId, notFound, internalErrorf, req)) return;

      const body = req.body || {};
      if (!GOAL_TYPES.includes(body.goal_type) || !body.description) {
        badRequest(res, 'goal_type and description are required');
        return;
      }

      try {
        const goal = goals.create(friendshipId, req.userId, body);
        created(res, goal);
      } catch (err) {
        internalErrorf(req, res, 'could not create the shared goal', err);
      }
    },

    // list returns every shared goal for the friendship with each
    // participant's progress computed live.
    list(req, res) {
      const friendshipId = Number(req.params.id);
      if (!Number.isInteger(friendshipId)) {
        badRequest(res, 'invalid friendship id');
        return;
      }
      const auth = authorizeFriendship(res, friendships, friendshipId, req.userId, notFound, internalErrorf, req);
      if (!auth) return;

      let goalList;
      try {
        goalList = goals.listForFriendship(friendshipId);
      } catch (err) {
        internalErrorf(req, res, 'could not load shared goals', err);
        return;
      }

      try {
        for (const g of goalList) {
          g.progress = computeProgress(g, auth.friendship);
        }
      } catch (err) {
        internalErrorf(req, res, 'could not compute goal progress', err);
        return;
      }

      ok(res, goalList);
    },

    // complete lets either participant mark a shared goal done — it
    // belongs to the friendship, not to whoever created it.
    complete(req, res) {
      const id = Number(req.params.goalId);
      if (!Number.isInteger(id)) {
        badRequest(res, 'invalid goal id');
        return;
      }

      let goal;
      try {
        goal = goals.getById(id);
      } catch (err) {
        if (err === ErrNotFound) {
          notFound(res, 'goal not found');
          return;
        }
        internalErrorf(req, res, 'could not load the goal', err);
        return;
      }

      if (!authorizeFriendship(res, friendships, goal.friendship_id, req.userId, notFound, internalErrorf, req)) return;

      try {
        const completed = goals.complete(id);
        ok(res, completed);
      } catch (err) {
        internalErrorf(req, res, 'could not complete the goal', err);
      }
    },
  };
}

module.exports = { createSharedGoalController };
