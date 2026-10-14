import axios from 'axios'

export interface User {
  id: number
  email: string
  created_at: string
}

export interface AuthResponse {
  token: string
  user: User
}

export type ActivityLevel = 'sedentary' | 'light' | 'moderate' | 'active' | 'very_active'
export type Goal = 'cut' | 'maintain' | 'bulk' | 'performance'
export type TrainingLocation = 'gym' | 'home' | 'both'

export interface OnboardingRequest {
  age: number
  sex: 'male' | 'female'
  height_cm: number
  weight_kg: number
  body_fat_pct?: number
  activity_level: ActivityLevel
  goal: Goal
  training_location: TrainingLocation
  equipment: string[]
  likes: string[]
  dislikes: string[]
  restrictions: string[]
}

export interface UserProfile extends OnboardingRequest {
  user_id: number
  created_at: string
  updated_at: string
}

export interface NutritionTargets {
  id: number
  user_id: number
  calories: number
  protein_g: number
  carbs_g: number
  fat_g: number
  updated_by: 'user' | 'ai'
  reason: string
  created_at: string
}

export interface OnboardingResponse {
  profile: UserProfile
  targets: NutritionTargets
}

export interface PeruvianFood {
  id: number
  name: string
  category: string
  serving_desc: string
  serving_grams: number
  calories: number
  protein_g: number
  carbs_g: number
  fat_g: number
  tags: string[]
}

export interface MealItem {
  food: PeruvianFood
  servings: number
  total_grams: number
  calories: number
  protein_g: number
  carbs_g: number
  fat_g: number
}

export interface Meal {
  name: string
  items: MealItem[]
  calories: number
  protein_g: number
  carbs_g: number
  fat_g: number
}

export interface MealPlan {
  meals: Meal[]
  calories: number
  protein_g: number
  carbs_g: number
  fat_g: number
  target: NutritionTargets
}

export interface BodyMeasurement {
  id: number
  user_id: number
  logged_on: string
  weight_kg: number
  waist_cm?: number | null
  hip_cm?: number | null
  arm_cm?: number | null
  body_fat_pct?: number | null
  note?: string | null
  created_at: string
}

export interface CheckinRequest {
  weight_kg: number
  waist_cm?: number
  hip_cm?: number
  arm_cm?: number
  body_fat_pct?: number
  note?: string
}

export interface CheckinStatus {
  needs_checkin: boolean
  last_logged_on?: string
  days_since_last?: number
}

export interface ChatHistoryMessage {
  role: 'user' | 'assistant'
  text: string
}

export interface NutritionTargetChangePayload {
  calories: number
  protein_g: number
  carbs_g: number
  fat_g: number
}

export interface PendingChange {
  id: number
  user_id: number
  change_type: 'nutrition_target'
  payload: NutritionTargetChangePayload
  reason: string
  status: 'pending' | 'approved' | 'rejected'
  created_at: string
  resolved_at?: string | null
}

export interface ChatResponse {
  reply: string
  onboarding_completed: boolean
  targets?: NutritionTargets
  pending_change?: PendingChange
}

const REQUEST_TIMEOUT = 10_000

