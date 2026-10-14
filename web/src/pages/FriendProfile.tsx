import { ArrowLeft, CalendarDays, Check, Dumbbell, Flag, Plus, Target, TrendingUp, User } from 'lucide-react'
import { useEffect, useState, type FormEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  apiErrorMessage,
  completeSharedGoal,
  createSharedGoal,
  getFriendProfile,
  listSharedGoals,
  type FriendProfile as FriendProfileData,
  type GoalType,
  type SharedGoal,
} from '../services/api'

const GOAL_TYPE_LABELS: Record<GoalType, string> = {
  workouts_per_week: 'Entrenamientos por semana',
  weight_target: 'Meta de peso',
  custom: 'Personalizada',
}

export default function FriendProfile() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const friendshipId = Number(id)

  const [profile, setProfile] = useState<FriendProfileData | null>(null)
  const [goals, setGoals] = useState<SharedGoal[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showNewGoal, setShowNewGoal] = useState(false)

  function refresh() {
    return Promise.all([getFriendProfile(friendshipId), listSharedGoals(friendshipId)]).then(([p, g]) => {
      setProfile(p)
      setGoals(g)
    })
  }

  useEffect(() => {
    if (!friendshipId) return
    refresh()
      .catch(() => setError('No se pudo cargar el perfil de tu amigo.'))
      .finally(() => setLoading(false))
  }, [friendshipId])

  async function handleComplete(goalId: number) {
    try {
      await completeSharedGoal(goalId)
      await refresh()
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo completar la meta.'))
    }
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-cream">
        <p className="text-sm text-stone-400">Cargando…</p>
      </div>
    )
  }

  if (!profile) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-cream px-4 text-center">
        <p className="text-sm text-stone-500">{error || 'No se encontró este perfil.'}</p>
        <button onClick={() => navigate('/friends')} className="btn-secondary">
          Volver a amigos
        </button>
      </div>
    )
  }

  const activeGoals = goals.filter((g) => g.status === 'active')
  const pastGoals = goals.filter((g) => g.status !== 'active')

  return (
    <div className="min-h-screen bg-cream px-4 pb-10 pt-6">
      <button onClick={() => navigate('/friends')} className="mb-4 flex items-center gap-1 text-sm font-medium text-stone-500 hover:text-stone-700">
        <ArrowLeft className="h-4 w-4" /> Amigos
      </button>

      <div className="mb-4 flex items-center gap-3">
        <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-yellow-100">
          <User className="h-6 w-6 text-yellow-800" />
        </div>
        <h1 className="font-display text-2xl font-bold text-stone-900">{profile.email}</h1>
      </div>

      {error && <p className="mb-3 text-sm text-red-600">{error}</p>}

      {profile.targets ? (
        <div className="overflow-hidden rounded-3xl bg-gradient-to-br from-yellow-700 to-yellow-800 p-5 text-white shadow-lg shadow-yellow-700/30">
          <p className="text-sm font-medium text-yellow-50">Meta: {profile.profile?.goal ?? ''}</p>
          <p className="font-display mt-1 text-4xl font-bold">{profile.targets.calories} kcal</p>
          <p className="text-sm text-yellow-50">
            {profile.targets.protein_g}g prot · {profile.targets.carbs_g}g carbos · {profile.targets.fat_g}g grasa
          </p>
        </div>
      ) : (
        <div className="card">
          <p className="text-sm text-stone-400">Todavía no tiene un plan calculado.</p>
        </div>
      )}

      <div className="mt-4 grid grid-cols-2 gap-3">
        <div className="card text-center">
          <Dumbbell className="mx-auto h-5 w-5 text-yellow-700" />
          <p className="font-display mt-1 text-2xl font-bold text-stone-900">{profile.workouts_this_week}</p>
          <p className="text-xs text-stone-500">entrenamientos esta semana</p>
        </div>
        <div className="card text-center">
          <TrendingUp className="mx-auto h-5 w-5 text-yellow-700" />
          <p className="font-display mt-1 text-2xl font-bold text-stone-900">{profile.total_workouts}</p>
          <p className="text-xs text-stone-500">entrenamientos en total</p>
        </div>
      </div>

      {profile.recent_checkins.length > 0 && (
        <div className="mt-4">
          <h2 className="font-display mb-2 text-lg font-bold text-stone-800">Check-ins recientes</h2>
          <div className="card space-y-1.5">
            {profile.recent_checkins.map((ci) => (
              <div key={ci.id} className="flex items-center justify-between text-sm">
                <span className="flex items-center gap-1.5 text-stone-500">
                  <CalendarDays className="h-3.5 w-3.5" />
                  {new Date(ci.logged_on).toLocaleDateString('es-PE', { day: 'numeric', month: 'short' })}
                </span>
                <span className="font-medium text-stone-900">{ci.weight_kg} kg</span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="mt-4">
        <div className="mb-2 flex items-center justify-between">
          <h2 className="font-display text-lg font-bold text-stone-800">Meta conjunta</h2>
          <button onClick={() => setShowNewGoal((v) => !v)} className="flex cursor-pointer items-center gap-1 text-sm font-semibold text-yellow-800 hover:text-yellow-900">
            <Plus className="h-4 w-4" /> Nueva
          </button>
        </div>

        {showNewGoal && (
          <NewGoalForm
            friendshipId={friendshipId}
            onCreated={() => {
              setShowNewGoal(false)
              refresh()
            }}
            onError={(msg) => setError(msg)}
          />
        )}

        {activeGoals.length === 0 && !showNewGoal && <p className="text-sm text-stone-400">Sin metas conjuntas activas.</p>}

        <div className="space-y-2">
          {activeGoals.map((goal) => (
            <GoalCard key={goal.id} goal={goal} onComplete={handleComplete} />
          ))}
        </div>

        {pastGoals.length > 0 && (
          <div className="mt-3 space-y-2 opacity-60">
            {pastGoals.map((goal) => (
              <GoalCard key={goal.id} goal={goal} onComplete={handleComplete} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function GoalCard({ goal, onComplete }: { goal: SharedGoal; onComplete: (id: number) => void }) {
  const progressEntries = Object.values(goal.progress ?? {})
  return (
    <div className="card">
      <div className="flex items-start justify-between gap-2">
        <div>
          <p className="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-yellow-700">
            <Target className="h-3.5 w-3.5" /> {GOAL_TYPE_LABELS[goal.goal_type]}
          </p>
          <p className="mt-1 text-sm font-medium text-stone-900">{goal.description}</p>
          {goal.target_date && <p className="mt-0.5 text-xs text-stone-400">Fecha límite: {goal.target_date}</p>}
        </div>
        {goal.status === 'active' && (
          <button
            onClick={() => onComplete(goal.id)}
            aria-label="Marcar como completada"
            className="flex h-8 w-8 shrink-0 cursor-pointer items-center justify-center rounded-full bg-green-600 text-white hover:bg-green-700"
          >
            <Check className="h-4 w-4" />
          </button>
        )}
        {goal.status === 'completed' && <Flag className="h-5 w-5 shrink-0 text-green-600" />}
      </div>

      {progressEntries.length > 0 && goal.goal_type !== 'custom' && (
        <div className="mt-2 grid grid-cols-2 gap-2">
          {progressEntries.map((p) => (
            <div key={p.user_id} className="rounded-lg bg-stone-50 px-2.5 py-1.5 text-center">
              <p className="truncate text-xs text-stone-500">{p.email}</p>
              <p className="text-sm font-semibold text-stone-900">
                {goal.goal_type === 'workouts_per_week' && `${p.workouts_this_week ?? 0}${goal.target_value ? ` / ${goal.target_value}` : ''}`}
                {goal.goal_type === 'weight_target' && (p.current_weight_kg != null ? `${p.current_weight_kg} kg` : '—')}
              </p>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function NewGoalForm({
  friendshipId,
  onCreated,
  onError,
}: {
  friendshipId: number
  onCreated: () => void
  onError: (msg: string) => void
}) {
  const [goalType, setGoalType] = useState<GoalType>('workouts_per_week')
  const [description, setDescription] = useState('')
  const [targetValue, setTargetValue] = useState('')
  const [targetDate, setTargetDate] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    try {
      await createSharedGoal(friendshipId, {
        goal_type: goalType,
        description,
        target_value: targetValue ? Number(targetValue) : undefined,
        target_date: targetDate || undefined,
      })
      onCreated()
    } catch (err) {
      onError(apiErrorMessage(err, 'No se pudo crear la meta.'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="card mb-2 space-y-2">
      <select value={goalType} onChange={(e) => setGoalType(e.target.value as GoalType)} className="input">
        <option value="workouts_per_week">Entrenamientos por semana</option>
        <option value="weight_target">Meta de peso</option>
        <option value="custom">Personalizada</option>
      </select>
      <input
        required
        placeholder="Ej. Entrenar 4 veces esta semana los dos"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        className="input"
      />
      {goalType !== 'custom' && (
        <input
          type="number"
          step="0.1"
          placeholder={goalType === 'workouts_per_week' ? 'Meta numérica (ej. 4)' : 'Peso objetivo (kg)'}
          value={targetValue}
          onChange={(e) => setTargetValue(e.target.value)}
          className="input"
        />
      )}
      <input type="date" value={targetDate} onChange={(e) => setTargetDate(e.target.value)} className="input" />
      <button type="submit" disabled={submitting} className="btn-primary w-full py-2.5">
        {submitting ? 'Creando…' : 'Crear meta conjunta'}
      </button>
    </form>
  )
}
