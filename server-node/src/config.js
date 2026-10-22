function getEnv(key, fallback) {
  const v = process.env[key];
  return v && v !== '' ? v : fallback;
}

function loadConfig() {
  return {
    env: getEnv('ENV', 'development'),
    port: getEnv('PORT', '3000'),
    jwtSecret: getEnv('JWT_SECRET', ''),
    // Comma-separated list — see routes.js for why.
    corsOrigin: getEnv('CORS_ORIGIN', 'http://localhost:5173'),
    dbPath: getEnv('DB_PATH', './data/sami.db'),
    anthropicApiKey: getEnv('ANTHROPIC_API_KEY', ''),
  };
}

module.exports = { loadConfig };
