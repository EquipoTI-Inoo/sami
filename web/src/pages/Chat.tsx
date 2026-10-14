import { Check, Flame, Send, X } from 'lucide-react'
import { useEffect, useRef, useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import BottomNav from '../components/BottomNav'
import {
  apiErrorMessage,
  approvePendingChange,
  getChatHistory,
  getPendingChanges,
  rejectPendingChange,
  sendChatMessage,
  type ChatHistoryMessage,
  type PendingChange,
} from '../services/api'

const GREETING = '¡Hola! Soy Sami, tu coach. Cuéntame de ti — edad, qué buscas, cómo entrenas — y te armo un plan a tu medida.'

interface DisplayMessage extends ChatHistoryMessage {
  id: string
}

let msgId = 0
function nextId() {
  msgId += 1
  return `m${msgId}`
}

export default function Chat() {
  const navigate = useNavigate()
  const [messages, setMessages] = useState<DisplayMessage[]>([])
  const [input, setInput] = useState('')
  const [sending, setSending] = useState(false)
  const [loadingHistory, setLoadingHistory] = useState(true)
  const [error, setError] = useState('')
  const [onboardingDone, setOnboardingDone] = useState(false)
  const [pendingChange, setPendingChange] = useState<PendingChange | null>(null)
  const [resolvingChange, setResolvingChange] = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    getChatHistory()
      .then((history) => {
        if (history.length === 0) {
          setMessages([{ id: nextId(), role: 'assistant', text: GREETING }])
        } else {
          setMessages(history.map((m) => ({ ...m, id: nextId() })))
        }
      })
      .catch(() => {
        setMessages([{ id: nextId(), role: 'assistant', text: GREETING }])
      })
      .finally(() => setLoadingHistory(false))

    getPendingChanges()
      .then((pending) => setPendingChange(pending[0] ?? null))
      .catch(() => {})
  }, [])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, sending])

  useEffect(() => {
    if (!loadingHistory && !sending) inputRef.current?.focus()
  }, [loadingHistory, sending])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const text = input.trim()
    if (!text || sending) return

    setError('')
    setInput('')
    setMessages((prev) => [...prev, { id: nextId(), role: 'user', text }])
    setSending(true)

    try {
      const res = await sendChatMessage(text)
      setMessages((prev) => [...prev, { id: nextId(), role: 'assistant', text: res.reply }])
      if (res.onboarding_completed) setOnboardingDone(true)
      if (res.pending_change) setPendingChange(res.pending_change)
    } catch (err) {
      setError(apiErrorMessage(err, 'Tu coach no está disponible ahora mismo.'))
    } finally {
      setSending(false)
    }
  }

  async function handleResolveChange(approve: boolean) {
    if (!pendingChange || resolvingChange) return
    setResolvingChange(true)
    try {
      const resolved = approve ? await approvePendingChange(pendingChange.id) : await rejectPendingChange(pendingChange.id)
      setPendingChange(null)
      setMessages((prev) => [
        ...prev,
        {
          id: nextId(),
          role: 'assistant',
          text: resolved.status === 'approved' ? 'Listo, actualicé tu plan con los nuevos valores.' : 'Entendido, dejo tu plan como está.',
        },
      ])
    } catch (err) {
      setError(apiErrorMessage(err, 'No se pudo guardar tu decisión.'))
    } finally {
      setResolvingChange(false)
    }
  }

  return (
    <div className="flex min-h-screen flex-col bg-cream pb-16">
      <header className="flex items-center gap-2 border-b border-stone-200 bg-white px-4 py-3">
        <div className="flex h-8 w-8 items-center justify-center rounded-full bg-yellow-700">
          <Flame className="h-4 w-4 text-white" strokeWidth={2.25} />
        </div>
        <h1 className="font-display text-xl font-bold text-stone-900">Sami</h1>
      </header>

      <div className="flex-1 space-y-3 overflow-y-auto px-4 py-4">
        {loadingHistory ? (
          <p className="text-sm text-stone-400">Cargando conversación…</p>
        ) : (
          messages.map((m) => <ChatBubble key={m.id} from={m.role} text={m.text} />)
        )}
        {sending && <TypingBubble />}
        {pendingChange && (
          <PendingChangeCard change={pendingChange} resolving={resolvingChange} onResolve={handleResolveChange} />
        )}
        <div ref={bottomRef} />
      </div>

      {onboardingDone && (
        <div className="border-t border-stone-200 bg-white p-4">
          <button onClick={() => navigate('/')} className="btn-primary w-full py-3">
            Ir a mi dashboard
          </button>
        </div>
      )}

      <form onSubmit={handleSubmit} className="border-t border-stone-200 bg-white p-4">
        {error && <p className="mb-2 text-xs text-red-600">{error}</p>}
        <div className="flex gap-2">
          <input
            ref={inputRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            disabled={sending || loadingHistory}
            placeholder="Escríbele a tu coach…"
            className="input flex-1"
          />
          <button
            type="submit"
            disabled={sending || loadingHistory || !input.trim()}
            aria-label="Enviar"
            className="btn-primary px-4"
          >
            <Send className="h-4 w-4" />
          </button>
        </div>
      </form>

      <BottomNav />
    </div>
  )
}