export const api = axios.create({
  baseURL: '/api/v1',
  timeout: REQUEST_TIMEOUT,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export async function register(email: string, password: string): Promise<AuthResponse> {
  const res = await api.post<{ data: AuthResponse }>('/auth/register', { email, password })
  return res.data.data
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  const res = await api.post<{ data: AuthResponse }>('/auth/login', { email, password })
  return res.data.data
}

export async function me(): Promise<User> {
  const res = await api.get<{ data: User }>('/me')
  return res.data.data
}

export async function submitOnboarding(req: OnboardingRequest): Promise<OnboardingResponse> {
  const res = await api.post<{ data: OnboardingResponse }>('/onboarding', req)
  return res.data.data
}

export async function getProfile(): Promise<UserProfile | null> {
  try {
    const res = await api.get<{ data: UserProfile }>('/profile')
    return res.data.data
  } catch (err) {
    if (axios.isAxiosError(err) && err.response?.status === 404) return null
    throw err
  }
}

export async function getNutritionTargets(): Promise<NutritionTargets | null> {
  try {
    const res = await api.get<{ data: NutritionTargets }>('/nutrition/targets')
    return res.data.data
  } catch (err) {
    if (axios.isAxiosError(err) && err.response?.status === 404) return null
    throw err
  }
}

const CHAT_REQUEST_TIMEOUT = 30_000 // the coach may call a tool + make a second round-trip to Claude

export async function sendChatMessage(message: string): Promise<ChatResponse> {
  const res = await api.post<{ data: ChatResponse }>('/chat', { message }, { timeout: CHAT_REQUEST_TIMEOUT })
  return res.data.data
}

export async function getChatHistory(): Promise<ChatHistoryMessage[]> {
  const res = await api.get<{ data: ChatHistoryMessage[] }>('/chat/history')
  return res.data.data
}

export async function getPendingChanges(): Promise<PendingChange[]> {
  const res = await api.get<{ data: PendingChange[] }>('/pending-changes')
  return res.data.data
}

export async function approvePendingChange(id: number): Promise<PendingChange> {
  const res = await api.post<{ data: PendingChange }>(`/pending-changes/${id}/approve`)
  return res.data.data
}

export async function rejectPendingChange(id: number): Promise<PendingChange> {
  const res = await api.post<{ data: PendingChange }>(`/pending-changes/${id}/reject`)
  return res.data.data
}

export async function getMealPlan(): Promise<MealPlan | null> {
  try {
    const res = await api.get<{ data: MealPlan }>('/nutrition/meal-plan')
    return res.data.data
  } catch (err) {
    if (axios.isAxiosError(err) && err.response?.status === 404) return null
    throw err
  }
}

export async function getCheckinStatus(): Promise<CheckinStatus> {
  const res = await api.get<{ data: CheckinStatus }>('/checkin/status')
  return res.data.data
}

export async function submitCheckin(req: CheckinRequest): Promise<BodyMeasurement> {
  const res = await api.post<{ data: BodyMeasurement }>('/checkin', req)
  return res.data.data
}

export async function getTargetsHistory(): Promise<NutritionTargets[]> {
  const res = await api.get<{ data: NutritionTargets[] }>('/nutrition/targets/history')
  return res.data.data
}

export type MuscleGroup = 'piernas' | 'pecho' | 'espalda' | 'hombros' | 'brazos' | 'core' | 'cardio'
export type ExerciseLocation = 'gym' | 'home' | 'both'

export interface Exercise {
  id: number
  name: string
  muscle_group: MuscleGroup
  equipment: string[]
  location: ExerciseLocation
}

export interface WorkoutSet {
  id: number
  workout_exercise_id: number
  set_number: number
  reps: number
  weight_kg: number | null
  completed_at: string
}

export interface WorkoutExercise {
  id: number
  workout_id: number
  exercise_id: number
  exercise?: Exercise
  order_index: number
  sets: WorkoutSet[]
}

export interface Workout {
  id: number
  user_id: number
  started_at: string
  completed_at: string | null
  notes: string | null
  exercise_count: number
  exercises?: WorkoutExercise[]
}

export async function getExercises(location?: 'gym' | 'home'): Promise<Exercise[]> {
  const res = await api.get<{ data: Exercise[] }>('/exercises', { params: location ? { location } : undefined })
  return res.data.data
}

export async function startWorkout(): Promise<Workout> {
  const res = await api.post<{ data: Workout }>('/workouts')
  return res.data.data
}

export async function listWorkouts(): Promise<Workout[]> {
  const res = await api.get<{ data: Workout[] }>('/workouts')
  return res.data.data
}

export async function getWorkout(id: number): Promise<Workout> {
  const res = await api.get<{ data: Workout }>(`/workouts/${id}`)
  return res.data.data
}

export async function addWorkoutExercise(workoutId: number, exerciseId: number): Promise<WorkoutExercise> {
  const res = await api.post<{ data: WorkoutExercise }>(`/workouts/${workoutId}/exercises`, { exercise_id: exerciseId })
  return res.data.data
}

export async function logWorkoutSet(workoutId: number, workoutExerciseId: number, reps: number, weightKg?: number): Promise<WorkoutSet> {
  const res = await api.post<{ data: WorkoutSet }>(`/workouts/${workoutId}/exercises/${workoutExerciseId}/sets`, {
    reps,
    weight_kg: weightKg,
  })
  return res.data.data
}

export async function completeWorkout(id: number): Promise<Workout> {
  const res = await api.post<{ data: Workout }>(`/workouts/${id}/complete`)
  return res.data.data
}

export interface ProgramExercise {
  id: number
  exercise_id: number
  exercise?: Exercise
  order_index: number
  target_sets: number
  target_reps_min: number
  target_reps_max: number
}

export interface ProgramDay {
  id: number
  day_index: number
  name: string
  exercises: ProgramExercise[]
}

export interface Program {
  id: number
  user_id: number
  goal: Goal
  location: ExerciseLocation
  created_at: string
  days: ProgramDay[]
}

export async function getLatestProgram(): Promise<Program | null> {
  try {
    const res = await api.get<{ data: Program }>('/programs/latest')
    return res.data.data
  } catch (err) {
    if (axios.isAxiosError(err) && err.response?.status === 404) return null
    throw err
  }
}

export async function generateProgram(): Promise<Program> {
  const res = await api.post<{ data: Program }>('/programs/generate')
  return res.data.data
}

export interface Friendship {
  id: number
  requester_id: number
  recipient_id: number
  status: 'pending' | 'accepted' | 'rejected'
  created_at: string
  resolved_at: string | null
  friend_id: number
  friend_email: string
}

export interface FriendProfile {
  email: string
  profile?: UserProfile
  targets?: NutritionTargets
  recent_checkins: BodyMeasurement[]
  workouts_this_week: number
  total_workouts: number
}

export type GoalType = 'workouts_per_week' | 'weight_target' | 'custom'

export interface GoalProgress {
  user_id: number
  email: string
  workouts_this_week?: number
  current_weight_kg?: number
}

export interface SharedGoal {
  id: number
  friendship_id: number
  created_by: number
  goal_type: GoalType
  description: string
  target_value: number | null
  target_date: string | null
  status: 'active' | 'completed' | 'abandoned'
  created_at: string
  completed_at: string | null
  progress?: Record<string, GoalProgress>
}

export interface CreateSharedGoalRequest {
  goal_type: GoalType
  description: string
  target_value?: number
  target_date?: string
}

export async function sendFriendRequest(email: string): Promise<Friendship> {
  const res = await api.post<{ data: Friendship }>('/friends/requests', { email })
  return res.data.data
}

export async function getIncomingFriendRequests(): Promise<Friendship[]> {
  const res = await api.get<{ data: Friendship[] }>('/friends/requests')
  return res.data.data
}

export async function acceptFriendRequest(id: number): Promise<Friendship> {
  const res = await api.post<{ data: Friendship }>(`/friends/requests/${id}/accept`)
  return res.data.data
}

export async function rejectFriendRequest(id: number): Promise<Friendship> {
  const res = await api.post<{ data: Friendship }>(`/friends/requests/${id}/reject`)
  return res.data.data
}

export async function listFriends(): Promise<Friendship[]> {
  const res = await api.get<{ data: Friendship[] }>('/friends')
  return res.data.data
}

export async function getFriendProfile(friendshipId: number): Promise<FriendProfile> {
  const res = await api.get<{ data: FriendProfile }>(`/friends/${friendshipId}/profile`)
  return res.data.data
}

export async function createSharedGoal(friendshipId: number, req: CreateSharedGoalRequest): Promise<SharedGoal> {
  const res = await api.post<{ data: SharedGoal }>(`/friends/${friendshipId}/goals`, req)
  return res.data.data
}

export async function listSharedGoals(friendshipId: number): Promise<SharedGoal[]> {
  const res = await api.get<{ data: SharedGoal[] }>(`/friends/${friendshipId}/goals`)
  return res.data.data
}

export async function completeSharedGoal(goalId: number): Promise<SharedGoal> {
  const res = await api.post<{ data: SharedGoal }>(`/goals/${goalId}/complete`)
  return res.data.data
}

export type MealType = 'desayuno' | 'almuerzo' | 'cena' | 'snack'

export interface FoodLogEntry {
  id: number
  user_id: number
  food_id: number
  food?: PeruvianFood
  meal_type: MealType
  servings: number
  total_grams: number
  logged_on: string
  created_at: string
  calories: number
  protein_g: number
  carbs_g: number
  fat_g: number
}

export interface CreateFoodLogEntryRequest {
  food_id: number
  meal_type: MealType
  servings: number
  logged_on?: string
}

export interface MacroTotals {
  calories: number
  protein_g: number
  carbs_g: number
  fat_g: number
}

export interface DailyFoodLog {
  logged_on: string
  entries: FoodLogEntry[]
  consumed: MacroTotals
  target?: NutritionTargets
}

export async function getFoods(): Promise<PeruvianFood[]> {
  const res = await api.get<{ data: PeruvianFood[] }>('/foods')
  return res.data.data
}

export async function logFood(req: CreateFoodLogEntryRequest): Promise<FoodLogEntry> {
  const res = await api.post<{ data: FoodLogEntry }>('/food-log', req)
  return res.data.data
}

export async function getDailyFoodLog(date?: string): Promise<DailyFoodLog> {
  const res = await api.get<{ data: DailyFoodLog }>('/food-log', { params: date ? { date } : undefined })
  return res.data.data
}

export async function deleteFoodLogEntry(id: number): Promise<void> {
  await api.delete(`/food-log/${id}`)
}

export function apiErrorMessage(err: unknown, fallback: string): string {
  if (axios.isAxiosError(err)) {
    if (!err.response) {
      return 'No se pudo conectar con el servidor. Revisa tu conexión e inténtalo de nuevo.'
    }
    const data = err.response.data as { error?: string } | undefined
    if (data?.error) return data.error
  }
  return fallback
}
