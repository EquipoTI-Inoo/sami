import { Check, ChevronLeft, Flag, Plus, SkipForward, Timer, X } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  addWorkoutExercise,
  apiErrorMessage,
  completeWorkout,
  getExercises,
  getWorkout,
  logWorkoutSet,
  type Exercise,
  type ExerciseLocation,
  type Workout,
  type WorkoutExercise,
} from '../services/api'

const REST_SECONDS = 90

const MUSCLE_GROUP_LABELS: Record<string, string> = {
  piernas: 'Piernas',
  pecho: 'Pecho',
  espalda: 'Espalda',
  hombros: 'Hombros',
  brazos: 'Brazos',
  core: 'Core',
  cardio: 'Cardio',
}

export default function GymMode() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const workoutId = Number(id)

  const [workout, setWorkout] = useState<Workout | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [pickerOpen, setPickerOpen] = useState(false)
  const [restSeconds, setRestSeconds] = useState<number | null>(null)
  const [finishing, setFinishing] = useState(false)

  useEffect(() => {
    if (!workoutId) return
    getWorkout(workoutId)
      .then(setWorkout)
      .catch(() => setError('No se pudo cargar el entrenamiento.'))
      .finally(() => setLoading(false))
  }, [workoutId])

  useEffect(() => {
    if (restSeconds === null || restSeconds <= 0) return
    const timer = setTimeout(() => setRestSeconds((s) => (s === null ? null : s - 1)), 1000)
    return () => clearTimeout(timer)
  }, [restSeconds])

  async function refreshWorkout() {
    const fresh = await getWorkout(workoutId)
    setWorkout(fresh)
  }

  async function handleAddExercise(exerciseId: number) {
    try {
      await addWorkoutExercise(workoutId, exerciseId)
      setPickerOpen(false)
      await refreshWorkout()
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo agregar el ejercicio.'))
    }
  }

  async function handleLogSet(workoutExerciseId: number, reps: number, weightKg: number | undefined) {
    try {
      await logWorkoutSet(workoutId, workoutExerciseId, reps, weightKg)
      await refreshWorkout()
      setRestSeconds(REST_SECONDS)
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo registrar el set.'))
    }
  }

  async function handleFinish() {
    setFinishing(true)
    try {
      await completeWorkout(workoutId)
      navigate('/workouts')
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo terminar el entrenamiento.'))
      setFinishing(false)
    }
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-cream">
        <p className="text-sm text-stone-400">Cargando entrenamiento…</p>
      </div>
    )
  }

  if (!workout) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-cream px-4 text-center">
        <p className="text-sm text-stone-500">{error || 'No se encontró el entrenamiento.'}</p>
        <button onClick={() => navigate('/workouts')} className="btn-secondary">
          Volver a entrenar
        </button>
      </div>
    )
  }

  const isDone = Boolean(workout.completed_at)

  return (
    <div className="min-h-screen bg-cream pb-32">
      <header className="sticky top-0 z-20 flex items-center justify-between border-b border-stone-200 bg-white/95 px-4 py-3 backdrop-blur-sm">
        <button onClick={() => navigate('/workouts')} className="flex cursor-pointer items-center gap-1 text-sm font-medium text-stone-500 hover:text-stone-700">
          <ChevronLeft className="h-4 w-4" /> Entrenar
        </button>
        {restSeconds !== null && restSeconds > 0 && (
          <div className="flex items-center gap-2 rounded-full bg-yellow-50 px-3 py-1.5">
            <Timer className="h-4 w-4 text-yellow-800" />
            <span className="font-display text-sm font-bold text-yellow-800">
              {Math.floor(restSeconds / 60)}:{String(restSeconds % 60).padStart(2, '0')}
            </span>
            <button onClick={() => setRestSeconds(null)} aria-label="Saltar descanso" className="cursor-pointer text-yellow-600 hover:text-yellow-800">
              <SkipForward className="h-3.5 w-3.5" />
            </button>
          </div>
        )}
      </header>

      <div className="px-4 py-4">
        {error && <p className="mb-3 text-sm text-red-600">{error}</p>}

        {(workout.exercises ?? []).length === 0 && (
          <p className="mb-3 text-sm text-stone-400">Agrega tu primer ejercicio para empezar.</p>
        )}

        <div className="space-y-3">
          {(workout.exercises ?? []).map((we) => (
            <ExerciseCard key={we.id} workoutExercise={we} disabled={isDone} onLogSet={handleLogSet} />
          ))}
        </div>

        {!isDone && (
          <button
            onClick={() => setPickerOpen(true)}
            className="btn-secondary mt-4 w-full border-dashed py-3"
          >
            <Plus className="mr-1.5 h-4 w-4" /> Agregar ejercicio
          </button>
        )}
      </div>

      {!isDone && (
        <div className="fixed inset-x-0 bottom-0 z-20 border-t border-stone-200 bg-white p-4">
          <button onClick={handleFinish} disabled={finishing} className="btn-primary w-full bg-green-600 py-3.5 shadow-green-600/30 hover:bg-green-700">
            <Flag className="mr-1.5 h-4 w-4" /> {finishing ? 'Terminando…' : 'Terminar entrenamiento'}
          </button>
        </div>
      )}

      {pickerOpen && <ExercisePicker onPick={handleAddExercise} onClose={() => setPickerOpen(false)} />}
    </div>
  )
}

