import { Check, Plus, Salad } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import BottomNav from '../components/BottomNav'
import { apiErrorMessage, getMealPlan, logFood, type Meal, type MealPlan, type MealType } from '../services/api'

const MEAL_TYPE_BY_NAME: Record<string, MealType> = {
  desayuno: 'desayuno',
  almuerzo: 'almuerzo',
  cena: 'cena',
}

export default function Diet() {
  const navigate = useNavigate()
  const [plan, setPlan] = useState<MealPlan | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    getMealPlan()
      .then((p) => {
        if (!p) {
          navigate('/chat')
          return
        }
        setPlan(p)
      })
      .catch(() => setError('No se pudo cargar tu dieta recomendada.'))
      .finally(() => setLoading(false))
  }, [navigate])

  return (
    <div className="min-h-screen bg-cream px-4 pb-28 pt-6">
      <header className="mb-5 flex items-center gap-2">
        <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-yellow-700 shadow-sm shadow-yellow-700/40">
          <Salad className="h-5 w-5 text-white" strokeWidth={2.25} />
        </div>
        <h1 className="font-display text-2xl font-bold tracking-tight text-stone-900">Tu dieta recomendada</h1>
      </header>

      {loading && <p className="text-sm text-stone-400">Cargando…</p>}
      {error && <p className="text-sm text-red-600">{error}</p>}

      {plan && (
        <>
          <div className="overflow-hidden rounded-3xl bg-gradient-to-br from-yellow-700 to-yellow-900 p-5 text-white shadow-lg shadow-yellow-900/30">
            <p className="text-sm font-medium text-yellow-50">Meta diaria</p>
            <p className="font-display mt-1 text-4xl font-bold">{plan.target.calories} kcal</p>
            <p className="mt-1 text-sm text-yellow-50">
              {plan.target.protein_g}g proteína · {plan.target.carbs_g}g carbos · {plan.target.fat_g}g grasa
            </p>
          </div>

          <p className="mt-3 text-xs text-stone-400">
            Armado con lo que nos contaste que te gusta — es un punto de partida, no una receta fija. Toca{' '}
            <Plus className="inline h-3 w-3 align-text-top" /> en cualquier alimento para registrarlo directo en tu diario de hoy.
          </p>

          <div className="mt-4 space-y-3">
            {plan.meals.map((meal) => (
              <DietMealCard key={meal.name} meal={meal} />
            ))}
          </div>
        </>
      )}

      <BottomNav />
    </div>
  )
}

function DietMealCard({ meal }: { meal: Meal }) {
  const [loggedIds, setLoggedIds] = useState<Set<number>>(new Set())
  const [error, setError] = useState('')
  const mealType = MEAL_TYPE_BY_NAME[meal.name.toLowerCase()]

  async function handleQuickLog(foodId: number, servings: number) {
    if (!mealType) return
    setError('')
    try {
      await logFood({ food_id: foodId, meal_type: mealType, servings })
      setLoggedIds((prev) => new Set(prev).add(foodId))
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo registrar.'))
    }
  }

  return (
    <div className="card">
      <div className="mb-2 flex items-baseline justify-between">
        <h3 className="text-sm font-semibold capitalize text-stone-900">{meal.name}</h3>
        <span className="text-xs text-stone-400">{meal.calories} kcal</span>
      </div>
      {error && <p className="mb-2 text-xs text-red-600">{error}</p>}
      <ul className="space-y-1.5">
        {meal.items.map((item) => {
          const logged = loggedIds.has(item.food.id)
          return (
            <li key={item.food.id} className="flex items-center justify-between text-sm">
              <span className="text-stone-700">
                {item.food.name}
                <span className="text-stone-400"> · {item.total_grams}g</span>
              </span>
              <span className="flex shrink-0 items-center gap-2">
                <span className="text-xs text-stone-400">{item.calories} kcal</span>
                {mealType && (
                  <button
                    onClick={() => handleQuickLog(item.food.id, item.servings)}
                    disabled={logged}
                    aria-label={logged ? 'Ya registrado' : 'Registrar en mi diario'}
                    className={`flex h-6 w-6 cursor-pointer items-center justify-center rounded-full transition-colors duration-200 ${
                      logged ? 'bg-green-100 text-green-700' : 'bg-yellow-50 text-yellow-700 hover:bg-yellow-100'
                    }`}
                  >
                    {logged ? <Check className="h-3.5 w-3.5" /> : <Plus className="h-3.5 w-3.5" />}
                  </button>
                )}
              </span>
            </li>
          )
        })}
      </ul>
    </div>
  )
}
