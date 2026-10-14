import { ChevronRight, Flame, LogOut, Moon, Plus, Salad, Sparkles, Sun, Sunrise, Trash2, Utensils, type LucideIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import CalorieRing from '../components/CalorieRing'
import FoodPicker from '../components/FoodPicker'
import BottomNav from '../components/BottomNav'
import { useAuth } from '../context/useAuth'
import {
  deleteFoodLogEntry,
  getCheckinStatus,
  getDailyFoodLog,
  getMealPlan,
  getTargetsHistory,
  type CheckinStatus,
  type DailyFoodLog,
  type MealPlan,
  type MealType,
  type NutritionTargets,
} from '../services/api'

const MEAL_SECTIONS: { type: MealType; label: string; icon: LucideIcon }[] = [
  { type: 'desayuno', label: 'Desayuno', icon: Sunrise },
  { type: 'almuerzo', label: 'Almuerzo', icon: Sun },
  { type: 'cena', label: 'Cena', icon: Moon },
  { type: 'snack', label: 'Snacks', icon: Utensils },
]

export default function Dashboard() {
  const { logout } = useAuth()
  const navigate = useNavigate()
  const [dailyLog, setDailyLog] = useState<DailyFoodLog | null>(null)
  const [plan, setPlan] = useState<MealPlan | null>(null)
  const [loading, setLoading] = useState(true)
  const [checkinStatus, setCheckinStatus] = useState<CheckinStatus | null>(null)
  const [history, setHistory] = useState<NutritionTargets[]>([])
  const [pickerMeal, setPickerMeal] = useState<MealType | null>(null)

  function refreshLog() {
    return getDailyFoodLog().then(setDailyLog)
  }

  useEffect(() => {
    getMealPlan()
      .then((p) => {
        if (!p) {
          navigate('/chat')
          return
        }
        setPlan(p)
        refreshLog()
        getCheckinStatus().then(setCheckinStatus)
        getTargetsHistory().then(setHistory)
      })
      .finally(() => setLoading(false))
  }, [navigate])

  async function handleRemoveEntry(id: number) {
    await deleteFoodLogEntry(id)
    await refreshLog()
  }

  const target = dailyLog?.target
  const consumed = dailyLog?.consumed

  return (
    <div className="min-h-screen bg-cream px-4 pb-28 pt-6">
      <header className="mb-5 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-yellow-700 shadow-sm shadow-yellow-700/40">
            <Flame className="h-5 w-5 text-white" strokeWidth={2.25} />
          </div>
          <h1 className="font-display text-2xl font-bold tracking-tight text-stone-900">Sami</h1>
        </div>
        <button
          onClick={logout}
          aria-label="Cerrar sesión"
          className="flex h-10 w-10 cursor-pointer items-center justify-center rounded-full text-stone-400 transition-colors duration-200 hover:bg-stone-100 hover:text-stone-600"
        >
          <LogOut className="h-5 w-5" />
        </button>
      </header>

      {checkinStatus?.needs_checkin && (
        <div className="mb-4 flex items-center justify-between gap-3 rounded-2xl bg-amber-50 p-4 ring-1 ring-amber-200">
          <div>
            <p className="text-sm font-semibold text-amber-900">Toca hacer tu check-in semanal</p>
            <p className="text-xs text-amber-700">
              {checkinStatus.days_since_last !== undefined
                ? `Han pasado ${checkinStatus.days_since_last} días desde tu último registro.`
                : 'Registra peso y cómo te sentiste esta semana.'}
            </p>
          </div>
          <button onClick={() => navigate('/checkin')} className="btn-primary shrink-0 bg-amber-600 px-3 py-2 text-xs shadow-amber-600/30 hover:bg-amber-700">
            Check-in
          </button>
        </div>
      )}

      {loading && <p className="text-sm text-stone-400">Cargando tu día…</p>}

      {target && consumed && (
        <>
          <div className="overflow-hidden rounded-3xl bg-gradient-to-br from-yellow-700 to-yellow-900 p-5 text-white shadow-lg shadow-yellow-900/30">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-yellow-50">Hoy</p>
                <p className="font-display text-2xl font-bold">
                  {consumed.calories} <span className="text-base font-normal text-yellow-50">/ {target.calories} kcal</span>
                </p>
              </div>
              <CalorieRing consumed={consumed.calories} target={target.calories} />
            </div>
            <div className="mt-4 grid grid-cols-3 gap-2 text-center">
              <MacroPill label="Proteína" value={consumed.protein_g} of={target.protein_g} />
              <MacroPill label="Carbos" value={consumed.carbs_g} of={target.carbs_g} />
              <MacroPill label="Grasa" value={consumed.fat_g} of={target.fat_g} />
            </div>
          </div>

          <div className="mt-6 space-y-3">
            {MEAL_SECTIONS.map((section) => (
              <MealLogSection
                key={section.type}
                section={section}
                entries={dailyLog?.entries.filter((e) => e.meal_type === section.type) ?? []}
                onAdd={() => setPickerMeal(section.type)}
                onRemove={handleRemoveEntry}
              />
            ))}
          </div>

          {plan && (
            <div className="mt-6">
              <h2 className="font-display mb-2 text-lg font-bold text-stone-800">Tu dieta recomendada</h2>
              <button onClick={() => navigate('/diet')} className="card flex w-full cursor-pointer items-center justify-between text-left">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-yellow-50">
                    <Salad className="h-5 w-5 text-yellow-700" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-stone-900">{plan.calories} kcal sugeridas para hoy</p>
                    <p className="text-xs text-stone-500">
                      {plan.meals.map((m) => `${m.name} ${m.calories} kcal`).join(' · ')}
                    </p>
                  </div>
                </div>
                <ChevronRight className="h-4 w-4 shrink-0 text-stone-300" />
              </button>
            </div>
          )}

          {history.length > 1 && (
            <div className="mt-6">
              <h2 className="font-display mb-2 text-lg font-bold text-stone-800">Historial de tu plan</h2>
              <div className="space-y-2">
                {history.map((t) => (
                  <TargetsHistoryRow key={t.id} targets={t} />
                ))}
              </div>
            </div>
          )}
        </>
      )}

      <BottomNav />

      {pickerMeal && (
        <FoodPicker
          mealType={pickerMeal}
          onClose={() => setPickerMeal(null)}
          onLogged={() => {
            setPickerMeal(null)
            refreshLog()
          }}
        />
      )}
    </div>
  )
}

function MealLogSection({
  section,
  entries,
  onAdd,
  onRemove,
}: {
  section: { type: MealType; label: string; icon: LucideIcon }
  entries: DailyFoodLog['entries']
  onAdd: () => void
  onRemove: (id: number) => void
}) {
  const Icon = section.icon
  const totalKcal = entries.reduce((sum, e) => sum + e.calories, 0)

  return (
    <div className="card">
      <div className="mb-2 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-yellow-50">
            <Icon className="h-4 w-4 text-yellow-700" />
          </div>
          <h3 className="text-sm font-semibold text-stone-900">{section.label}</h3>
        </div>
        <div className="flex items-center gap-2">
          {totalKcal > 0 && <span className="text-xs text-stone-400">{totalKcal} kcal</span>}
          <button
            onClick={onAdd}
            aria-label={`Agregar a ${section.label}`}
            className="flex h-7 w-7 cursor-pointer items-center justify-center rounded-full bg-yellow-700 text-white hover:bg-yellow-800"
          >
            <Plus className="h-4 w-4" />
          </button>
        </div>
      </div>
      {entries.length === 0 ? (
        <p className="text-xs text-stone-400">Nada registrado todavía.</p>
      ) : (
        <ul className="space-y-1">
          {entries.map((e) => (
            <li key={e.id} className="group flex items-center justify-between text-sm">
              <span className="text-stone-700">
                {e.food?.name}
                <span className="text-stone-400"> · {e.total_grams}g</span>
              </span>
              <span className="flex shrink-0 items-center gap-2">
                <span className="text-xs text-stone-400">{e.calories} kcal</span>
                <button onClick={() => onRemove(e.id)} aria-label="Quitar" className="cursor-pointer text-stone-300 hover:text-red-500">
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function TargetsHistoryRow({ targets }: { targets: NutritionTargets }) {
  const date = new Date(targets.created_at).toLocaleDateString('es-PE', { day: 'numeric', month: 'short', year: 'numeric' })
  const isAi = targets.updated_by === 'ai'
  return (
    <div className="card">
      <div className="flex items-baseline justify-between">
        <span className="text-sm font-semibold text-stone-900">{targets.calories} kcal</span>
        <span className="text-xs text-stone-400">{date}</span>
      </div>
      <p className="mt-1 text-xs text-stone-500">
        {targets.protein_g}g proteína · {targets.carbs_g}g carbos · {targets.fat_g}g grasa
      </p>
      <p className="mt-1.5 flex items-center gap-1 text-xs text-stone-400">
        {isAi && <Sparkles className="h-3 w-3 text-yellow-700" />}
        {isAi ? 'Ajustado por tu coach' : 'Ajustado por ti'}
        {targets.reason ? ` — ${targets.reason}` : ''}
      </p>
    </div>
  )
}

function MacroPill({ label, value, of }: { label: string; value: number; of: number }) {
  return (
    <div className="rounded-xl bg-white/15 py-2">
      <p className="text-base font-bold text-white">
        {value}
        <span className="text-xs font-normal text-yellow-50">/{of}g</span>
      </p>
      <p className="text-[11px] text-yellow-50">{label}</p>
    </div>
  )
}
