import { ArrowLeft, ClipboardCheck } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiErrorMessage, submitCheckin } from '../services/api'

export default function Checkin() {
  const navigate = useNavigate()
  const [weightKg, setWeightKg] = useState('')
  const [waistCm, setWaistCm] = useState('')
  const [hipCm, setHipCm] = useState('')
  const [armCm, setArmCm] = useState('')
  const [bodyFatPct, setBodyFatPct] = useState('')
  const [note, setNote] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      await submitCheckin({
        weight_kg: Number(weightKg),
        waist_cm: waistCm ? Number(waistCm) : undefined,
        hip_cm: hipCm ? Number(hipCm) : undefined,
        arm_cm: armCm ? Number(armCm) : undefined,
        body_fat_pct: bodyFatPct ? Number(bodyFatPct) : undefined,
        note: note.trim() || undefined,
      })
      navigate('/')
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo guardar tu check-in.'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="min-h-screen bg-cream px-4 py-6">
      <button
        onClick={() => navigate('/')}
        className="mb-4 flex items-center gap-1 text-sm font-medium text-stone-500 hover:text-stone-700"
      >
        <ArrowLeft className="h-4 w-4" /> Volver
      </button>

      <form onSubmit={handleSubmit} className="mx-auto w-full max-w-sm space-y-4 rounded-3xl bg-white p-6 shadow-sm shadow-stone-300/30 ring-1 ring-stone-100">
        <div className="flex items-center gap-3">
          <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-amber-100">
            <ClipboardCheck className="h-6 w-6 text-amber-600" />
          </div>
          <div>
            <h1 className="font-display text-2xl font-bold text-stone-900">Check-in semanal</h1>
            <p className="mt-0.5 text-sm text-stone-500">Solo el peso es obligatorio.</p>
          </div>
        </div>

        <div className="space-y-1">
          <label htmlFor="weight" className="text-sm font-medium text-stone-700">
            Peso (kg)
          </label>
          <input
            id="weight"
            type="number"
            step="0.1"
            required
            min={30}
            max={300}
            value={weightKg}
            onChange={(e) => setWeightKg(e.target.value)}
            className="input"
          />
        </div>

        <div className="grid grid-cols-3 gap-3">
          <div className="space-y-1">
            <label htmlFor="waist" className="text-sm font-medium text-stone-700">
              Cintura (cm)
            </label>
            <input id="waist" type="number" step="0.1" value={waistCm} onChange={(e) => setWaistCm(e.target.value)} className="input" />
          </div>
          <div className="space-y-1">
            <label htmlFor="hip" className="text-sm font-medium text-stone-700">
              Cadera (cm)
            </label>
            <input id="hip" type="number" step="0.1" value={hipCm} onChange={(e) => setHipCm(e.target.value)} className="input" />
          </div>
          <div className="space-y-1">
            <label htmlFor="arm" className="text-sm font-medium text-stone-700">
              Brazo (cm)
            </label>
            <input id="arm" type="number" step="0.1" value={armCm} onChange={(e) => setArmCm(e.target.value)} className="input" />
          </div>
        </div>

        <div className="space-y-1">
          <label htmlFor="bodyfat" className="text-sm font-medium text-stone-700">
            % de grasa corporal (si lo mides)
          </label>
          <input
            id="bodyfat"
            type="number"
            step="0.1"
            min={3}
            max={70}
            value={bodyFatPct}
            onChange={(e) => setBodyFatPct(e.target.value)}
            className="input"
          />
        </div>

        <div className="space-y-1">
          <label htmlFor="note" className="text-sm font-medium text-stone-700">
            ¿Cómo te sentiste esta semana?
          </label>
          <textarea
            id="note"
            rows={3}
            placeholder="Energía, sueño, antojos, lo que quieras contarle a tu coach…"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            className="input resize-none"
          />
        </div>

        {error && <p className="text-sm text-red-600">{error}</p>}

        <button type="submit" disabled={submitting} className="btn-primary w-full py-3">
          {submitting ? 'Guardando…' : 'Guardar check-in'}
        </button>
      </form>
    </div>
  )
}
