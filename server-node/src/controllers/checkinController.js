const { ErrNotFound } = require('../errors');
const { ok, created, badRequest, internalErrorf } = require('../utils/response');
const { isDue } = require('../checkin/checkin');

function createCheckinController(profiles) {
  return {
    // getStatus tells the frontend whether a weekly check-in is due, so the
    // Dashboard can show an in-app reminder banner — in-app only, no email,
    // checked on read rather than via a real scheduler.
    getStatus(req, res) {
      let latest;
      try {
        latest = profiles.getLatestBodyMeasurement(req.userId);
      } catch (err) {
        if (err === ErrNotFound) {
          // No measurements at all means onboarding hasn't happened yet —
          // nothing to remind about.
          ok(res, { needs_checkin: false });
          return;
        }
        internalErrorf(req, res, 'could not load check-in status', err);
        return;
      }

      const { due, daysSince } = isDue(latest.logged_on, new Date());
      ok(res, { needs_checkin: due, last_logged_on: latest.logged_on, days_since_last: daysSince });
    },

    submit(req, res) {
      const body = req.body || {};
      if (typeof body.weight_kg !== 'number' || body.weight_kg < 30 || body.weight_kg > 300) {
        badRequest(res, 'missing or invalid check-in fields');
        return;
      }

      try {
        const measurement = profiles.insertBodyMeasurement(req.userId, {
          weightKg: body.weight_kg,
          waistCm: body.waist_cm,
          hipCm: body.hip_cm,
          armCm: body.arm_cm,
          bodyFatPct: body.body_fat_pct,
          note: body.note || null,
        });
        created(res, measurement);
      } catch (err) {
        internalErrorf(req, res, 'could not save your check-in', err);
      }
    },
  };
}

module.exports = { createCheckinController };
