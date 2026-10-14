import { Check, ChevronRight, Mail, UserPlus, Users, X } from 'lucide-react'
import { useEffect, useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import BottomNav from '../components/BottomNav'
import {
  acceptFriendRequest,
  apiErrorMessage,
  getIncomingFriendRequests,
  listFriends,
  rejectFriendRequest,
  sendFriendRequest,
  type Friendship,
} from '../services/api'

export default function Friends() {
  const navigate = useNavigate()
  const [friends, setFriends] = useState<Friendship[]>([])
  const [incoming, setIncoming] = useState<Friendship[]>([])
  const [loading, setLoading] = useState(true)
  const [email, setEmail] = useState('')
  const [sending, setSending] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  function refresh() {
    return Promise.all([listFriends(), getIncomingFriendRequests()]).then(([f, i]) => {
      setFriends(f)
      setIncoming(i)
    })
  }

  useEffect(() => {
    refresh()
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  async function handleSendRequest(e: FormEvent) {
    e.preventDefault()
    setError('')
    setNotice('')
    setSending(true)
    try {
      await sendFriendRequest(email)
      setEmail('')
      setNotice('Solicitud enviada.')
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo enviar la solicitud.'))
    } finally {
      setSending(false)
    }
  }

  async function handleAccept(id: number) {
    try {
      await acceptFriendRequest(id)
      await refresh()
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo aceptar la solicitud.'))
    }
  }

  async function handleReject(id: number) {
    try {
      await rejectFriendRequest(id)
      await refresh()
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo rechazar la solicitud.'))
    }
  }

  return (
    <div className="min-h-screen bg-cream px-4 pb-28 pt-6">
      <header className="mb-5 flex items-center gap-2">
        <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-yellow-700 shadow-sm shadow-yellow-700/40">
          <Users className="h-5 w-5 text-white" strokeWidth={2.25} />
        </div>
        <h1 className="font-display text-2xl font-bold tracking-tight text-stone-900">Amigos</h1>
      </header>

      <form onSubmit={handleSendRequest} className="card">
        <p className="mb-2 text-sm font-semibold text-stone-800">Agregar amigo</p>
        <div className="flex gap-2">
          <input
            type="email"
            required
            placeholder="email@ejemplo.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="input flex-1"
          />
          <button type="submit" disabled={sending} aria-label="Enviar solicitud" className="btn-primary px-4">
            <Mail className="h-4 w-4" />
          </button>
        </div>
        {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
        {notice && <p className="mt-2 text-sm text-green-700">{notice}</p>}
      </form>

      {incoming.length > 0 && (
        <div className="mt-4">
          <h2 className="font-display mb-2 text-lg font-bold text-stone-800">Solicitudes</h2>
          <div className="space-y-2">
            {incoming.map((req) => (
              <div key={req.id} className="card flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="flex h-9 w-9 items-center justify-center rounded-full bg-stone-100">
                    <UserPlus className="h-4 w-4 text-stone-500" />
                  </div>
                  <p className="text-sm font-medium text-stone-900">{req.friend_email}</p>
                </div>
                <div className="flex gap-1.5">
                  <button
                    onClick={() => handleAccept(req.id)}
                    aria-label="Aceptar"
                    className="flex h-9 w-9 cursor-pointer items-center justify-center rounded-full bg-green-600 text-white hover:bg-green-700"
                  >
                    <Check className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => handleReject(req.id)}
                    aria-label="Rechazar"
                    className="flex h-9 w-9 cursor-pointer items-center justify-center rounded-full border border-stone-300 text-stone-500 hover:bg-stone-50"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="mt-4">
        <h2 className="font-display mb-2 text-lg font-bold text-stone-800">Tus amigos</h2>
        {loading && <p className="text-sm text-stone-400">Cargando…</p>}
        {!loading && friends.length === 0 && <p className="text-sm text-stone-400">Todavía no tienes amigos agregados.</p>}
        <div className="space-y-2">
          {friends.map((f) => (
            <button key={f.id} onClick={() => navigate(`/friends/${f.id}`)} className="card flex w-full cursor-pointer items-center justify-between text-left">
              <div className="flex items-center gap-2">
                <div className="flex h-9 w-9 items-center justify-center rounded-full bg-yellow-50">
                  <Users className="h-4 w-4 text-yellow-700" />
                </div>
                <p className="text-sm font-medium text-stone-900">{f.friend_email}</p>
              </div>
              <ChevronRight className="h-4 w-4 text-stone-300" />
            </button>
          ))}
        </div>
      </div>

      <BottomNav />
    </div>
  )
}
