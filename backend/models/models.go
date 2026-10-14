package models

import "encoding/json"

type User struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type OnboardingRequest struct {
	Age              int      `json:"age" binding:"required,min=13,max=100"`
	Sex              string   `json:"sex" binding:"required,oneof=male female"`
	HeightCm         float64  `json:"height_cm" binding:"required,min=100,max=250"`
	WeightKg         float64  `json:"weight_kg" binding:"required,min=30,max=300"`
	BodyFatPct       *float64 `json:"body_fat_pct" binding:"omitempty,min=3,max=70"`
	ActivityLevel    string   `json:"activity_level" binding:"required,oneof=sedentary light moderate active very_active"`
	Goal             string   `json:"goal" binding:"required,oneof=cut maintain bulk performance"`
	TrainingLocation string   `json:"training_location" binding:"required,oneof=gym home both"`
	Equipment        []string `json:"equipment"`
	Likes            []string `json:"likes"`
	Dislikes         []string `json:"dislikes"`
	Restrictions     []string `json:"restrictions"`
}

type UserProfile struct {
	UserID           int64    `json:"user_id"`
	Age              int      `json:"age"`
	Sex              string   `json:"sex"`
	HeightCm         float64  `json:"height_cm"`
	BodyFatPct       *float64 `json:"body_fat_pct"`
	ActivityLevel    string   `json:"activity_level"`
	Goal             string   `json:"goal"`
	TrainingLocation string   `json:"training_location"`
	Equipment        []string `json:"equipment"`
	Likes            []string `json:"likes"`
	Dislikes         []string `json:"dislikes"`
	Restrictions     []string `json:"restrictions"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type NutritionTargets struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	Calories  int    `json:"calories"`
	ProteinG  int    `json:"protein_g"`
	CarbsG    int    `json:"carbs_g"`
	FatG      int    `json:"fat_g"`
	UpdatedBy string `json:"updated_by"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
}

type BodyMeasurement struct {
	ID         int64    `json:"id"`
	UserID     int64    `json:"user_id"`
	LoggedOn   string   `json:"logged_on"`
	WeightKg   float64  `json:"weight_kg"`
	WaistCm    *float64 `json:"waist_cm"`
	HipCm      *float64 `json:"hip_cm"`
	ArmCm      *float64 `json:"arm_cm"`
	BodyFatPct *float64 `json:"body_fat_pct"`
	Note       *string  `json:"note"`
	CreatedAt  string   `json:"created_at"`
}

type CheckinRequest struct {
	WeightKg   float64  `json:"weight_kg" binding:"required,min=30,max=300"`
	WaistCm    *float64 `json:"waist_cm" binding:"omitempty,min=30,max=200"`
	HipCm      *float64 `json:"hip_cm" binding:"omitempty,min=30,max=200"`
	ArmCm      *float64 `json:"arm_cm" binding:"omitempty,min=10,max=100"`
	BodyFatPct *float64 `json:"body_fat_pct" binding:"omitempty,min=3,max=70"`
	Note       string   `json:"note"`
}

type Exercise struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	MuscleGroup string   `json:"muscle_group"`
	Equipment   []string `json:"equipment"`
	Location    string   `json:"location"`
}

type Workout struct {
	ID            int64             `json:"id"`
	UserID        int64             `json:"user_id"`
	StartedAt     string            `json:"started_at"`
	CompletedAt   *string           `json:"completed_at"`
	Notes         *string           `json:"notes"`
	ExerciseCount int               `json:"exercise_count"`
	Exercises     []WorkoutExercise `json:"exercises,omitempty"`
}

type WorkoutExercise struct {
	ID         int64        `json:"id"`
	WorkoutID  int64        `json:"workout_id"`
	ExerciseID int64        `json:"exercise_id"`
	Exercise   *Exercise    `json:"exercise,omitempty"`
	OrderIndex int          `json:"order_index"`
	Sets       []WorkoutSet `json:"sets,omitempty"`
}

type WorkoutSet struct {
	ID                int64    `json:"id"`
	WorkoutExerciseID int64    `json:"workout_exercise_id"`
	SetNumber         int      `json:"set_number"`
	Reps              int      `json:"reps"`
	WeightKg          *float64 `json:"weight_kg"`
	CompletedAt       string   `json:"completed_at"`
}

type AddWorkoutExerciseRequest struct {
	ExerciseID int64 `json:"exercise_id" binding:"required"`
}

type LogSetRequest struct {
	Reps     int      `json:"reps" binding:"required,min=1,max=1000"`
	WeightKg *float64 `json:"weight_kg" binding:"omitempty,min=0,max=1000"`
}

type Program struct {
	ID        int64        `json:"id"`
	UserID    int64        `json:"user_id"`
	Goal      string       `json:"goal"`
	Location  string       `json:"location"`
	CreatedAt string       `json:"created_at"`
	Days      []ProgramDay `json:"days"`
}

type ProgramDay struct {
	ID        int64             `json:"id"`
	DayIndex  int               `json:"day_index"`
	Name      string            `json:"name"`
	Exercises []ProgramExercise `json:"exercises"`
}

type ProgramExercise struct {
	ID            int64     `json:"id"`
	ExerciseID    int64     `json:"exercise_id"`
	Exercise      *Exercise `json:"exercise,omitempty"`
	OrderIndex    int       `json:"order_index"`
	TargetSets    int       `json:"target_sets"`
	TargetRepsMin int       `json:"target_reps_min"`
	TargetRepsMax int       `json:"target_reps_max"`
}

