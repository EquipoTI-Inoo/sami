// Ported 1:1 from backend/checkin/checkin.go.

const DUE_AFTER_DAYS = 7; // "¿han pasado 7 días desde el último check-in?" — checked on read, no scheduler.

function isDue(lastLoggedOn, now) {
  const last = new Date(`${lastLoggedOn}T00:00:00Z`);
  const daysSince = Math.floor((now.getTime() - last.getTime()) / (1000 * 60 * 60 * 24));
  return { due: daysSince >= DUE_AFTER_DAYS, daysSince };
}

module.exports = { isDue, DUE_AFTER_DAYS };
