const jwt = require('jsonwebtoken');

const TOKEN_TTL = '30d'; // two self-hosted users, long-lived session over security-critical rotation

function generateToken(userId, secret) {
  return jwt.sign({ user_id: userId }, secret, { expiresIn: TOKEN_TTL });
}

function parseToken(token, secret) {
  const claims = jwt.verify(token, secret); // throws on invalid/expired
  return claims;
}

module.exports = { generateToken, parseToken };
