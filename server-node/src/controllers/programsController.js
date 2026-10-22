const { ErrNotFound } = require('../errors');
const { ok, created, notFound, internalError, internalErrorf } = require('../utils/response');
const { generateProgramForProfile } = require('./shared');

function createProgramController(profiles, exercises, programs) {
  return {
    getLatest(req, res) {
      try {
        const program = programs.getLatest(req.userId);
        ok(res, program);
      } catch (err) {
        if (err === ErrNotFound) {
          notFound(res, 'no program yet — generate one first');
          return;
        }
        internalErrorf(req, res, 'could not load your program', err);
      }
    },

    // Generate lets the person request a fresh program on demand (e.g.
    // after changing equipment) without re-running the whole onboarding flow.
    generate(req, res) {
      let profile;
      try {
        profile = profiles.getProfile(req.userId);
      } catch (err) {
        if (err === ErrNotFound) {
          notFound(res, 'complete onboarding first');
          return;
        }
        internalErrorf(req, res, 'could not load your profile', err);
        return;
      }

      let program;
      try {
        program = generateProgramForProfile(exercises, programs, req.userId, profile.goal, profile.training_location);
      } catch (err) {
        internalErrorf(req, res, 'could not generate your program', err);
        return;
      }
      if (!program) {
        internalError(res, 'the exercise catalog has nothing to build a program from');
        return;
      }

      created(res, program);
    },
  };
}

module.exports = { createProgramController };
