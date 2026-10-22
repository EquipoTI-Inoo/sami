const bcrypt = require('bcryptjs');

const COST = 10; // matches Go's bcrypt.DefaultCost

function hashPassword(password) {
  return bcrypt.hashSync(password, COST);
}

function checkPassword(hash, password) {
  return bcrypt.compareSync(password, hash);
}

module.exports = { hashPassword, checkPassword };
