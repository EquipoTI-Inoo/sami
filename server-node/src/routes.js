const express = require('express');

const { requireAuth } = require('./middleware/auth');
const { createAuthController } = require('./controllers/authController');
const { createOnboardingController } = require('./controllers/onboardingController');
const { createNutritionController } = require('./controllers/nutritionController');
const { createChatController } = require('./controllers/chatController');
const { createCheckinController } = require('./controllers/checkinController');
const { createPendingChangeController } = require('./controllers/pendingChangesController');
const { createExerciseController } = require('./controllers/exercisesController');
const { createWorkoutController } = require('./controllers/workoutsController');
const { createProgramController } = require('./controllers/programsController');
const { createFriendController } = require('./controllers/friendsController');
const { createSharedGoalController } = require('./controllers/sharedGoalsController');
const { createFoodLogController } = require('./controllers/foodLogController');

function wrap(handler) {
  return (req, res, next) => {
    try {
      const result = handler(req, res);
      if (result && typeof result.catch === 'function') result.catch(next);
    } catch (err) {
      next(err);
    }
  };
}

function setup(cfg, stores, aiClient) {
  const { users, profiles, foods, chats, changes, exercises, workouts, programs, friendships, sharedGoals, foodLog } = stores;

  const router = express.Router();

  router.get('/health', (req, res) => res.status(200).json({ status: 'ok' }));

  const auth = createAuthController(users, cfg.jwtSecret);
  const onboarding = createOnboardingController(profiles, exercises, programs);
  const nutritionCtrl = createNutritionController(profiles, foods);
  const chatCtrl = createChatController(profiles, chats, changes, exercises, programs, aiClient);
  const checkinCtrl = createCheckinController(profiles);
  const pendingChangeCtrl = createPendingChangeController(profiles, changes);
  const exerciseCtrl = createExerciseController(exercises);
  const workoutCtrl = createWorkoutController(workouts, exercises);
  const programCtrl = createProgramController(profiles, exercises, programs);
  const friendCtrl = createFriendController(users, friendships, profiles, workouts);
  const sharedGoalCtrl = createSharedGoalController(friendships, sharedGoals, users, workouts, profiles);
  const foodLogCtrl = createFoodLogController(foodLog, foods, profiles);

  const v1 = express.Router();

  v1.post('/auth/register', wrap(auth.register));
  v1.post('/auth/login', wrap(auth.login));

  const protect = requireAuth(cfg.jwtSecret);

  v1.get('/me', protect, wrap(auth.me));
  v1.post('/onboarding', protect, wrap(onboarding.submit));
  v1.get('/profile', protect, wrap(onboarding.getProfile));
  v1.get('/nutrition/targets', protect, wrap(onboarding.getNutritionTargets));
  v1.get('/nutrition/meal-plan', protect, wrap(nutritionCtrl.getMealPlan));
  v1.get('/nutrition/targets/history', protect, wrap(nutritionCtrl.getTargetsHistory));
  v1.get('/foods', protect, wrap(nutritionCtrl.listFoods));
  v1.post('/food-log', protect, wrap(foodLogCtrl.create));
  v1.get('/food-log', protect, wrap(foodLogCtrl.getDay));
  v1.delete('/food-log/:id', protect, wrap(foodLogCtrl.delete));
  v1.post('/chat', protect, wrap(chatCtrl.sendMessage));
  v1.get('/chat/history', protect, wrap(chatCtrl.getHistory));
  v1.get('/checkin/status', protect, wrap(checkinCtrl.getStatus));
  v1.post('/checkin', protect, wrap(checkinCtrl.submit));
  v1.get('/pending-changes', protect, wrap(pendingChangeCtrl.listPending));
  v1.post('/pending-changes/:id/approve', protect, wrap(pendingChangeCtrl.approve));
  v1.post('/pending-changes/:id/reject', protect, wrap(pendingChangeCtrl.reject));
  v1.get('/exercises', protect, wrap(exerciseCtrl.list));
  v1.post('/workouts', protect, wrap(workoutCtrl.start));
  v1.get('/workouts', protect, wrap(workoutCtrl.list));
  v1.get('/workouts/:id', protect, wrap(workoutCtrl.get));
  v1.post('/workouts/:id/exercises', protect, wrap(workoutCtrl.addExercise));
  v1.post('/workouts/:id/exercises/:weId/sets', protect, wrap(workoutCtrl.logSet));
  v1.post('/workouts/:id/complete', protect, wrap(workoutCtrl.complete));
  v1.get('/programs/latest', protect, wrap(programCtrl.getLatest));
  v1.post('/programs/generate', protect, wrap(programCtrl.generate));
  v1.post('/friends/requests', protect, wrap(friendCtrl.sendRequest));
  v1.get('/friends/requests', protect, wrap(friendCtrl.listIncoming));
  v1.post('/friends/requests/:id/accept', protect, wrap(friendCtrl.accept));
  v1.post('/friends/requests/:id/reject', protect, wrap(friendCtrl.reject));
  v1.get('/friends', protect, wrap(friendCtrl.listFriends));
  v1.get('/friends/:id/profile', protect, wrap(friendCtrl.getProfile));
  v1.post('/friends/:id/goals', protect, wrap(sharedGoalCtrl.create));
  v1.get('/friends/:id/goals', protect, wrap(sharedGoalCtrl.list));
  v1.post('/goals/:goalId/complete', protect, wrap(sharedGoalCtrl.complete));

  router.use('/api/v1', v1);

  return router;
}

module.exports = { setup };
