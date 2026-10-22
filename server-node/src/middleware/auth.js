const { parseToken } = require('../utils/jwt');
const { unauthorized } = require('../utils/response');

function requireAuth(jwtSecret) {
  return (req, res, next) => {
    const header = req.headers['authorization'] || '';
    const parts = header.split(' ');
    if (parts.length !== 2 || parts[0] !== 'Bearer') {
      unauthorized(res, 'missing or malformed authorization header');
      return;
    }

    try {
      const claims = parseToken(parts[1], jwtSecret);
      req.userId = claims.user_id;
      next();
    } catch (err) {
      unauthorized(res, 'invalid or expired token');
    }
  };
}

module.exports = { requireAuth };
