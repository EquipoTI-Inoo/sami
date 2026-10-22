function ok(res, data) {
  res.status(200).json({ data });
}

function created(res, data) {
  res.status(201).json({ data });
}

function badRequest(res, msg) {
  res.status(400).json({ error: msg });
}

function unauthorized(res, msg) {
  res.status(401).json({ error: msg });
}

function conflict(res, msg) {
  res.status(409).json({ error: msg });
}

function notFound(res, msg) {
  res.status(404).json({ error: msg });
}

function internalError(res, msg) {
  res.status(500).json({ error: msg });
}

// internalErrorf logs the real error server-side (the response msg is
// deliberately generic — never echoes internals to the client).
function internalErrorf(req, res, msg, err) {
  console.error(`${req.method} ${req.originalUrl}:`, err);
  internalError(res, msg);
}

module.exports = { ok, created, badRequest, unauthorized, conflict, notFound, internalError, internalErrorf };