type Friendship struct {
	ID          int64   `json:"id"`
	RequesterID int64   `json:"requester_id"`
	RecipientID int64   `json:"recipient_id"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	ResolvedAt  *string `json:"resolved_at"`
	// Populated relative to whoever is asking, so the frontend never has to
	// cross-reference its own user id against requester/recipient.
	FriendID    int64  `json:"friend_id"`
	FriendEmail string `json:"friend_email"`
}

type FriendRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type FriendProfile struct {
	Email            string            `json:"email"`
	Profile          *UserProfile      `json:"profile,omitempty"`
	Targets          *NutritionTargets `json:"targets,omitempty"`
	RecentCheckins   []BodyMeasurement `json:"recent_checkins"`
	WorkoutsThisWeek int               `json:"workouts_this_week"`
	TotalWorkouts    int               `json:"total_workouts"`
}

type SharedGoal struct {
	ID           int64    `json:"id"`
	FriendshipID int64    `json:"friendship_id"`
	CreatedBy    int64    `json:"created_by"`
	GoalType     string   `json:"goal_type"`
	Description  string   `json:"description"`
	TargetValue  *float64 `json:"target_value"`
	TargetDate   *string  `json:"target_date"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"created_at"`
	CompletedAt  *string  `json:"completed_at"`
	// Progress is computed at read time, never stored — see
	// SharedGoalController. Keyed by user id (as strings, for valid JSON keys).
	Progress map[string]GoalProgress `json:"progress,omitempty"`
}

type GoalProgress struct {
	UserID           int64    `json:"user_id"`
	Email            string   `json:"email"`
	WorkoutsThisWeek int      `json:"workouts_this_week,omitempty"`
	CurrentWeightKg  *float64 `json:"current_weight_kg,omitempty"`
}

type CreateSharedGoalRequest struct {
	GoalType    string   `json:"goal_type" binding:"required,oneof=workouts_per_week weight_target custom"`
	Description string   `json:"description" binding:"required"`
	TargetValue *float64 `json:"target_value"`
	TargetDate  *string  `json:"target_date"`
}

type PendingChange struct {
	ID         int64           `json:"id"`
	UserID     int64           `json:"user_id"`
	ChangeType string          `json:"change_type"`
	Payload    json.RawMessage `json:"payload"`
	Reason     string          `json:"reason"`
	Status     string          `json:"status"`
	CreatedAt  string          `json:"created_at"`
	ResolvedAt *string         `json:"resolved_at"`
}

type NutritionTargetChangePayload struct {
	Calories int `json:"calories"`
	ProteinG int `json:"protein_g"`
	CarbsG   int `json:"carbs_g"`
	FatG     int `json:"fat_g"`
}

type CheckinStatus struct {
	NeedsCheckin  bool   `json:"needs_checkin"`
	LastLoggedOn  string `json:"last_logged_on"`
	DaysSinceLast int    `json:"days_since_last"`
}

type OnboardingResponse struct {
	Profile UserProfile      `json:"profile"`
	Targets NutritionTargets `json:"targets"`
}

type PeruvianFood struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	ServingDesc  string   `json:"serving_desc"`
	ServingGrams float64  `json:"serving_grams"`
	Calories     float64  `json:"calories"`
	ProteinG     float64  `json:"protein_g"`
	CarbsG       float64  `json:"carbs_g"`
	FatG         float64  `json:"fat_g"`
	Tags         []string `json:"tags"`
}

type FoodLogEntry struct {
	ID         int64         `json:"id"`
	UserID     int64         `json:"user_id"`
	FoodID     int64         `json:"food_id"`
	Food       *PeruvianFood `json:"food,omitempty"`
	MealType   string        `json:"meal_type"`
	Servings   float64       `json:"servings"`
	TotalGrams int           `json:"total_grams"`
	LoggedOn   string        `json:"logged_on"`
	CreatedAt  string        `json:"created_at"`
	Calories   int           `json:"calories"`
	ProteinG   int           `json:"protein_g"`
	CarbsG     int           `json:"carbs_g"`
	FatG       int           `json:"fat_g"`
}

type CreateFoodLogEntryRequest struct {
	FoodID   int64   `json:"food_id" binding:"required"`
	MealType string  `json:"meal_type" binding:"required,oneof=desayuno almuerzo cena snack"`
	Servings float64 `json:"servings" binding:"required,gt=0,max=20"`
	LoggedOn string  `json:"logged_on"` // optional "2006-01-02", defaults to today (UTC)
}

type MacroTotals struct {
	Calories int `json:"calories"`
	ProteinG int `json:"protein_g"`
	CarbsG   int `json:"carbs_g"`
	FatG     int `json:"fat_g"`
}

type DailyFoodLog struct {
	LoggedOn string            `json:"logged_on"`
	Entries  []FoodLogEntry    `json:"entries"`
	Consumed MacroTotals       `json:"consumed"`
	Target   *NutritionTargets `json:"target,omitempty"`
}

type MealItem struct {
	Food       PeruvianFood `json:"food"`
	Servings   float64      `json:"servings"`
	TotalGrams int          `json:"total_grams"`
	Calories   int          `json:"calories"`
	ProteinG   int          `json:"protein_g"`
	CarbsG     int          `json:"carbs_g"`
	FatG       int          `json:"fat_g"`
}

type Meal struct {
	Name     string     `json:"name"`
	Items    []MealItem `json:"items"`
	Calories int        `json:"calories"`
	ProteinG int        `json:"protein_g"`
	CarbsG   int        `json:"carbs_g"`
	FatG     int        `json:"fat_g"`
}

type MealPlan struct {
	Meals    []Meal           `json:"meals"`
	Calories int              `json:"calories"`
	ProteinG int              `json:"protein_g"`
	CarbsG   int              `json:"carbs_g"`
	FatG     int              `json:"fat_g"`
	Target   NutritionTargets `json:"target"`
}
