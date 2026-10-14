package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"coachia-backend/ai"
	"coachia-backend/config"
	"coachia-backend/db"
	"coachia-backend/models"
	"coachia-backend/routes"
	"coachia-backend/stores"
)

func setupTestRouter(t *testing.T) http.Handler {
	t.Helper()
	router, _ := setupTestRouterAndStore(t)
	return router
}

func setupTestRouterAndStore(t *testing.T) (http.Handler, *stores.PendingChangeStore) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	if err := db.Migrate(conn); err != nil {
		t.Fatalf("migrating test db: %v", err)
	}

	cfg := config.Config{JWTSecret: "test-secret", CORSOrigin: "http://localhost:5173"}
	users := stores.NewUserStore(conn)
	profiles := stores.NewProfileStore(conn)
	foods := stores.NewFoodStore(conn)
	chats := stores.NewChatStore(conn)
	changes := stores.NewPendingChangeStore(conn)
	exercises := stores.NewExerciseStore(conn)
	workouts := stores.NewWorkoutStore(conn)
	programs := stores.NewProgramStore(conn)
	friendships := stores.NewFriendshipStore(conn)
	sharedGoals := stores.NewSharedGoalStore(conn)
	foodLog := stores.NewFoodLogStore(conn)
	aiClient := ai.NewClient("test-key", "claude-haiku-5-5")
	return routes.Setup(cfg, users, profiles, foods, chats, changes, exercises, workouts, programs, friendships, sharedGoals, foodLog, aiClient), changes
}

func doJSON(t *testing.T, router http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshaling request body: %v", err)
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	router := setupTestRouter(t)
	rec := doJSON(t, router, http.MethodGet, "/health", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRegisterLoginMe(t *testing.T) {
	router := setupTestRouter(t)

	registerReq := models.RegisterRequest{Email: "pair@coachia.local", Password: "password123"}

	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/register", registerReq, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var registerResp struct {
		Data models.AuthResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &registerResp); err != nil {
		t.Fatalf("decoding register response: %v", err)
	}
	if registerResp.Data.Token == "" {
		t.Fatal("expected a token in register response")
	}
	if registerResp.Data.User.Email != registerReq.Email {
		t.Fatalf("expected email %q, got %q", registerReq.Email, registerResp.Data.User.Email)
	}

	// Duplicate registration must be rejected.
	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/register", registerReq, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate register: expected 409, got %d", rec.Code)
	}

	// Wrong password must be rejected.
	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/login", models.LoginRequest{
		Email: registerReq.Email, Password: "wrong-password",
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad login: expected 401, got %d", rec.Code)
	}

	// Correct login.
	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/login", models.LoginRequest{
		Email: registerReq.Email, Password: registerReq.Password,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var loginResp struct {
		Data models.AuthResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decoding login response: %v", err)
	}

	// /me without a token is rejected.
	rec = doJSON(t, router, http.MethodGet, "/api/v1/me", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("/me without token: expected 401, got %d", rec.Code)
	}

	// /me with a valid token succeeds.
	rec = doJSON(t, router, http.MethodGet, "/api/v1/me", nil, loginResp.Data.Token)
	if rec.Code != http.StatusOK {
		t.Fatalf("/me with token: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var meResp struct {
		Data models.User `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &meResp); err != nil {
		t.Fatalf("decoding /me response: %v", err)
	}
	if meResp.Data.Email != registerReq.Email {
		t.Fatalf("expected /me email %q, got %q", registerReq.Email, meResp.Data.Email)
	}
}

func TestOnboardingFlow(t *testing.T) {
	router := setupTestRouter(t)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Email: "onboarding@coachia.local", Password: "password123",
	}, "")
	var reg struct {
		Data models.AuthResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &reg); err != nil {
		t.Fatalf("decoding register response: %v", err)
	}
	token := reg.Data.Token

	// Before onboarding, profile and targets don't exist yet.
	rec = doJSON(t, router, http.MethodGet, "/api/v1/profile", nil, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("profile before onboarding: expected 404, got %d", rec.Code)
	}

	onboardingReq := models.OnboardingRequest{
		Age: 30, Sex: "male", HeightCm: 180, WeightKg: 80,
		ActivityLevel: "moderate", Goal: "cut", TrainingLocation: "gym",
		Likes: []string{"pollo", "arroz"}, Dislikes: []string{"pescado"},
	}
	rec = doJSON(t, router, http.MethodPost, "/api/v1/onboarding", onboardingReq, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("onboarding: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var onboardingResp struct {
		Data models.OnboardingResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &onboardingResp); err != nil {
		t.Fatalf("decoding onboarding response: %v", err)
	}
	if onboardingResp.Data.Targets.Calories <= 0 {
		t.Fatalf("expected positive calorie target, got %d", onboardingResp.Data.Targets.Calories)
	}
	if len(onboardingResp.Data.Profile.Likes) != 2 {
		t.Fatalf("expected 2 likes round-tripped, got %d", len(onboardingResp.Data.Profile.Likes))
	}

	// Profile and targets now exist.
	rec = doJSON(t, router, http.MethodGet, "/api/v1/profile", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("profile after onboarding: expected 200, got %d", rec.Code)
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/nutrition/targets", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("targets after onboarding: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/nutrition/meal-plan", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("meal-plan after onboarding: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var mealPlanResp struct {
		Data models.MealPlan `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &mealPlanResp); err != nil {
		t.Fatalf("decoding meal-plan response: %v", err)
	}
	if len(mealPlanResp.Data.Meals) != 3 {
		t.Fatalf("expected 3 meals, got %d", len(mealPlanResp.Data.Meals))
	}
	if mealPlanResp.Data.Calories <= 0 {
		t.Fatalf("expected positive meal plan calories, got %d", mealPlanResp.Data.Calories)
	}
}

