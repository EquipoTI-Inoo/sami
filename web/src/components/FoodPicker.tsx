import { Minus, Plus, Search, X } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { apiErrorMessage, getFoods, logFood, type MealType, type PeruvianFood } from '../services/api'

const CATEGORY_LABELS: Record<string, string> = {
  protein: 'Proteínas',
  carb: 'Carbohidratos',
  legume: 'Legumbres',
  vegetable: 'Vegetales',
  fruit: 'Frutas',
  fat: 'Grasas',
  dairy: 'Lácteos',
}

export default function FoodPicker({
  mealType,
  onClose,
  onLogged,
}: {
  mealType: MealType
  onClose: () => void
  onLogged: () => void
}) {
  const [foods, setFoods] = useState<PeruvianFood[]>([])
  const [loading, setLoading] = useState(true)
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<PeruvianFood | null>(null)
  const [servings, setServings] = useState(1)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    getFoods()
      .then(setFoods)
      .catch(() => setError('No se pudo cargar el catálogo de alimentos.'))
      .finally(() => setLoading(false))
  }, [])

  const results = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return foods
    return foods.filter((f) => f.name.toLowerCase().includes(q))
  }, [foods, query])

  const grouped = results.reduce<Record<string, PeruvianFood[]>>((acc, f) => {
    ;(acc[f.category] ??= []).push(f)
    return acc
  }, {})

  async function handleAdd() {
    if (!selected) return
    setSaving(true)
    setError('')
    try {
      await logFood({ food_id: selected.id, meal_type: mealType, servings })
      onLogged()
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo registrar el alimento.'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-30 flex items-end bg-stone-900/40" onClick={onClose}>
      <div onClick={(e) => e.stopPropagation()} className="max-h-[85vh] w-full overflow-y-auto rounded-t-3xl bg-white p-4">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="font-display text-xl font-bold text-stone-900">¿Qué comiste?</h2>
          <button onClick={onClose} aria-label="Cerrar" className="cursor-pointer rounded-full p-1.5 text-stone-400 hover:bg-stone-100">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="relative mb-3">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-stone-400" />
          <input
            autoFocus
            placeholder="Buscar alimento…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="input pl-9"
          />
        </div>

        {error && <p className="mb-2 text-sm text-red-600">{error}</p>}

        {selected ? (
          <div className="rounded-2xl bg-yellow-50 p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-semibold text-stone-900">{selected.name}</p>
                <p className="text-xs text-stone-500">
                  {selected.serving_desc} · {Math.round(selected.calories)} kcal
                </p>
              </div>
              <button onClick={() => setSelected(null)} className="cursor-pointer text-xs font-medium text-stone-500 hover:text-stone-700">
                Cambiar
              </button>
            </div>

            <div className="mt-3 flex items-center justify-center gap-4">
              <button
                onClick={() => setServings((s) => Math.max(0.5, Math.round((s - 0.5) * 2) / 2))}
                aria-label="Menos porciones"
                className="flex h-9 w-9 cursor-pointer items-center justify-center rounded-full border border-stone-300 bg-white text-stone-600 hover:bg-stone-50"
              >
                <Minus className="h-4 w-4" />
              </button>
              <div className="w-24 text-center">
                <p className="font-display text-2xl font-bold text-stone-900">{Math.round(selected.serving_grams * servings)}g</p>
                <p className="text-[11px] text-stone-400">{servings}× porción</p>
              </div>
              <button
                onClick={() => setServings((s) => Math.round((s + 0.5) * 2) / 2)}
                aria-label="Más porciones"
                className="flex h-9 w-9 cursor-pointer items-center justify-center rounded-full border border-stone-300 bg-white text-stone-600 hover:bg-stone-50"
              >
                <Plus className="h-4 w-4" />
              </button>
            </div>
            <p className="mt-1 text-center text-xs text-stone-500">≈ {Math.round(selected.calories * servings)} kcal</p>

            <button onClick={handleAdd} disabled={saving} className="btn-primary mt-3 w-full py-3">
              {saving ? 'Guardando…' : 'Agregar'}
            </button>
          </div>
        ) : (
          <>
            {loading && <p className="text-sm text-stone-400">Cargando…</p>}
            <div className="space-y-4">
              {Object.entries(grouped).map(([category, items]) => (
                <div key={category}>
                  <h3 className="mb-1.5 text-xs font-bold uppercase tracking-wide text-stone-400">
                    {CATEGORY_LABELS[category] ?? category}
                  </h3>
                  <div className="space-y-1">
                    {items.map((f) => (
                      <button
                        key={f.id}
                        onClick={() => setSelected(f)}
                        className="flex w-full cursor-pointer items-center justify-between rounded-xl px-3 py-2.5 text-left text-sm hover:bg-yellow-50"
                      >
                        <span className="text-stone-800">
                          {f.name}
                          <span className="block text-xs text-stone-400">{Math.round(f.serving_grams)}g de referencia</span>
                        </span>
                        <span className="shrink-0 text-xs text-stone-400">{Math.round(f.calories)} kcal</span>
                      </button>
                    ))}
                  </div>
                </div>
              ))}
              {!loading && results.length === 0 && <p className="text-sm text-stone-400">Sin resultados.</p>}
            </div>
          </>
        )}
      </div>
    </div>
  )
}
