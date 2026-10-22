const { ErrNotFound } = require('../errors');
const { ok, created, badRequest, notFound, conflict, internalErrorf } = require('../utils/response');
const { startOfWeek, authorizeFriendship } = require('./shared');

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

function createFriendController(users, friendships, profiles, workouts) {
  function resolveIncoming(req, res, status) {
    const id = Number(req.params.id);
    if (!Number.isInteger(id)) {
      badRequest(res, 'invalid request id');
      return;
    }

    let friendship;
    try {
      friendship = friendships.getById(id);
    } catch (err) {
      if (err === ErrNotFound) {
        notFound(res, 'friend request not found');
        return;
      }
      internalErrorf(req, res, 'could not load the friend request', err);
      return;
    }
    if (friendship.recipient_id !== req.userId) {
      notFound(res, 'friend request not found');
      return;
    }
    if (friendship.status !== 'pending') {
      conflict(res, 'this request was already resolved');
      return;
    }

    try {
      const resolved = friendships.resolve(id, status);
      ok(res, resolved);
    } catch (err) {
      internalErrorf(req, res, 'could not save your decision', err);
    }
  }

  return {
    // sendRequest looks the target up by email — nothing is visible to
    // either side until the recipient explicitly accepts.
    sendRequest(req, res) {
      const email = req.body ? req.body.email : null;
      if (!email || !EMAIL_RE.test(email)) {
        badRequest(res, 'a valid email is required');
        return;
      }

      let target;
      try {
        target = users.findByEmail(email);
      } catch (err) {
        if (err === ErrNotFound) {
          notFound(res, 'no account with that email');
          return;
        }
        internalErrorf(req, res, 'could not look up that email', err);
        return;
      }
      if (target.id === req.userId) {
        badRequest(res, "you can't send a friend request to yourself");
        return;
      }

      let existing = null;
      try {
        existing = friendships.findBetween(req.userId, target.id);
      } catch (err) {
        if (err !== ErrNotFound) {
          internalErrorf(req, res, 'could not check existing friendships', err);
          return;
        }
      }
      if (existing && (existing.status === 'accepted' || existing.status === 'pending')) {
        conflict(res, "you're already friends, or a request between you two is already pending");
        return;
      }
      // A past 'rejected' friendship falls through — a fresh request is allowed.

      try {
        const friendship = friendships.create(req.userId, target.id);
        created(res, friendship);
      } catch (err) {
        internalErrorf(req, res, 'could not send the friend request', err);
      }
    },

    listIncoming(req, res) {
      try {
        ok(res, friendships.listIncomingPending(req.userId));
      } catch (err) {
        internalErrorf(req, res, 'could not load incoming requests', err);
      }
    },

    listFriends(req, res) {
      try {
        ok(res, friendships.listAccepted(req.userId));
      } catch (err) {
        internalErrorf(req, res, 'could not load your friends', err);
      }
    },

    accept(req, res) {
      resolveIncoming(req, res, 'accepted');
    },

    reject(req, res) {
      resolveIncoming(req, res, 'rejected');
    },

    // getProfile returns a friend's shared profile — current plan/goal,
    // recent check-ins and workout counts.
    getProfile(req, res) {
      const friendshipId = Number(req.params.id);
      if (!Number.isInteger(friendshipId)) {
        badRequest(res, 'invalid friendship id');
        return;
      }

      const auth = authorizeFriendship(res, friendships, friendshipId, req.userId, notFound, internalErrorf, req);
      if (!auth) return;
      const friendUserId = auth.otherUserId;

      let friendUser;
      try {
        friendUser = users.findById(friendUserId);
      } catch (err) {
        internalErrorf(req, res, "could not load your friend's account", err);
        return;
      }

      let profile = null;
      try {
        profile = profiles.getProfile(friendUserId);
      } catch (err) {
        if (err !== ErrNotFound) {
          internalErrorf(req, res, "could not load your friend's profile", err);
          return;
        }
      }

      let targets = null;
      try {
        targets = profiles.getLatestNutritionTargets(friendUserId);
      } catch (err) {
        if (err !== ErrNotFound) {
          internalErrorf(req, res, "could not load your friend's targets", err);
          return;
        }
      }

      let checkins;
      try {
        checkins = profiles.listRecentBodyMeasurements(friendUserId, 8);
      } catch (err) {
        internalErrorf(req, res, "could not load your friend's check-ins", err);
        return;
      }

      let workoutsThisWeek, totalWorkouts;
      try {
        workoutsThisWeek = workouts.countCompletedSince(friendUserId, startOfWeek(new Date()));
        totalWorkouts = workouts.countCompleted(friendUserId);
      } catch (err) {
        internalErrorf(req, res, "could not load your friend's workouts", err);
        return;
      }

      ok(res, {
        email: friendUser.email,
        profile,
        targets,
        recent_checkins: checkins,
        workouts_this_week: workoutsThisWeek,
        total_workouts: totalWorkouts,
      });
    },
  };
}

module.exports = { createFriendController };
