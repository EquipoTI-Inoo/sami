import { Flame } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/useAuth'
import { apiErrorMessage } from '../services/api'

export default function Login() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      await login(email, password)
      navigate('/')
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo iniciar sesión.'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-cream via-cream to-yellow-100/60 px-4">
      <form onSubmit={handleSubmit} className="w-full max-w-sm space-y-5 rounded-3xl bg-white p-7 shadow-lg shadow-yellow-900/5 ring-1 ring-stone-100">
        <div className="flex flex-col items-center gap-3 text-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-yellow-700 shadow-sm shadow-yellow-700/40">
            <Flame className="h-7 w-7 text-white" strokeWidth={2.25} />
          </div>
          <div>
            <h1 className="font-display text-3xl font-bold tracking-tight text-stone-900">Sami</h1>
            <p className="mt-1 text-sm text-stone-500">Inicia sesión para continuar.</p>
          </div>
        </div>

        <div className="space-y-1">
          <label htmlFor="email" className="text-sm font-medium text-stone-700">
            Email
          </label>
          <input
            id="email"
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="input"
          />
        </div>

        <div className="space-y-1">
          <label htmlFor="password" className="text-sm font-medium text-stone-700">
            Contraseña
          </label>
          <input
            id="password"
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="input"
          />
        </div>

        {error && <p className="text-sm text-red-600">{error}</p>}

        <button type="submit" disabled={submitting} className="btn-primary w-full py-3">
          {submitting ? 'Entrando…' : 'Entrar'}
        </button>

        <p className="text-center text-sm text-stone-500">
          ¿No tienes cuenta?{' '}
          <Link to="/register" className="font-semibold text-yellow-800 hover:text-yellow-900">
            Regístrate
          </Link>
        </p>
      </form>
    </div>
  )
}