function ChatBubble({ from, text }: { from: 'coach' | 'user' | 'assistant'; text: string }) {
  const isCoach = from !== 'user'
  return (
    <div className={`flex items-end gap-2 ${isCoach ? '' : 'justify-end'}`}>
      {isCoach && (
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-yellow-700">
          <Flame className="h-4 w-4 text-white" strokeWidth={2.25} />
        </div>
      )}
      <div
        className={`max-w-[80%] whitespace-pre-line rounded-2xl px-4 py-2.5 text-sm ${
          isCoach ? 'bg-white text-stone-900 shadow-sm shadow-stone-300/30 ring-1 ring-stone-100' : 'bg-yellow-700 text-white'
        }`}
      >
        {text}
      </div>
    </div>
  )
}

function PendingChangeCard({
  change,
  resolving,
  onResolve,
}: {
  change: PendingChange
  resolving: boolean
  onResolve: (approve: boolean) => void
}) {
  const { payload } = change
  return (
    <div className="ml-10 max-w-[80%] rounded-2xl border border-amber-200 bg-amber-50 p-4">
      <p className="text-sm font-semibold text-amber-900">Tu coach propone un cambio de plan</p>
      <p className="mt-1 text-xs text-amber-700">{change.reason}</p>
      <div className="mt-3 grid grid-cols-4 gap-2 text-center">
        <PendingMacro label="kcal" value={payload.calories} />
        <PendingMacro label="Prot" value={payload.protein_g} suffix="g" />
        <PendingMacro label="Carb" value={payload.carbs_g} suffix="g" />
        <PendingMacro label="Grasa" value={payload.fat_g} suffix="g" />
      </div>
      <div className="mt-3 flex gap-2">
        <button
          onClick={() => onResolve(true)}
          disabled={resolving}
          className="flex flex-1 cursor-pointer items-center justify-center gap-1 rounded-lg bg-green-600 py-2 text-xs font-semibold text-white transition-colors duration-200 hover:bg-green-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <Check className="h-3.5 w-3.5" /> Sí, aplícalo
        </button>
        <button
          onClick={() => onResolve(false)}
          disabled={resolving}
          className="flex flex-1 cursor-pointer items-center justify-center gap-1 rounded-lg border border-amber-300 bg-white py-2 text-xs font-semibold text-amber-900 transition-colors duration-200 hover:bg-amber-50 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <X className="h-3.5 w-3.5" /> Déjalo así
        </button>
      </div>
    </div>
  )
}

function PendingMacro({ label, value, suffix }: { label: string; value: number; suffix?: string }) {
  return (
    <div className="rounded-lg bg-white py-1.5">
      <p className="text-sm font-semibold text-stone-900">
        {value}
        {suffix}
      </p>
      <p className="text-[10px] text-stone-500">{label}</p>
    </div>
  )
}

function TypingBubble() {
  return (
    <div className="flex items-end gap-2">
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-yellow-700">
        <Flame className="h-4 w-4 text-white" strokeWidth={2.25} />
      </div>
      <div className="flex gap-1 rounded-2xl bg-white px-4 py-3 shadow-sm shadow-stone-300/30 ring-1 ring-stone-100">
        <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-stone-400 [animation-delay:-0.3s]" />
        <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-stone-400 [animation-delay:-0.15s]" />
        <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-stone-400" />
      </div>
    </div>
  )
}
