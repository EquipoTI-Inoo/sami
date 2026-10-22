// Shared sentinel errors — stores throw these, controllers check `err ===
// ErrNotFound` by identity, same pattern as Go's errors.Is(err, ErrNotFound).
const ErrNotFound = new Error('not found');
const ErrEmailTaken = new Error('email already registered');

module.exports = { ErrNotFound, ErrEmailTaken };
