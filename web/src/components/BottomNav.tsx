import { Dumbbell, Home, MessageCircle, Salad, Users } from 'lucide-react'
import { useLocation, useNavigate } from 'react-router-dom'

const TABS = [
  { to: '/', label: 'Inicio', icon: Home },
  { to: '/chat', label: 'Sami', icon: MessageCircle },
  { to: '/diet', label: 'Dieta', icon: Salad },
  { to: '/workouts', label: 'Entrenar', icon: Dumbbell },
  { to: '/friends', label: 'Amigos', icon: Users },
]

export default function BottomNav() {
  const { pathname } = useLocation()
  const navigate = useNavigate()

  return (
    <nav className="fixed inset-x-0 bottom-0 z-30 border-t border-stone-200 bg-white/95 backdrop-blur-sm">
      <div className="mx-auto flex max-w-lg items-stretch justify-around px-2 pb-[env(safe-area-inset-bottom)]">
        {TABS.map(({ to, label, icon: Icon }) => {
          const active = pathname === to
          return (
            <button
              key={to}
              onClick={() => navigate(to)}
              className={`flex min-h-[56px] flex-1 cursor-pointer flex-col items-center justify-center gap-0.5 py-2 transition-colors duration-200 ${
                active ? 'text-yellow-800' : 'text-stone-400 hover:text-stone-600'
              }`}
            >
              <Icon className="h-6 w-6" strokeWidth={active ? 2.5 : 2} />
              <span className={`text-[11px] ${active ? 'font-semibold' : 'font-medium'}`}>{label}</span>
            </button>
          )
        })}
      </div>
    </nav>
  )
}
