const path = require('path');
const fs = require('fs');
const express = require('express');
const cors = require('cors');

const { loadConfig } = require('./src/config');
const { openDb, migrate } = require('./src/db');
const { UserStore } = require('./src/stores/userStore');
const { ProfileStore } = require('./src/stores/profileStore');
const { FoodStore } = require('./src/stores/foodStore');
const { ChatStore } = require('./src/stores/chatStore');
const { PendingChangeStore } = require('./src/stores/pendingChangeStore');
const { ExerciseStore } = require('./src/stores/exerciseStore');
const { WorkoutStore } = require('./src/stores/workoutStore');
const { ProgramStore } = require('./src/stores/programStore');
const { FriendshipStore } = require('./src/stores/friendshipStore');
const { SharedGoalStore } = require('./src/stores/sharedGoalStore');
const { FoodLogStore } = require('./src/stores/foodLogStore');
const { newClient } = require('./src/ai/claude');
const { setup: setupRoutes } = require('./src/routes');

const CLAUDE_MODEL = 'claude-haiku-5-5';

const cfg = loadConfig();

if (!cfg.jwtSecret) {
  console.error('JWT_SECRET is required');
  process.exit(1);
}
if (!cfg.anthropicApiKey) {
  console.error('ANTHROPIC_API_KEY is required — the coach chat is core to this app, not optional');
  process.exit(1);
}

const conn = openDb(cfg.dbPath);
migrate(conn);

const stores = {
  users: new UserStore(conn),
  profiles: new ProfileStore(conn),
  foods: new FoodStore(conn),
  chats: new ChatStore(conn),
  changes: new PendingChangeStore(conn),
  exercises: new ExerciseStore(conn),
  workouts: new WorkoutStore(conn),
  programs: new ProgramStore(conn),
  friendships: new FriendshipStore(conn),
  sharedGoals: new SharedGoalStore(conn),
  foodLog: new FoodLogStore(conn),
};

const aiClient = newClient(cfg.anthropicApiKey, CLAUDE_MODEL);

const app = express();
app.use(express.json());

// CORS_ORIGIN is comma-separated: a self-hosted 2-person app is reached from
// more than one origin at once (e.g. the PC and a phone on the same network).
const origins = cfg.corsOrigin.split(',').map((o) => o.trim());
app.use(
  cors({
    origin: origins,
    methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
    allowedHeaders: ['Authorization', 'Content-Type'],
    credentials: true,
    maxAge: 12 * 60 * 60,
  })
);

app.use(setupRoutes(cfg, stores, aiClient));

// Serves the built React frontend — this hosting has no separate nginx
// layer in front of the app, so one process handles both the API and the
// static site, unlike the Docker setup's two containers.
const publicDir = path.join(__dirname, 'public');
if (fs.existsSync(publicDir)) {
  app.use(express.static(publicDir));
  app.get('*', (req, res, next) => {
    if (req.path.startsWith('/api/')) return next();
    res.sendFile(path.join(publicDir, 'index.html'));
  });
}

app.use((err, req, res, next) => {
  console.error(`${req.method} ${req.originalUrl}:`, err);
  res.status(500).json({ error: 'internal server error' });
});

// Passenger injects PORT and reverse-proxies to it — never hardcode a port.
const port = process.env.PORT || cfg.port;
app.listen(port, () => {
  console.log(`sami-backend listening on :${port} (env=${cfg.env})`);
});