function ExerciseCard({
  workoutExercise,
  disabled,
  onLogSet,
}: {
  workoutExercise: WorkoutExercise
  disabled: boolean
  onLogSet: (workoutExerciseId: number, reps: number, weightKg: number | undefined) => void
}) {
  const [reps, setReps] = useState('')
  const [weight, setWeight] = useState('')

  function submit() {
    const repsNum = Number(reps)
    if (!repsNum || repsNum <= 0) return
    onLogSet(workoutExercise.id, repsNum, weight ? Number(weight) : undefined)
    setReps('')
    setWeight('')
  }

  return (
    <div className="card">
      <h3 className="text-sm font-semibold text-stone-900">{workoutExercise.exercise?.name ?? 'Ejercicio'}</h3>
      <p className="text-xs text-stone-400">{MUSCLE_GROUP_LABELS[workoutExercise.exercise?.muscle_group ?? ''] ?? ''}</p>

      {workoutExercise.sets.length > 0 && (
        <div className="mt-2 space-y-1">
          {workoutExercise.sets.map((set) => (
            <div key={set.id} className="flex items-center justify-between rounded-lg bg-stone-50 px-3 py-1.5 text-sm">
              <span className="font-medium text-stone-600">Set {set.set_number}</span>
              <span className="text-stone-700">
                {set.reps} reps{set.weight_kg ? ` · ${set.weight_kg} kg` : ''}
              </span>
            </div>
          ))}
        </div>
      )}

      {!disabled && (
        <div className="mt-3 flex items-end gap-2">
          <div className="flex-1 space-y-1">
            <label className="text-xs font-medium text-stone-500">Reps</label>
            <input type="number" min={1} value={reps} onChange={(e) => setReps(e.target.value)} className="input" />
          </div>
          <div className="flex-1 space-y-1">
            <label className="text-xs font-medium text-stone-500">Peso (kg)</label>
            <input type="number" min={0} step="0.5" value={weight} onChange={(e) => setWeight(e.target.value)} className="input" />
          </div>
          <button onClick={submit} aria-label="Registrar set" className="btn-primary h-[42px] w-[42px] shrink-0 px-0">
            <Check className="h-4 w-4" />
          </button>
        </div>
      )}
    </div>
  )
}

function ExercisePicker({ onPick, onClose }: { onPick: (exerciseId: number) => void; onClose: () => void }) {
  const [catalog, setCatalog] = useState<Exercise[]>([])
  const [loading, setLoading] = useState(true)
  const [location, setLocation] = useState<ExerciseLocation | ''>('')
  const dialogRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    setLoading(true)
    getExercises(location === 'gym' || location === 'home' ? location : undefined)
      .then(setCatalog)
      .finally(() => setLoading(false))
  }, [location])

  const grouped = catalog.reduce<Record<string, Exercise[]>>((acc, ex) => {
    ;(acc[ex.muscle_group] ??= []).push(ex)
    return acc
  }, {})

  return (
    <div className="fixed inset-0 z-30 flex items-end bg-stone-900/40" onClick={onClose}>
      <div
        ref={dialogRef}
        onClick={(e) => e.stopPropagation()}
        className="max-h-[80vh] w-full overflow-y-auto rounded-t-3xl bg-white p-4"
      >
        <div className="mb-3 flex items-center justify-between">
          <h2 className="font-display text-xl font-bold text-stone-900">Elige un ejercicio</h2>
          <button onClick={onClose} aria-label="Cerrar" className="cursor-pointer rounded-full p-1.5 text-stone-400 hover:bg-stone-100">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="mb-3 flex gap-2">
          {(['', 'home', 'gym'] as const).map((loc) => (
            <button
              key={loc}
              onClick={() => setLocation(loc)}
              className={`cursor-pointer rounded-full px-3 py-1.5 text-xs font-semibold transition-colors duration-200 ${
                location === loc ? 'bg-yellow-700 text-white' : 'bg-stone-100 text-stone-600 hover:bg-stone-200'
              }`}
            >
              {loc === '' ? 'Todo' : loc === 'home' ? 'Casa' : 'Gym'}
            </button>
          ))}
        </div>

        {loading && <p className="text-sm text-stone-400">Cargando…</p>}

        <div className="space-y-4">
          {Object.entries(grouped).map(([group, exercises]) => (
            <div key={group}>
              <h3 className="mb-1.5 text-xs font-bold uppercase tracking-wide text-stone-400">
                {MUSCLE_GROUP_LABELS[group] ?? group}
              </h3>
              <div className="space-y-1">
                {exercises.map((ex) => (
                  <button
                    key={ex.id}
                    onClick={() => onPick(ex.id)}
                    className="flex w-full cursor-pointer items-center justify-between rounded-xl px-3 py-2.5 text-left text-sm text-stone-800 hover:bg-yellow-50"
                  >
                    {ex.name}
                    <Plus className="h-4 w-4 text-yellow-700" />
                  </button>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
