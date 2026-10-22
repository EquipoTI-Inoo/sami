const { ok, badRequest, internalErrorf } = require('../utils/response');

function createExerciseController(exercises) {
  return {
    // listExercises powers the exercise catalog browser — ?location=gym|home
    // filters to what's usable there (a "both" exercise always shows), no
    // query param returns the whole catalog.
    list(req, res) {
      const location = req.query.location || '';
      if (location !== '' && location !== 'gym' && location !== 'home') {
        badRequest(res, 'location must be gym or home');
        return;
      }

      try {
        ok(res, exercises.listAll(location));
      } catch (err) {
        internalErrorf(req, res, 'could not load the exercise catalog', err);
      }
    },
  };
}

module.exports = { createExerciseController };
