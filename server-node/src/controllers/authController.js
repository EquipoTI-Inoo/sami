const { ErrEmailTaken, ErrNotFound } = require('../errors');
const { hashPassword, checkPassword } = require('../utils/password');
const { generateToken } = require('../utils/jwt');
const { ok, created, badRequest, unauthorized, conflict, internalError } = require('../utils/response');

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

function createAuthController(users, jwtSecret) {
  return {
    register(req, res) {
      const { email, password } = req.body || {};
      if (!email || !EMAIL_RE.test(email) || !password || password.length < 8) {
        badRequest(res, 'email and password (min 8 chars) are required');
        return;
      }

      let hash;
      try {
        hash = hashPassword(password);
      } catch {
        internalError(res, 'could not process password');
        return;
      }

      let user;
      try {
        user = users.create(email, hash);
      } catch (err) {
        if (err === ErrEmailTaken) {
          conflict(res, 'email already registered');
          return;
        }
        internalError(res, 'could not create user');
        return;
      }

      let token;
      try {
        token = generateToken(user.id, jwtSecret);
      } catch {
        internalError(res, 'could not generate token');
        return;
      }

      created(res, { token, user });
    },

    login(req, res) {
      const { email, password } = req.body || {};
      if (!email || !EMAIL_RE.test(email) || !password) {
        badRequest(res, 'email and password are required');
        return;
      }

      let row;
      try {
        row = users.findByEmailWithPassword(email);
      } catch (err) {
        if (err === ErrNotFound) {
          unauthorized(res, 'invalid email or password');
          return;
        }
        internalError(res, 'could not log in');
        return;
      }

      if (!checkPassword(row.password_hash, password)) {
        unauthorized(res, 'invalid email or password');
        return;
      }

      let token;
      try {
        token = generateToken(row.id, jwtSecret);
      } catch {
        internalError(res, 'could not generate token');
        return;
      }

      ok(res, { token, user: { id: row.id, email: row.email, created_at: row.created_at } });
    },

    me(req, res) {
      try {
        const user = users.findById(req.userId);
        ok(res, user);
      } catch (err) {
        if (err === ErrNotFound) {
          unauthorized(res, 'user no longer exists');
          return;
        }
        internalError(res, 'could not load user');
      }
    },
  };
}

module.exports = { createAuthController };