// TestPendingChangeApproveAppliesTargets exercises the full confirmation
// contract from CoachIA PRD §5.5 ("autonomía cero sin confirmación
// explícita"): a proposed nutrition_target change must sit as pending and
// must NOT touch nutrition_targets until the user approves it explicitly.
// The proposal itself is created directly against the store rather than via
// chat, since chat requires a live Claude API call this test doesn't make.
func TestPendingChangeApproveAppliesTargets(t *testing.T) {
	router, changes := setupTestRouterAndStore(t)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Email: "pending@coachia.local", Password: "password123",
	}, "")
	var reg struct {
		Data models.AuthResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &reg); err != nil {
		t.Fatalf("decoding register response: %v", err)
	}
	token := reg.Data.Token
	userID := reg.Data.User.ID

	onboardingReq := models.OnboardingRequest{
		Age: 30, Sex: "male", HeightCm: 180, WeightKg: 80,
		ActivityLevel: "moderate", Goal: "cut", TrainingLocation: "gym",
	}
	rec = doJSON(t, router, http.MethodPost, "/api/v1/onboarding", onboardingReq, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("onboarding: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	payload, err := json.Marshal(models.NutritionTargetChangePayload{Calories: 2000, ProteinG: 150, CarbsG: 200, FatG: 60})
	if err != nil {
		t.Fatalf("marshaling payload: %v", err)
	}
	change, err := changes.Create(userID, "nutrition_target", payload, "no bajó de peso en dos semanas")
	if err != nil {
		t.Fatalf("creating pending change: %v", err)
	}

	// Listed as pending before any decision.
	rec = doJSON(t, router, http.MethodGet, "/api/v1/pending-changes", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list pending: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var listResp struct {
		Data []models.PendingChange `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decoding list response: %v", err)
	}
	if len(listResp.Data) != 1 {
		t.Fatalf("expected 1 pending change, got %d", len(listResp.Data))
	}

	// Targets must still be whatever onboarding calculated — approval hasn't happened yet.
	rec = doJSON(t, router, http.MethodGet, "/api/v1/nutrition/targets", nil, token)
	var beforeResp struct {
		Data models.NutritionTargets `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &beforeResp); err != nil {
		t.Fatalf("decoding targets response: %v", err)
	}
	if beforeResp.Data.Calories == 2000 {
		t.Fatal("targets were updated before approval — autonomy-without-confirmation rule was violated")
	}

	path := fmt.Sprintf("/api/v1/pending-changes/%d/approve", change.ID)
	rec = doJSON(t, router, http.MethodPost, path, nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/nutrition/targets", nil, token)
	var afterResp struct {
		Data models.NutritionTargets `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &afterResp); err != nil {
		t.Fatalf("decoding targets response: %v", err)
	}
	if afterResp.Data.Calories != 2000 || afterResp.Data.UpdatedBy != "ai" {
		t.Fatalf("expected approved targets (2000 kcal, updated_by=ai), got %+v", afterResp.Data)
	}

	// A second approval attempt must be rejected — it was already resolved.
	rec = doJSON(t, router, http.MethodPost, path, nil, token)
	if rec.Code != http.StatusConflict {
		t.Fatalf("re-approve: expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPendingChangeRejectDoesNotApplyTargets(t *testing.T) {
	router, changes := setupTestRouterAndStore(t)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Email: "reject@coachia.local", Password: "password123",
	}, "")
	var reg struct {
		Data models.AuthResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &reg); err != nil {
		t.Fatalf("decoding register response: %v", err)
	}
	token := reg.Data.Token
	userID := reg.Data.User.ID

	rec = doJSON(t, router, http.MethodPost, "/api/v1/onboarding", models.OnboardingRequest{
		Age: 30, Sex: "female", HeightCm: 165, WeightKg: 60,
		ActivityLevel: "light", Goal: "maintain", TrainingLocation: "home",
	}, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("onboarding: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	payload, _ := json.Marshal(models.NutritionTargetChangePayload{Calories: 1800, ProteinG: 120, CarbsG: 180, FatG: 50})
	change, err := changes.Create(userID, "nutrition_target", payload, "quiere probar bajar un poco")
	if err != nil {
		t.Fatalf("creating pending change: %v", err)
	}

	rec = doJSON(t, router, http.MethodPost, fmt.Sprintf("/api/v1/pending-changes/%d/reject", change.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("reject: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/nutrition/targets", nil, token)
	var afterResp struct {
		Data models.NutritionTargets `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &afterResp); err != nil {
		t.Fatalf("decoding targets response: %v", err)
	}
	if afterResp.Data.Calories == 1800 {
		t.Fatal("rejected change was applied to targets anyway")
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/pending-changes", nil, token)
	var listResp struct {
		Data []models.PendingChange `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decoding list response: %v", err)
	}
	if len(listResp.Data) != 0 {
		t.Fatalf("expected 0 pending changes after rejection, got %d", len(listResp.Data))
	}
}

func registerUser(t *testing.T, router http.Handler, email string) string {
	t.Helper()
	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Email: email, Password: "password123",
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register %s: expected 201, got %d: %s", email, rec.Code, rec.Body.String())
	}
	var resp struct {
		Data models.AuthResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding register response: %v", err)
	}
	return resp.Data.Token
}

func TestWorkoutFlow(t *testing.T) {
	router := setupTestRouter(t)
	token := registerUser(t, router, "lifter@coachia.local")

	// Catalog is filterable by location — "home" includes "both" exercises.
	rec := doJSON(t, router, http.MethodGet, "/api/v1/exercises?location=home", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list exercises: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var exercisesResp struct {
		Data []models.Exercise `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &exercisesResp); err != nil {
		t.Fatalf("decoding exercises response: %v", err)
	}
	if len(exercisesResp.Data) == 0 {
		t.Fatal("expected a non-empty exercise catalog for location=home")
	}
	for _, e := range exercisesResp.Data {
		if e.Location == "gym" {
			t.Fatalf("location=home returned a gym-only exercise: %+v", e)
		}
	}
	exerciseID := exercisesResp.Data[0].ID

	rec = doJSON(t, router, http.MethodPost, "/api/v1/workouts", nil, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("start workout: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var workoutResp struct {
		Data models.Workout `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &workoutResp); err != nil {
		t.Fatalf("decoding workout response: %v", err)
	}
	workoutID := workoutResp.Data.ID

	rec = doJSON(t, router, http.MethodPost, fmt.Sprintf("/api/v1/workouts/%d/exercises", workoutID),
		models.AddWorkoutExerciseRequest{ExerciseID: exerciseID}, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add exercise: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var weResp struct {
		Data models.WorkoutExercise `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &weResp); err != nil {
		t.Fatalf("decoding workout exercise response: %v", err)
	}
	workoutExerciseID := weResp.Data.ID

	weight := 40.0
	setPath := fmt.Sprintf("/api/v1/workouts/%d/exercises/%d/sets", workoutID, workoutExerciseID)
	rec = doJSON(t, router, http.MethodPost, setPath, models.LogSetRequest{Reps: 10, WeightKg: &weight}, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("log set: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, router, http.MethodPost, setPath, models.LogSetRequest{Reps: 8, WeightKg: &weight}, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("log second set: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var set2Resp struct {
		Data models.WorkoutSet `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &set2Resp); err != nil {
		t.Fatalf("decoding set response: %v", err)
	}
	if set2Resp.Data.SetNumber != 2 {
		t.Fatalf("expected the second logged set to be set_number 2, got %d", set2Resp.Data.SetNumber)
	}

	rec = doJSON(t, router, http.MethodPost, fmt.Sprintf("/api/v1/workouts/%d/complete", workoutID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("complete workout: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodGet, fmt.Sprintf("/api/v1/workouts/%d", workoutID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("get workout: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var fullResp struct {
		Data models.Workout `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &fullResp); err != nil {
		t.Fatalf("decoding full workout response: %v", err)
	}
	if fullResp.Data.CompletedAt == nil {
		t.Fatal("expected completed_at to be set after completing the workout")
	}
	if len(fullResp.Data.Exercises) != 1 || len(fullResp.Data.Exercises[0].Sets) != 2 {
		t.Fatalf("expected 1 exercise with 2 sets nested, got %+v", fullResp.Data.Exercises)
	}

	// A second user must never be able to see or write to this workout.
	otherToken := registerUser(t, router, "other-lifter@coachia.local")
	rec = doJSON(t, router, http.MethodGet, fmt.Sprintf("/api/v1/workouts/%d", workoutID), nil, otherToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("other user get workout: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, router, http.MethodPost, setPath, models.LogSetRequest{Reps: 5}, otherToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("other user log set: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestProgramGeneratedAtOnboarding confirms applyOnboarding's best-effort
// program generation actually produces a usable 3-day program with the
// goal's rep scheme, reachable right after onboarding with no extra call.
func TestProgramGeneratedAtOnboarding(t *testing.T) {
	router := setupTestRouter(t)
	token := registerUser(t, router, "lifter-program@coachia.local")

	rec := doJSON(t, router, http.MethodPost, "/api/v1/onboarding", models.OnboardingRequest{
		Age: 25, Sex: "male", HeightCm: 175, WeightKg: 70,
		ActivityLevel: "active", Goal: "bulk", TrainingLocation: "home",
	}, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("onboarding: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/programs/latest", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("get latest program: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var progResp struct {
		Data models.Program `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &progResp); err != nil {
		t.Fatalf("decoding program response: %v", err)
	}
	if len(progResp.Data.Days) != 3 {
		t.Fatalf("expected 3 program days, got %d", len(progResp.Data.Days))
	}
	for _, day := range progResp.Data.Days {
		if len(day.Exercises) == 0 {
			t.Fatalf("day %q has no exercises", day.Name)
		}
		for _, ex := range day.Exercises {
			if ex.Exercise == nil || ex.Exercise.Location == "gym" {
				t.Fatalf("home program included a gym-only exercise: %+v", ex)
			}
			if ex.TargetSets != 4 || ex.TargetRepsMin != 6 || ex.TargetRepsMax != 10 {
				t.Fatalf("expected bulk's 4x6-10 scheme, got %d sets %d-%d reps", ex.TargetSets, ex.TargetRepsMin, ex.TargetRepsMax)
			}
		}
	}

	// Manual regeneration must also work on demand.
	rec = doJSON(t, router, http.MethodPost, "/api/v1/programs/generate", nil, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("regenerate program: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestFriendRequestAndSharedGoalFlow exercises the whole social feature end
// to end: request by email, accept, view the friend's shared profile
// (plan/targets, check-ins, workout counts), create a shared goal with
// progress computed for both sides, complete it — and confirms an outsider
// can access none of it.
func TestFriendRequestAndSharedGoalFlow(t *testing.T) {
	router := setupTestRouter(t)

	aliceToken := registerUser(t, router, "alice@coachia.local")
	bobToken := registerUser(t, router, "bob@coachia.local")
	outsiderToken := registerUser(t, router, "outsider@coachia.local")

	for _, tc := range []struct {
		token string
		req   models.OnboardingRequest
	}{
		{aliceToken, models.OnboardingRequest{Age: 29, Sex: "female", HeightCm: 165, WeightKg: 60, ActivityLevel: "moderate", Goal: "cut", TrainingLocation: "home"}},
		{bobToken, models.OnboardingRequest{Age: 31, Sex: "male", HeightCm: 178, WeightKg: 80, ActivityLevel: "moderate", Goal: "bulk", TrainingLocation: "gym"}},
	} {
		rec := doJSON(t, router, http.MethodPost, "/api/v1/onboarding", tc.req, tc.token)
		if rec.Code != http.StatusCreated {
			t.Fatalf("onboarding: expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	}

	// Self-request is rejected outright.
	rec := doJSON(t, router, http.MethodPost, "/api/v1/friends/requests", models.FriendRequest{Email: "alice@coachia.local"}, aliceToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("self friend request: expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	// Request to a nonexistent email is a clean 404, not a 500.
	rec = doJSON(t, router, http.MethodPost, "/api/v1/friends/requests", models.FriendRequest{Email: "nobody@coachia.local"}, aliceToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("request to unknown email: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}

	// Alice sends Bob a request.
	rec = doJSON(t, router, http.MethodPost, "/api/v1/friends/requests", models.FriendRequest{Email: "bob@coachia.local"}, aliceToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send friend request: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var reqResp struct {
		Data models.Friendship `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &reqResp); err != nil {
		t.Fatalf("decoding friend request response: %v", err)
	}
	friendshipID := reqResp.Data.ID

	// A duplicate request while pending is rejected.
	rec = doJSON(t, router, http.MethodPost, "/api/v1/friends/requests", models.FriendRequest{Email: "bob@coachia.local"}, aliceToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate pending request: expected 409, got %d: %s", rec.Code, rec.Body.String())
	}

	// Bob sees it as incoming.
	rec = doJSON(t, router, http.MethodGet, "/api/v1/friends/requests", nil, bobToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("list incoming: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var incomingResp struct {
		Data []models.Friendship `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &incomingResp); err != nil {
		t.Fatalf("decoding incoming requests: %v", err)
	}
	if len(incomingResp.Data) != 1 || incomingResp.Data[0].FriendEmail != "alice@coachia.local" {
		t.Fatalf("expected 1 incoming request from alice, got %+v", incomingResp.Data)
	}

	// Alice can't accept her own outgoing request — only the recipient can.
	rec = doJSON(t, router, http.MethodPost, fmt.Sprintf("/api/v1/friends/requests/%d/accept", friendshipID), nil, aliceToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("requester accepting own request: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}

	// Bob accepts.
	rec = doJSON(t, router, http.MethodPost, fmt.Sprintf("/api/v1/friends/requests/%d/accept", friendshipID), nil, bobToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("accept: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Both now see each other in their friends list.
	rec = doJSON(t, router, http.MethodGet, "/api/v1/friends", nil, aliceToken)
	var aliceFriends struct {
		Data []models.Friendship `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &aliceFriends); err != nil {
		t.Fatalf("decoding alice's friends: %v", err)
	}
	if len(aliceFriends.Data) != 1 || aliceFriends.Data[0].FriendEmail != "bob@coachia.local" {
		t.Fatalf("expected alice to see bob as a friend, got %+v", aliceFriends.Data)
	}

	// Alice views Bob's shared profile.
	rec = doJSON(t, router, http.MethodGet, fmt.Sprintf("/api/v1/friends/%d/profile", friendshipID), nil, aliceToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("get friend profile: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var friendProfileResp struct {
		Data models.FriendProfile `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &friendProfileResp); err != nil {
		t.Fatalf("decoding friend profile: %v", err)
	}
	if friendProfileResp.Data.Email != "bob@coachia.local" || friendProfileResp.Data.Targets == nil {
		t.Fatalf("expected bob's profile with targets, got %+v", friendProfileResp.Data)
	}

	// An outsider gets 404, not a peek, for both the profile and the friendship's goals.
	rec = doJSON(t, router, http.MethodGet, fmt.Sprintf("/api/v1/friends/%d/profile", friendshipID), nil, outsiderToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("outsider get friend profile: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}

	// Create a shared goal of type workouts_per_week.
	target := 3.0
	rec = doJSON(t, router, http.MethodPost, fmt.Sprintf("/api/v1/friends/%d/goals", friendshipID), models.CreateSharedGoalRequest{
		GoalType: "workouts_per_week", Description: "Entrenar 3 veces esta semana los dos", TargetValue: &target,
	}, aliceToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create shared goal: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var goalResp struct {
		Data models.SharedGoal `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &goalResp); err != nil {
		t.Fatalf("decoding shared goal: %v", err)
	}
	goalID := goalResp.Data.ID

	// Outsider can't even list the goals.
	rec = doJSON(t, router, http.MethodGet, fmt.Sprintf("/api/v1/friends/%d/goals", friendshipID), nil, outsiderToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("outsider list goals: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}

	// Bob lists the goals and sees progress computed for both sides.
	rec = doJSON(t, router, http.MethodGet, fmt.Sprintf("/api/v1/friends/%d/goals", friendshipID), nil, bobToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("list goals: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var listResp struct {
		Data []models.SharedGoal `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decoding goals list: %v", err)
	}
	if len(listResp.Data) != 1 || len(listResp.Data[0].Progress) != 2 {
		t.Fatalf("expected 1 goal with progress for both participants, got %+v", listResp.Data)
	}

	// Either side can mark the shared goal complete.
	rec = doJSON(t, router, http.MethodPost, fmt.Sprintf("/api/v1/goals/%d/complete", goalID), nil, bobToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("complete goal: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var completedResp struct {
		Data models.SharedGoal `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &completedResp); err != nil {
		t.Fatalf("decoding completed goal: %v", err)
	}
	if completedResp.Data.Status != "completed" {
		t.Fatalf("expected status completed, got %q", completedResp.Data.Status)
	}

	// Reject flow, on a fresh pair: outsider requests alice, alice rejects.
	rec = doJSON(t, router, http.MethodPost, "/api/v1/friends/requests", models.FriendRequest{Email: "alice@coachia.local"}, outsiderToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("outsider request alice: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &reqResp); err != nil {
		t.Fatalf("decoding outsider's request: %v", err)
	}
	rec = doJSON(t, router, http.MethodPost, fmt.Sprintf("/api/v1/friends/requests/%d/reject", reqResp.Data.ID), nil, aliceToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("reject: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	// A second resolution attempt on the same request must be rejected.
	rec = doJSON(t, router, http.MethodPost, fmt.Sprintf("/api/v1/friends/requests/%d/reject", reqResp.Data.ID), nil, aliceToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("re-reject: expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestFoodLogFlow confirms logging real food against the day's target works,
// that totals are summed correctly from the entries (never a separately
// stored number), and that one user can never delete another's entry.
func TestFoodLogFlow(t *testing.T) {
	router := setupTestRouter(t)
	token := registerUser(t, router, "logger@coachia.local")

	rec := doJSON(t, router, http.MethodPost, "/api/v1/onboarding", models.OnboardingRequest{
		Age: 28, Sex: "male", HeightCm: 178, WeightKg: 75,
		ActivityLevel: "moderate", Goal: "maintain", TrainingLocation: "gym",
	}, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("onboarding: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/foods", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list foods: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var foodsResp struct {
		Data []models.PeruvianFood `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &foodsResp); err != nil {
		t.Fatalf("decoding foods: %v", err)
	}
	if len(foodsResp.Data) == 0 {
		t.Fatal("expected a non-empty food catalog")
	}
	food := foodsResp.Data[0]

	rec = doJSON(t, router, http.MethodPost, "/api/v1/food-log", models.CreateFoodLogEntryRequest{
		FoodID: food.ID, MealType: "desayuno", Servings: 2,
	}, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("log food: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var entryResp struct {
		Data models.FoodLogEntry `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &entryResp); err != nil {
		t.Fatalf("decoding food log entry: %v", err)
	}
	wantCalories := int(food.Calories * 2)
	if entryResp.Data.Calories != wantCalories {
		t.Fatalf("expected %d kcal for 2 servings, got %d", wantCalories, entryResp.Data.Calories)
	}
	wantGrams := int(food.ServingGrams * 2)
	if entryResp.Data.TotalGrams != wantGrams {
		t.Fatalf("expected %dg for 2 servings of a %gg serving, got %dg", wantGrams, food.ServingGrams, entryResp.Data.TotalGrams)
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/food-log", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("get day: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var dayResp struct {
		Data models.DailyFoodLog `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &dayResp); err != nil {
		t.Fatalf("decoding daily food log: %v", err)
	}
	if len(dayResp.Data.Entries) != 1 || dayResp.Data.Consumed.Calories != wantCalories {
		t.Fatalf("expected 1 entry totaling %d kcal, got %+v", wantCalories, dayResp.Data)
	}
	if dayResp.Data.Target == nil || dayResp.Data.Target.Calories <= 0 {
		t.Fatalf("expected the day's target to be included, got %+v", dayResp.Data.Target)
	}

	// A second user can't delete the first user's entry.
	otherToken := registerUser(t, router, "other-logger@coachia.local")
	rec = doJSON(t, router, http.MethodDelete, fmt.Sprintf("/api/v1/food-log/%d", entryResp.Data.ID), nil, otherToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("other user delete entry: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}

	// The owner can.
	rec = doJSON(t, router, http.MethodDelete, fmt.Sprintf("/api/v1/food-log/%d", entryResp.Data.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete entry: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/food-log", nil, token)
	if err := json.Unmarshal(rec.Body.Bytes(), &dayResp); err != nil {
		t.Fatalf("decoding daily food log after delete: %v", err)
	}
	if len(dayResp.Data.Entries) != 0 || dayResp.Data.Consumed.Calories != 0 {
		t.Fatalf("expected an empty day after delete, got %+v", dayResp.Data)
	}
}
