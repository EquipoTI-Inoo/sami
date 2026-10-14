import { CalendarDays, Dumbbell, History, Play, RefreshCw, Sparkles } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import BottomNav from '../components/BottomNav'
import {
  addWorkoutExercise,
  apiErrorMessage,
  generateProgram,
  getLatestProgram,
  listWorkouts,
  startWorkout,
  type Program,
  type Workout,
} from '../services/api'

const MUSCLE_GROUP_LABELS: Record<string, string> = {
  piernas: 'Piernas',
  pecho: 'Pecho',
  espalda: 'Espalda',
  hombros: 'Hombros',
  brazos: 'Brazos',
  core: 'Core',
  cardio: 'Cardio',
}

export default function Workouts() {
  const navigate = useNavigate()
  const [history, setHistory] = useState<Workout[]>([])
  const [program, setProgram] = useState<Program | null>(null)
  const [activeDay, setActiveDay] = useState(0)
  const [loading, setLoading] = useState(true)
  const [starting, setStarting] = useState(false)
  const [regenerating, setRegenerating] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    Promise.all([listWorkouts(), getLatestProgram()])
      .then(([w, p]) => {
        setHistory(w)
        setProgram(p)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const inProgress = history.find((w) => !w.completed_at)
  const completed = history.filter((w) => w.completed_at)

  async function handleStart() {
    setError('')
    setStarting(true)
    try {
      const workout = await startWorkout()
      navigate(`/workouts/${workout.id}`)
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo iniciar el entrenamiento.'))
    } finally {
      setStarting(false)
    }
  }

  async function handleStartDay(dayIndex: number) {
    if (!program) return
    setError('')
    setStarting(true)
    try {
      const workout = await startWorkout()
      const day = program.days[dayIndex]
      for (const pe of day.exercises) {
        await addWorkoutExercise(workout.id, pe.exercise_id)
      }
      navigate(`/workouts/${workout.id}`)
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo iniciar el entrenamiento.'))
    } finally {
      setStarting(false)
    }
  }

  async function handleRegenerate() {
    setRegenerating(true)
    try {
      const fresh = await generateProgram()
      setProgram(fresh)
      setActiveDay(0)
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo generar tu programa.'))
    } finally {
      setRegenerating(false)
    }
  }

  return (
    <div className="min-h-screen bg-cream px-4 pb-28 pt-6">
      <header className="mb-5 flex items-center gap-2">
        <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-yellow-700 shadow-sm shadow-yellow-700/40">
          <Dumbbell className="h-5 w-5 text-white" strokeWidth={2.25} />
        </div>
        <h1 className="font-display text-2xl font-bold tracking-tight text-stone-900">Entrenar</h1>
      </header>

      {inProgress && (
        <div className="mb-4 overflow-hidden rounded-3xl bg-gradient-to-br from-yellow-700 to-yellow-800 p-5 text-white shadow-lg shadow-yellow-700/30">
          <p className="text-sm font-medium text-yellow-50">Tienes un entrenamiento en curso</p>
          <p className="font-display mt-1 text-xl font-bold">
            Empezado {new Date(inProgress.started_at).toLocaleTimeString('es-PE', { hour: 'numeric', minute: '2-digit' })}
          </p>
          <button
            onClick={() => navigate(`/workouts/${inProgress.id}`)}
            className="mt-4 flex w-full cursor-pointer items-center justify-center gap-2 rounded-xl bg-white py-3 text-sm font-semibold text-yellow-800 transition-colors duration-200 hover:bg-yellow-50"
          >
            <Play className="h-4 w-4" /> Continuar entrenamiento
          </button>
        </div>
      )}

      {error && <p className="mb-3 text-sm text-red-600">{error}</p>}

      {!loading && !program && (
        <div className="card text-center">
          <p className="text-sm text-stone-500">Todavía no tienes un programa generado.</p>
          <button onClick={handleRegenerate} disabled={regenerating} className="btn-primary mt-3 w-full py-3">
            <Sparkles className="mr-1.5 h-4 w-4" /> {regenerating ? 'Generando…' : 'Generar mi programa'}
          </button>
        </div>
      )}

      {program && (
        <div className="card">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="font-display text-lg font-bold text-stone-800">Tu programa</h2>
            <button
              onClick={handleRegenerate}
              disabled={regenerating}
              aria-label="Regenerar programa"
              className="cursor-pointer rounded-full p-1.5 text-stone-400 hover:bg-stone-100 hover:text-stone-600 disabled:opacity-50"
            >
              <RefreshCw className={`h-4 w-4 ${regenerating ? 'animate-spin' : ''}`} />
            </button>
          </div>

          <div className="mb-3 flex gap-2">
            {program.days.map((day, i) => (
              <button
                key={day.id}
                onClick={() => setActiveDay(i)}
                className={`flex-1 cursor-pointer rounded-xl py-2 text-xs font-semibold transition-colors duration-200 ${
                  activeDay === i ? 'bg-yellow-700 text-white' : 'bg-stone-100 text-stone-600 hover:bg-stone-200'
                }`}
              >
                Día {i + 1}
              </button>
            ))}
          </div>

          {program.days[activeDay] && (
            <>
              <p className="mb-2 text-xs font-medium text-stone-500">{program.days[activeDay].name}</p>
              <div className="space-y-1.5">
                {program.days[activeDay].exercises.map((pe) => (
                  <div key={pe.id} className="flex items-center justify-between rounded-xl bg-stone-50 px-3 py-2">
                    <div>
                      <p className="text-sm font-medium text-stone-800">{pe.exercise?.name}</p>
                      <p className="text-xs text-stone-400">{MUSCLE_GROUP_LABELS[pe.exercise?.muscle_group ?? ''] ?? ''}</p>
                    </div>
                    <span className="shrink-0 text-xs font-semibold text-yellow-800">
                      {pe.target_sets}x{pe.target_reps_min}-{pe.target_reps_max}
                    </span>
                  </div>
                ))}
              </div>
              <button onClick={() => handleStartDay(activeDay)} disabled={starting || Boolean(inProgress)} className="btn-primary mt-3 w-full py-3">
                <Play className="mr-1.5 h-4 w-4" /> {starting ? 'Iniciando…' : `Empezar día ${activeDay + 1}`}
              </button>
            </>
          )}
        </div>
      )}

      {!program && !loading && (
        <button onClick={handleStart} disabled={starting} className="btn-secondary mt-3 w-full py-3">
          <Play className="mr-1.5 h-4 w-4" /> {starting ? 'Iniciando…' : 'Empezar entrenamiento libre'}
        </button>
      )}

      <div className="mt-6">
        <div className="mb-2 flex items-center gap-1.5">
          <History className="h-4 w-4 text-stone-400" />
          <h2 className="font-display text-lg font-bold text-stone-800">Historial</h2>
        </div>

        {loading && <p className="text-sm text-stone-400">Cargando…</p>}
        {!loading && completed.length === 0 && <p className="text-sm text-stone-400">Todavía no has completado un entrenamiento.</p>}

        <div className="space-y-2">
          {completed.map((w) => (
            <button key={w.id} onClick={() => navigate(`/workouts/${w.id}`)} className="card flex w-full cursor-pointer items-center gap-3 text-left">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-stone-100">
                <CalendarDays className="h-5 w-5 text-stone-500" />
              </div>
              <div>
                <p className="text-sm font-medium text-stone-900">
                  {new Date(w.started_at).toLocaleDateString('es-PE', { day: 'numeric', month: 'short', year: 'numeric' })}
                </p>
                <p className="text-xs text-stone-500">{w.exercise_count} ejercicios</p>
              </div>
            </button>
          ))}
        </div>
      </div>

      <BottomNav />
    </div>
  )
}
