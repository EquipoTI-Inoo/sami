# Sami — PRD (v0.1)

> Nombre definitivo: **Sami** — se cambió desde "CoachIA" (que antes se había cambiado
> desde "Fuelr"). El diferencial real de la plataforma es que se siente como tu nutri y
> coach personal, no como un producto genérico de IA — necesitaba un nombre de persona,
> cálido, no un acrónimo. "Sami" viene del quechua *sami* (energía vital / espíritu):
> encaja con una app de fitness/nutrición y suena a nombre propio corto y fácil de decir.
> El chat sigue siendo "Sami" en primera persona (`backend/controllers/chat.go`), no un
> asistente sin nombre.

Coach/nutricionista personal con IA, proyecto nuevo inspirado en los patrones de
[Lyftr](https://github.com/Cawlumm/lyftr) (mismo tipo de stack: Go + React + SQLite,
self-hosted) pero con código propio desde cero — pensado primero para 2 usuarios (tú y
tu pareja), web-only.

---

## 1. Visión

Lyftr resuelve el tracking de entrenamientos. Sami añade la capa que Lyftr no tiene:
alguien (algo) que interprete esos datos y los de nutrición/cuerpo, y te hable — como
haría un coach o nutricionista real en una consulta de seguimiento — en vez de dejarte
solo con gráficas.

La diferencia frente a MyFitnessPal/Fitia/etc.: el chat con IA no es un gimmick aparte,
es la interfaz principal de ajuste. No editas un formulario de "nuevo objetivo calórico";
le dices "esta semana entrené menos, bájame los carbos" y la IA lo ajusta y explica por qué.

## 2. Usuarios y contexto

- **v0**: 2 usuarios (tú + tu pareja), cuentas separadas, cada uno con su propio perfil,
  metas y conversación con la IA.
- Self-hosted, como Lyftr — un servidor, un SQLite, sin multi-tenant ni facturación.
- Comida de referencia: **peruana / lo que hay en una cocina peruana promedio** — no una
  base de datos genérica de USDA. Esto es la decisión de producto más importante del
  módulo de nutrición (ver §5.3).

## 3. Objetivos (qué significa "funciona")

1. Al terminar el onboarding, cada usuario tiene un objetivo calórico/macro y un plan de
   entrenamiento (gym u hogar) sin haber tenido que saber nada de nutrición de antemano.
2. Cada semana, el sistema pide proactivamente peso/medidas — no depende de que el
   usuario se acuerde solo.
3. El chat puede **leer y modificar** datos reales del perfil (peso objetivo, macros,
   preferencias de comida, nivel de actividad) — no es un chatbot decorativo que solo
   da consejos genéricos sin memoria del usuario.
4. Las sugerencias de comida son cosas que de verdad existen en una casa peruana (arroz,
   menestras, pollo, papa, camote, aji de gallina, etc.), no "100g de quinoa orgánica y
   salmón salvaje".

## 4. Alcance

### Dentro de v0 (MVP web)
- Onboarding con brief inicial (datos antropométricos + metas + gustos/no-gustos)
- Elegir modalidad: gym / casa / mixto
- Entrenamientos: módulo propio inspirado en el modelo de Lyftr (librería de ejercicios,
  programas, gym mode) — con un flag `equipment` para poder filtrar rutinas "solo casa /
  sin equipo"
- Check-in semanal: recordatorio in-app + email para actualizar peso, medidas, % grasa
- Motor de recomendación nutricional: calcula TDEE/macros con fórmula estándar, los
  traduce a un plan de comidas usando una base curada de comida peruana
- Chat con IA (Claude vía API) con **tool use** sobre los datos reales del usuario
  (lee perfil/progreso, propone y aplica cambios a metas/macros con confirmación)
- Dashboard de progreso (peso, medidas, adherencia a macros) — reusa charts de Lyftr

### Fuera de v0 (explícitamente, para no inflar el scope)
- Mobile app (Lyftr ya tiene base en React Native, pero no se toca hasta validar la web)
- Multi-usuario / cuentas públicas / registro abierto
- Escaneo de barcode de comida (Lyftr lo tiene para genérico; comida peruana casera no
  trae barcode — se vuelve relevante solo si se añade registro de empaquetados)
- Notificaciones push nativas (el recordatorio semanal es in-app + email, no push)
- Planes de comida generados libremente por LLM sin curación — ver §5.3, riesgo de
  alucinación en cifras nutricionales

## 5. Funcionalidades

### 5.1 Onboarding / Brief inicial

Un flujo de unos 5–7 pasos, no un formulario largo de una sola pantalla:

| Paso | Qué pide | Por qué |
|---|---|---|
| 1. Datos básicos | edad, sexo biológico (para fórmula TDEE), talla, peso actual | base del cálculo calórico |
| 2. Composición corporal (opcional) | % grasa (si lo sabe, o "no sé") | afecta recomendación de déficit/superávit, no bloquea si no lo sabe |
| 3. Actividad | nivel de actividad diaria (sedentario → muy activo), días de entreno/semana | multiplicador TDEE |
| 4. Modalidad de entreno | gym / casa / mixto, equipo disponible si es casa (mancuernas, bandas, nada) | filtra el catálogo de ejercicios |
| 5. Meta | bajar grasa / mantener / subir masa / rendimiento | define déficit/superávit objetivo |
| 6. Gustos y restricciones | comidas que le gustan, las que odia, alergias/intolerancias, restricciones (vegetariano, sin lactosa, etc.) | input directo al motor de comida peruana |
| 7. Resumen + ajuste manual | muestra el plan calculado, permite tocar el chat ya aquí: "esto no me cuadra, ajústalo" | engancha el chat desde el día 1 en vez de dejarlo escondido en un menú |

Salida del onboarding: fila en `user_profile` + fila inicial en `nutrition_targets` +
primera conversación de chat con un mensaje de la IA resumiendo el plan ("en base a lo
que me dijiste, tu meta es X kcal, Y g de proteína...").

### 5.2 Entrenamiento (inspirado en Lyftr, código propio)

- Mismo modelo de datos que Lyftr porque está bien resuelto: `Workout → WorkoutExercise[]
  → Set[]` y `Program → ProgramExercise[] → ProgramSet[]`.
- `exercises.equipment` y `exercises.location` (`gym` | `home` | `both`) desde el
  diseño inicial de la tabla, para poder filtrar el catálogo por "solo tengo mancuernas
  en casa" — en Lyftr esto no existe, es nuevo aquí.
- Catálogo de ejercicios: no hace falta inventar 800 ejercicios desde cero — igual que
  Lyftr, se puede seedear desde una base abierta de ejercicios (ej. free-exercise-db) y
  luego anotar `equipment`/`location` encima.
  (ver backend/stores/exercise.go de Lyftr como referencia, no como código a copiar)
- Gym Mode (pantalla completa, un ejercicio a la vez, rest timer) se reconstruye porque
  es sencillo y así no se arrastra nada del resto del codebase de Lyftr.
- El programa sugerido en el onboarding se genera según modalidad + meta (editable
  después). **Implementado en la Fase 6** — ver abajo. Igual que con las calorías (§5.3),
  es un motor de reglas el que decide sets/reps/split, nunca la IA inventando un peso en
  kg: eso depende de la fuerza real de la persona, que la app no conoce hasta que registra
  series de verdad.

### 5.8 Amigos y metas conjuntas

- Pensado para una app de pocas cuentas (hoy 2, la pareja), pero con visibilidad explícita
  en vez de automática: **solicitud por email + aceptar** antes de que cualquier dato se
  comparta — decisión explícita del usuario, nunca "toda cuenta ve a toda cuenta" aunque
  hoy solo haya dos personas.
- Lo que un amigo aceptado puede ver del otro (los tres, decidido explícitamente): plan y
  meta actual (kcal/macros/objetivo), progreso de peso/check-ins recientes, historial de
  entrenamientos (esta semana y total).
- **Meta conjunta con seguimiento real**, no solo texto: tipo `workouts_per_week` o
  `weight_target` (o `custom` sin métrica automática), con una descripción libre siempre
  visible y, si aplica, un valor objetivo y fecha límite. El progreso de cada lado se
  calcula en caliente a partir de sus propios `workouts`/`body_measurements` — nunca se
  guarda un número aparte que pueda desalinearse de la realidad.

### 5.3 Nutrición — el módulo nuevo real

**Decisión de arquitectura clave: la IA no inventa valores nutricionales.** Un LLM
alucina cifras de macros con confianza — eso es inaceptable en una app que calcula
déficits calóricos. En cambio:

1. **Base de comida peruana curada, con macros de fuente verificada** (no generada por
   IA): tabla `peruvian_foods` con ~150–300 alimentos/platos comunes (arroz, lentejas,
   pollo a la plancha, lomo saltado, causa, quinoa, camote, palta, etc.). Macros sacados
   de una base nutricional pública (ej. tablas de composición de alimentos del
   INS/CENAN-Perú si están disponibles, o USDA FoodData Central como respaldo para lo
   genérico) y luego **filtrados/adaptados** a porciones y nombres de plato tal como se
   comen en una casa peruana (ej. "1 plato de lomo saltado" en vez de "100g de carne +
   100g de papa" por separado). Es más trabajo de integración que escribir una tabla a
   mano, pero evita el riesgo de un número de macros inventado o mal recordado.
2. **Motor de reglas** calcula TDEE (Mifflin-St Jeor) → calorías objetivo según meta →
   split de macros (proteína por kg de peso, resto grasa/carbos) → arma combinaciones de
   la tabla de comida peruana que cuadren con el objetivo del día, respetando lo que el
   usuario dijo que le gusta/no le gusta/no puede comer.
3. **La IA (Claude) opera sobre el resultado del motor, no calcula ella misma las
   cifras**: explica el plan, lo ajusta reemplazando ítems dentro de la misma tabla
   curada ("no te gusta el pescado, cámbialo por pollo o huevo"), y puede tocar los
   *objetivos* (subir proteína, bajar déficit) — pero siempre a través de una función
   (tool call) que actualiza `nutrition_targets`, nunca escribiendo un número suelto en
   el chat que no queda guardado en ningún lado.

Esto es lo mismo que la regla de Lyftr en `CLAUDE.md` de "single source of truth, un solo
lugar escribe cada dato" — aquí el equivalente es: el motor de reglas es la única fuente
de verdad de macros/calorías; el chat solo la lee y la invoca.

### 5.4 Check-in semanal

- Cron / job semanal (o chequeo simple "¿han pasado 7 días desde el último check-in?" en
  cada login, suficiente para 2 usuarios — no hace falta infraestructura de scheduler) que:
  - Marca `needs_checkin = true` en el perfil
  - Dispara un email (o solo banner in-app para v0, email es v0.1 si se complica el envío)
- Pantalla de check-in: peso, medidas opcionales (cintura, cadera, brazo), % grasa si lo
  mide, cómo se sintió la semana (campo libre corto) — esto último es contexto que luego
  lee el chat para su siguiente mensaje proactivo ("veo que la semana pasada te sentiste
  muy cansado, ¿bajamos el volumen?").
- Cada check-in es una fila en `body_measurements` (mismo patrón append-only que usa
  Lyftr para peso) — nunca sobrescribe el histórico.
- **Decidido**: v0 es solo in-app (banner/alerta al entrar si ya pasó la semana), sin
  email. Se reconsidera si en la práctica no es suficiente para que se acuerden.

### 5.5 Chat con IA

- Pantalla de chat persistente, historial por usuario (`chat_sessions` / `chat_messages`).
- Backend: endpoint que llama a la API de Claude (Anthropic SDK) con:
  - **Modelo: Claude Haiku** (decidido — presupuesto bajo, ~$5-10/mes). Esto implica
    mandar el contexto **resumido**, no el historial completo del chat cada vez: un
    resumen corto del perfil/metas/últimos check-ins generado una vez y actualizado al
    cambiar algo, en vez de recalcularlo o mandar todo el historial crudo en cada
    mensaje. Si con el uso real se siente limitado, subir a Sonnet queda como palanca
    simple (es un solo parámetro del cliente).
  - **System prompt** con el perfil actual, metas, últimos check-ins, racha de
    entrenamientos — contexto real, no solo el texto del usuario.
  - **Tool use** definido en el backend: `propose_nutrition_target_change`,
    `propose_swap_meal_item`, `log_preference` (marcar que algo gusta/no gusta),
    `get_progress_summary`. La IA decide cuándo llamarlas; el backend las ejecuta como
    **propuesta pendiente de confirmación** (nunca aplica el cambio directo — ver regla
    de autonomía abajo).
  - **Decidido: autonomía cero sin confirmación explícita.** Todo cambio de meta/macro
    que la IA proponga queda en estado `pending` hasta que el usuario lo aprueba con un
    tap en el chat ("Sí, aplícalo" / "No, déjalo así"). Esto es el único punto de
    escritura real sobre `nutrition_targets` desde el chat — más auditable y más fácil
    de confiar que una IA que ajusta solo y avisa después.
- Pieza 100% nueva — no existe nada parecido en Lyftr.

### 5.6 Progreso / Dashboard

- Reusa el patrón de charts de Lyftr (Recharts): peso, tendencia de macros vs objetivo,
  adherencia a entrenamientos.
- Añade: línea de tiempo de ajustes hechos por el chat (auditable — "el 3 de marzo la IA
  bajó tu objetivo calórico de 2200 a 2000 porque dijiste que no estabas bajando de peso").

## 6. Modelo de datos (adiciones sobre el esquema de Lyftr)

```
user_profile
  user_id, age, sex, height_cm, body_fat_pct (nullable),
  activity_level, goal (cut|maintain|bulk|performance),
  training_location (gym|home|both), equipment (json: lo que tiene en casa)

food_preferences
  user_id, food_id (FK peruvian_foods), preference (like|dislike|allergy|restriction)

peruvian_foods
  id, name, category, macros_per_serving (kcal, protein, carbs, fat), serving_desc,
  common_in_household (bool), tags (json: vegetariano, sin_lactosa, etc.)

nutrition_targets
  user_id, calories, protein_g, carbs_g, fat_g, updated_at, updated_by (user|ai),
  reason (texto corto — por qué cambió, para el audit trail del §5.6)

pending_changes
  user_id, proposed_by (ai), change_type (nutrition_target|meal_swap), payload (json),
  reason (texto generado por la IA explicando el por qué), status (pending|approved|
  rejected), created_at, resolved_at — toda propuesta de la IA pasa por aquí antes de
  tocar nutrition_targets o meal_plan_days; nunca se escribe directo

meal_plan_days
  user_id, date, meals (json: lista de food_id + porciones) — generado por el motor,
  editable por el chat

body_measurements
  user_id, logged_on, weight, waist, hip, arm, body_fat_pct (todos nullable salvo peso)

checkin_schedule
  user_id, last_checkin_at, next_due_at

chat_sessions / chat_messages
  user_id, session_id, role (user|assistant|tool), content, tool_calls (json), created_at

exercises (alteración sobre tabla existente de Lyftr)
  + equipment (json), + location (gym|home|both)
```

## 7. Arquitectura técnica

- **Decidido: proyecto nuevo, repo propio — no un fork de Lyftr.** Se adopta la misma
  *familia* de stack porque está probada y es liviana para self-hosted (Go+Gin+SQLite
  atrás, React+TS+Tailwind+Vite adelante), pero el código se escribe desde cero. Lyftr
  se usa como referencia de patrones (cómo estructuran migraciones, auth JWT, el
  response helper `utils.OK`/`utils.BadRequest`, etc.) — se lee su código para no
  repetir errores ya resueltos, no se copia/pega.
- Migraciones append-only desde el día 1 (mismo principio que Lyftr, aplicado a un
  esquema propio): nunca modificar una migración ya aplicada, siempre agregar la
  siguiente.
- **Nuevo servicio**: cliente de Anthropic (Claude) en el backend — nunca se llama a la
  API de Claude desde el frontend (la API key no puede vivir en el cliente web).
- **Costo de API**: Haiku (decidido en §5.5), contexto resumido — con 2 usuarios el
  volumen de mensajes es bajo, el riesgo real de costo es mandar demasiado contexto por
  mensaje, no la cantidad de mensajes.
- **Despliegue**: mismo patrón que Lyftr — Docker Compose, nginx + backend, un volumen
  para el SQLite. Puede vivir en el mismo servidor si ya tienen uno corriendo Lyftr,
  como un stack de Compose separado (puertos/volúmenes distintos).
- **Reminders semanales**: chequeo "lazy" en cada request autenticado (¿pasaron 7 días
  desde el último check-in?) en vez de un cron real — decidido junto con "solo in-app"
  en §5.4, evita montar infraestructura de jobs para 2 usuarios.

## 8. Requerimientos no funcionales

- **Privacidad**: datos de salud de 2 personas en un servidor propio — sin analytics de
  terceros, sin trackers. Mismo espíritu "self-hosted, tus datos" que Lyftr.
- **Sin cuentas públicas en v0** — registro cerrado/manual (modo `first-user` o `closed`
  de Lyftr sirve de referencia directa).
- **Responsive mobile-first** — igual que Lyftr, 90% de uso real va a ser desde el celular
  aunque sea "web app".
- **Disponibilidad**: no crítica — es una app de 2 usuarios, downtime ocasional es
  aceptable en v0, no hace falta HA.

## 9. Decisiones tomadas (antes preguntas abiertas)

| # | Pregunta | Decisión |
|---|---|---|
| 1 | Presupuesto API de Claude | **Bajo (~$5-10/mes)** → modelo Haiku, contexto resumido (ver §5.5, §7) |
| 2 | Cómo armar la base de comida peruana | **Partir de una base nutricional pública y filtrar/adaptar** a platos y porciones peruanas reales (ver §5.3) |
| 3 | Autonomía de la IA sobre metas | **Confirmación explícita siempre** — toda propuesta pasa por `pending_changes` (ver §5.5, §6) |
| 4 | Email vs. in-app para el check-in semanal | **Solo in-app en v0** — sin proveedor de email por ahora (ver §5.4, §7) |
| 5 | Fork de Lyftr vs. proyecto nuevo | **Proyecto nuevo, repo propio** — Lyftr se usa solo como referencia de patrones, no se copia código (ver §7) |

### Riesgos que quedan abiertos (no son decisiones, son cosas a validar en el camino)

- Con base de datos nutricional pública + filtrado manual, el ritmo de carga de
  `peruvian_foods` depende de cuánto tiempo le dediquen — puede ser el cuello de botella
  real de la Fase 2, más que el código.
- Haiku con contexto resumido puede sentirse "genérico" en algunas respuestas comparado
  con Sonnet — si pasa, la mitigación ya está prevista (subir de modelo es un parámetro).
- Al no partir del código de Lyftr, auth/Docker/migraciones hay que construirlos de cero
  en la Fase 0 — es más trabajo inicial a cambio de un repo propio y más simple.

## 10. Roadmap (fases)

| Fase | Contenido | Estado |
|---|---|---|
| **0 — Setup del proyecto nuevo** | Scaffold backend (Go+Gin+SQLite) y frontend (React+Vite+Tailwind) desde cero, auth JWT básico, Docker Compose local — usando Lyftr solo como referencia de patrones | ✅ Hecho |
| **1 — Perfil y onboarding** | Tablas nuevas (`user_profile`, `body_measurements`, `nutrition_targets`), onboarding conversacional tipo chat, cálculo de TDEE/macros (Mifflin-St Jeor) | ✅ Hecho |
| **2 — Base de comida peruana + motor de plan** | Tabla curada (31 alimentos, macros de referencia estándar), algoritmo de armado de plan diario respetando gustos/restricciones | ✅ Hecho |
| **3 — Chat con IA** | Integración Claude, onboarding por tool-use, historial persistente | ✅ Hecho |
| **4 — Check-in semanal + dashboard de ajustes** | Recordatorio, pantalla de check-in, timeline de cambios | ✅ Hecho |
| **3/4 extra — Ajuste de metas vía chat con confirmación** | `propose_nutrition_target_change`, tabla `pending_changes`, aprobar/rechazar desde el chat | ✅ Hecho |
| **5 — Entrenamiento (Gym Mode) + rediseño visual** | Catálogo de ejercicios, registro de sets, Gym Mode con filtro casa/gym; rediseño completo de UI/UX | ✅ Hecho |
| **6 — Programa de entrenamiento + Amigos** | Rutina generada por reglas (sets/reps según meta), solicitudes de amistad, perfil compartido, metas conjuntas con seguimiento | ✅ Hecho |
| **7 — Diario de comidas + rebrand "Sami" + paleta premium** | Registrar lo que realmente comiste (no solo el plan sugerido), renombre CoachIA → Sami, paleta crema/dorado | ✅ Hecho |
| **8 — Pulido final mobile-web** | Verificar todo a 390px con el nuevo diseño | ⏳ Siguiente |

### Qué quedó implementado en la Fase 1

- **Backend**: `backend/nutrition/calculator.go` calcula calorías y macros (BMR Mifflin-St
  Jeor × multiplicador de actividad × ajuste por meta); `POST /api/v1/onboarding` guarda
  perfil + primer peso + metas en una sola llamada; `GET /api/v1/profile` y
  `GET /api/v1/nutrition/targets` devuelven 404 hasta que el usuario hace el onboarding.
- **Frontend**: página `/onboarding` (datos básicos, actividad, meta, dónde entrena,
  gustos/restricciones como texto libre); el Dashboard redirige ahí automáticamente si el
  usuario no tiene perfil, y si ya lo tiene muestra el plan (kcal + macros).
- **Onboarding conversacional**: primero se reemplazó el formulario por un chat guionado
  (botones, pasos en el cliente) y después, al adelantar la Fase 3, por el chat real con
  Claude — ver abajo. El guionado ya no existe en el código.

### Qué quedó implementado en la Fase 2

- **`backend/db/migrations.go` (migración 3)**: tabla `peruvian_foods` con 31 alimentos
  base (proteínas, carbohidratos, legumbres, vegetales, frutas, grasas, lácteos) y sus
  macros — valores de referencia nutricional estándar por ración típica peruana, no
  platos compuestos (ver §5.3: evita que un plato tipo "lomo saltado" tenga cifras
  inventadas; se construye desde los ingredientes base).
- **`backend/nutrition/mealplan.go`**: algoritmo que arma 3 comidas (desayuno/almuerzo/
  cena) escalando porciones para acercarse a las metas de calorías/macros del día,
  filtrando por `dislikes` (coincidencia de texto) y `restrictions` (vegetariano/vegano/
  sin lactosa vía tags) y priorizando lo que el usuario marcó en `likes`. Es heurístico,
  no un optimizador — los totales mostrados siempre se calculan de lo que realmente se
  eligió, nunca son una cifra aparte sin relación con los ítems de la lista.
- **`GET /api/v1/nutrition/meal-plan`**: devuelve el plan del día; el Dashboard ya lo
  muestra (comidas + items + kcal) en vez de solo el objetivo numérico.
- **Pendiente real para después**: la tabla de comida sigue chica (31 ítems) y el
  filtrado de `dislikes` es por substring, no por ingrediente normalizado — crecerá con
  uso real, como ya estaba anotado en los riesgos abiertos (§9).

### Qué quedó implementado en la Fase 3 (adelantada — chat real con Claude)

Se adelantó completa: el pedido explícito fue "que el chat sea un chat verdadero, no un
formulario con botones" — así que el onboarding guionado del punto anterior se reemplazó
por **IA real**, no solo por texto libre con reglas.

- **`backend/ai/claude.go`**: cliente HTTP mínimo (sin SDK externo) para la Messages API
  de Anthropic. Modelo `claude-haiku-5-5`, igual que se decidió en §9 por presupuesto.
- **`chat_messages` (migración 4)**: un solo hilo por usuario (no "conversaciones"
  plegables — de sobra para 2 personas). Cada fila guarda el array de content blocks tal
  cual lo manda/devuelve la API, así el historial se puede reproducir byte a byte en la
  siguiente llamada, incluyendo turnos de `tool_use`/`tool_result`.
- **Flujo de onboarding por tool-use**: mientras el usuario no tiene perfil, el system
  prompt le pide a Claude ir conociéndolo naturalmente y llamar a la herramienta
  `complete_onboarding` recién cuando ya tiene los datos obligatorios. El backend ejecuta
  esa herramienta a través del mismo `applyOnboarding()` que usa el endpoint REST directo
  (`controllers/onboarding.go`) — un solo lugar que escribe el perfil, nunca dos
  implementaciones que puedan desalinearse.
- **Modo general**: una vez que el perfil existe, el system prompt cambia — el coach ya
  conoce el plan actual y puede conversar, pero **todavía no tiene una herramienta para
  ajustar metas** (eso es la autonomía-con-confirmación de §5.5/§9, que sigue pendiente).
  Si se lo piden, el propio prompt le dice que lo explique con calidez en vez de inventar
  un cambio que no se guarda en ningún lado.
- **Frontend**: `web/src/pages/Chat.tsx` — chat de verdad, un solo input de texto libre,
  sin botones de opción múltiple. `GET /api/v1/chat/history` repone la conversación al
  recargar; el saludo inicial es puramente del cliente (no cuenta como turno real, evita
  gastar una llamada a la API solo para decir "hola").
- **Requiere `ANTHROPIC_API_KEY`** — igual que `JWT_SECRET`, el arranque falla sin ella
  (`main.go`). Está en `.env.example` (raíz y `backend/`); hay que pegarla en el `.env`
  real antes de levantar el stack.
- **Pendiente real para después**: una sola herramienta (`complete_onboarding`); agregar
  `propose_nutrition_target_change` (con confirmación explícita, como ya decidimos) es lo
  que de verdad vuelve este chat en "tu coach ajustándote el plan" y no solo "tu coach que
  te conoció una vez".
- **Dos bugs reales de "extended thinking" encontrados y corregidos probando el flujo
  completo de punta a punta (no solo con tests unitarios)**:
  1. Los bloques `thinking` que devuelve Claude antes de llamar a una herramienta traen un
     campo `thinking` que puede ser `""` — y la API exige que esa clave esté presente al
     reenviar el bloque en el siguiente turno, incluso vacía. Un `omitempty` general la
     borraba y la API rechazaba todo con `thinking.thinking: Field required`. Se arregló
     con un `MarshalJSON` a medida en `ContentBlock` (`backend/ai/claude.go`) que emite
     exactamente los campos que corresponden a cada tipo de bloque.
  2. La firma de un bloque `thinking` está atada criptográficamente al `system` prompt +
     lista de `tools` exactos que estaban activos cuando Claude lo generó. Como el modo
     onboarding y el modo general usan system prompts y tools distintos, reproducir un
     bloque `thinking` guardado del turno de onboarding en un turno posterior (ya en modo
     general) rompía con `Invalid signature in thinking block. The block is bound to a
     different conversation.` Se arregló con `stripThinkingBlocks()` en
     `backend/controllers/chat.go`: al cargar el historial persistido se descartan los
     bloques `thinking`/`redacted_thinking` (solo hacen falta dentro del mismo round-trip
     tool_use → tool_result armado en la misma request, nunca después). Ambos bugs tienen
     tests de regresión (`backend/ai/claude_test.go`, `backend/controllers/chat_test.go`) y
     se confirmaron en vivo con una conversación de onboarding completa + mensajes
     post-onboarding vía curl contra el stack real en Docker.

### Qué quedó implementado en la Fase 4 (check-in semanal + dashboard de ajustes)

- **`backend/checkin/checkin.go`**: lógica pura (sin DB) que decide si un check-in está
  vencido — `IsDue(lastLoggedOn, now)` compara contra 7 días, igual a lo decidido en §5.4
  ("chequeo simple en vez de un cron real"). Es su propio paquete, como `nutrition`, para
  poder testearlo sin tocar la base de datos.
- **`GET /api/v1/checkin/status`**: devuelve `needs_checkin`, la fecha del último registro
  y los días transcurridos, leyendo el último `body_measurements` del usuario — no hace
  falta la tabla `checkin_schedule` del modelo original en §6, el mismo historial
  append-only ya tiene el dato.
  `POST /api/v1/checkin`: guarda peso (obligatorio), cintura/cadera/brazo/% grasa
  (opcionales) y un campo libre de cómo se sintió la semana, como una fila más en
  `body_measurements` (la tabla ya tenía esas columnas desde la Fase 1, sin necesitar
  migración nueva).
- **Frontend**: `web/src/pages/Checkin.tsx` — formulario corto; el Dashboard muestra un
  banner si el check-in está vencido, con botón directo a `/checkin`.
- **Dashboard de ajustes**: `GET /api/v1/nutrition/targets/history` devuelve el historial
  completo de `nutrition_targets` (ya era append-only desde la Fase 1). El Dashboard lo
  pinta como una línea de tiempo (fecha, kcal/macros, quién lo cambió y por qué) — por
  ahora solo aparece cuando hay más de un registro, ya que con solo el de onboarding no es
  realmente un "historial" todavía.
- **Chat enterado del check-in**: si el último check-in tiene una nota de cómo se sintió
  la semana, el system prompt del modo general la incluye — así el coach puede reaccionar
  a ella sin que el usuario tenga que repetirla (idea original de §5.4: "veo que la semana
  pasada te sentiste cansado, ¿bajamos el volumen?"). Verificado en vivo: el coach mencionó
  espontáneamente una nota de check-in real en su respuesta.
- **Pendiente real para después**: esto es la mitad "de lectura" de la autonomía con
  confirmación — el coach ve el check-in pero todavía no puede proponer un cambio de meta a
  partir de él. Eso sigue atado a `propose_nutrition_target_change` (§5.5/§9), igual que se
  dejó pendiente en la Fase 3.
- **Incidente aparte, no funcional**: a mitad de esta fase el disco C: volvió a llenarse
  (quedaron 16MB libres) y corrompió el almacenamiento interno de Docker (WSL) —
  contenedores con `exec format error` y errores de `containerd` por archivos de snapshot
  faltantes. Se confirmó que los datos reales de la app no corrieron riesgo (la base SQLite
  vive en `./data`, un bind mount al disco D:, no dentro del volumen de Docker). Se liberó
  espacio (cachés de Chrome/Brave, temporales, caché de compilación de Go) y, con
  confirmación explícita del usuario, se borró el disco de datos de Docker
  (`docker_data.vhdx`) para forzar una reconstrucción limpia — las imágenes se reconstruyen
  solas desde los Dockerfiles, así que no hubo pérdida real.

### Ajuste de metas vía chat con confirmación (cierra el pendiente de las Fases 3 y 4)

El pendiente explícito que quedó anotado en ambas fases — que el coach pueda proponer un
cambio real de calorías/macros, nunca aplicado directo — ya está implementado, siguiendo
exactamente la regla de autonomía cero de §5.5/§9.

- **`pending_changes` (migración 5)**: tabla con `change_type` (hoy solo
  `nutrition_target`, `meal_swap` queda para cuando haga falta), `payload` (json),
  `reason` (texto de la IA) y `status` (`pending`/`approved`/`rejected`) — exactamente el
  diseño de §6. Nada en el camino del chat toca `nutrition_targets` directo; solo
  `PendingChangeController.resolve()` lo hace, y solo cuando `status == "approved"`.
- **Herramienta `propose_nutrition_target_change`**: se ofrece en el modo general del
  chat solo cuando no hay ya una propuesta pendiente (si hay una, el system prompt le dice
  a Claude que no proponga otra y que recuerde al usuario que puede resolver la que ya
  existe). El modelo debe pasar calorías/macros y una razón corta; si cualquiera de esos
  dos campos clave falta, el backend rechaza la propuesta sin guardarla (ver bug más abajo).
- **`GET /api/v1/pending-changes`**, **`POST /api/v1/pending-changes/:id/approve`**,
  **`POST /api/v1/pending-changes/:id/reject`**: approve aplica el cambio
  (`nutrition_targets` con `updated_by: "ai"` y la razón de la IA) y marca la propuesta
  resuelta; reject solo la marca resuelta. Un segundo intento de resolver la misma propuesta
  devuelve 409.
- **Frontend**: `web/src/pages/Chat.tsx` ahora muestra una tarjeta con los macros
  propuestos y botones "Sí, aplícalo" / "No, déjalo así" directo en la conversación —
  sobrevive un refresh de página porque se recarga vía `GET /pending-changes` al montar,
  no solo desde la respuesta del chat.

**Tres bugs reales encontrados probando el flujo de punta a punta (no solo con tests):**

1. **Tool_use sin tool_result rompe la conversación para siempre.** Cuando el modelo
   llamaba a una herramienta con datos incompletos, el código original hacía `continue` y
   nunca mandaba un `tool_result` de vuelta. La API exige que todo `tool_use` tenga su
   `tool_result` en el mensaje inmediatamente siguiente — si no, cualquier request futuro
   que reproduzca ese historial es rechazado con *"tool_use ids were found without
   tool_result blocks"*, dejando la conversación rota sin forma de recuperarla desde la
   app (solo editando la base a mano). Se corrigió extrayendo el manejo de cada tool call a
   `ChatController.handleToolCall()`, donde **todas** las ramas devuelven un `tool_result`
   — incluida la de rechazo — nunca un `continue` silencioso. Regresión cubierta en
   `chat_test.go` (`TestHandleToolCallAlwaysReturnsAToolResult`), que fuerza los cuatro
   casos de entrada inválida y confirma que ninguno se escapa sin resultado.
2. **`maxTokens: 1024` se quedaba corto.** El *extended thinking* de Haiku 5.5 a veces
   consume todo el presupuesto de tokens solo pensando, sin dejar nada para el texto visible
   (una respuesta vacía, "") o cortando la respuesta a media frase. Subido a 2048
   (`backend/ai/claude.go`). Como red de seguridad adicional — porque ni con más presupuesto
   esto se puede garantizar al 100% con un modelo chico — si la respuesta final queda vacía
   se muestra un mensaje genérico en vez de una burbuja en blanco.
3. **El modelo a veces "narra" una propuesta sin llamar a la herramienta.** Claude
   ocasionalmente escribía texto tipo "te propongo subir a 2300 kcal... queda pendiente tu
   confirmación" sin de verdad invocar `propose_nutrition_target_change` — el usuario vería
   un mensaje sobre una propuesta que nunca se guardó. Se reforzó el system prompt para que
   sea explícito: si no llamó a la herramienta en ese turno, no existe ninguna propuesta y
   no debe decir que la hizo. Esto reduce el problema pero, al ser un comportamiento del
   modelo y no del código, no se puede eliminar por completo solo con prompting.
- **Incidente aparte, no funcional**: Docker Desktop volvió a fallar dos veces más durante
  esta fase — un crash de containerd a mitad de un build y, después de reiniciar, el proxy
  de Docker que reenvía el puerto 8080 del host se quedó colgado (los contenedores
  respondían bien por dentro, pero `curl localhost:8080` devolvía "empty reply"). Se
  resolvió reiniciando el contenedor `frontend` para que Docker reconstruyera el
  mapeo de puerto.

### Verificación del cálculo de calorías/macros (pedido explícito: "que la IA de las
### recomendaciones correctas")

La IA no calcula calorías — eso ya estaba resuelto por diseño (§5.3: "la IA no inventa
valores nutricionales", el motor de reglas en `backend/nutrition/calculator.go` es la
única fuente de verdad). Lo que se verificó fue ese motor mismo, probando casos extremos
además de los normales, y se encontraron y corrigieron dos fallas reales:

1. **Sin piso de seguridad.** La fórmula de Mifflin-St Jeor pura, con un "cut" (-20%),
   puede caer muy por debajo de lo seguro para un perfil pequeño/mayor/poco activo — el
   caso de test ya existente (mujer, 60 años, 150cm, 40kg, sedentaria, cut) calculaba
   **841 kcal/día**, un déficit no recomendable sin supervisión médica. Se agregó un piso
   (1200 kcal mujeres / 1500 kcal hombres, guías clínicas comunes) que nunca se cruza.
2. **Los macros no siempre sumaban las calorías reportadas.** Con un peso corporal alto
   relativo a una meta calórica ajustada, la proteína (2g/kg de peso real) más la grasa fija
   (25%) podían superar el total de calorías por sí solas — los carbohidratos se iban a
   cero pero la proteína no se reducía, así que la suma real de macros terminaba por encima
   de las calorías que se le mostraban al usuario. Se corrigió escalando proteína y grasa
   proporcionalmente cuando juntas exceden el presupuesto, en vez de dejar que los
   carbohidratos absorban solo el desborde.

Ambos con tests de regresión nuevos (`TestCalculateNeverGoesBelowSafeFloor`,
`TestCalculateMacrosAlwaysSumToCalories`) además de los que ya existían.

### Qué quedó implementado en la Fase 5 (Entrenamiento + rediseño visual)

**Entrenamiento (§5.2):**

- **`exercises` (migración 6)**: catálogo curado a mano de ~49 ejercicios (no importado de
  una base externa como free-exercise-db, para no depender de una fuente de terceros en el
  arranque) con `muscle_group`, `equipment` (json) y `location` (`gym`/`home`/`both`) — el
  filtro "solo tengo esto en casa" que la Fase 5 pedía explícitamente.
- **`workouts` → `workout_exercises` → `workout_sets`**: mismo modelo relacional de Lyftr
  que pedía el PRD, código propio. Un entrenamiento se construye incrementalmente (se arma
  vacío, se le agregan ejercicios, se registran sets uno por uno) en vez de requerir un plan
  completo por adelantado — así calza con cómo se entrena en la práctica.
- **Endpoints**: `GET /exercises?location=`, `POST /workouts`, `GET /workouts`,
  `GET /workouts/:id`, `POST /workouts/:id/exercises`,
  `POST /workouts/:id/exercises/:weId/sets`, `POST /workouts/:id/complete` — todos
  verifican que el recurso pertenece al usuario autenticado antes de leer o escribir
  (`TestWorkoutFlow` incluye un segundo usuario intentando leer/escribir el entrenamiento
  del primero y confirma 404 en ambos casos).
- **Frontend**: `web/src/pages/Workouts.tsx` (reanudar o empezar, historial) y
  `web/src/pages/GymMode.tsx` (pantalla activa: agregar ejercicios desde un selector
  filtrable por casa/gym, registrar sets, rest timer de 90s, terminar entrenamiento).
- **Pendiente real para después**: el "programa sugerido según modalidad + meta" que
  menciona el PRD no se generó automáticamente — por ahora la persona arma su entrenamiento
  eligiendo del catálogo. Es una extensión natural, no algo que se dejó roto.

**Rediseño de UI/UX (pedido explícito: "se ve muy blanco"):**

- Se usó el skill `ui-ux-pro-max` para elegir sistema de diseño en vez de inventar colores
  a ojo: paleta energía-naranja (`#F97316`) + verde para aprobar/éxito (`#22C55E`, calza
  perfecto con los botones de confirmar/rechazar que ya existían), fondo cálido (`cream`,
  no blanco puro) en vez de `neutral-50`/`bg-white` plano, tipografía Barlow Condensed
  (títulos/números grandes) + Barlow (texto) en vez de la sans-serif genérica de Tailwind.
- Iconos de Lucide (`lucide-react`) en vez de emojis como interfaz (el avatar del chat era
  un 🤖 literal — ahora es un ícono de marca consistente).
- Barra de navegación inferior (`components/BottomNav.tsx`) — patrón mobile-first con
  Inicio/Coach/Entrenar, reemplaza los links de texto subrayado del header.
- Todas las pantallas (Login, Register, Dashboard, Chat, Checkin) + las nuevas de
  entrenamiento reescritas sobre este sistema: tarjetas con sombra suave y anillo sutil en
  vez de bordes planos, macros con iconos y color, tarjetas de comida con ícono por tipo
  (desayuno/almuerzo/cena).
- **No se generaron imágenes ni audio.** Se pidió explícitamente usar ElevenLabs o generar
  recursos — ninguna herramienta de generación de imágenes estaba disponible en la sesión
  (el MCP de Higgsfield, que sí genera imágenes/video, se había desconectado) y ElevenLabs
  no está conectado en absoluto (y de estarlo, es síntesis de voz, no generación de
  imágenes). El rediseño se hizo con sistema de diseño (color/tipografía/espaciado) e
  iconos SVG en vez de fotografía — ningún recurso visual externo generado.

### Qué quedó implementado en la Fase 6 (programa de entrenamiento + Amigos)

**Rutina generada por reglas (pedido explícito del brief inicial — "pesos y repeticiones
en base al objetivo"):**

- **`backend/training/program.go`**: `Generate(catalog, goal)` puro (sin DB, 100%
  testeable) — split clásico de 3 días full-body (piernas+empuje / tirón+hombros / full
  body+brazos), eligiendo ejercicios del catálogo por grupo muscular en round-robin para
  variar entre días cuando hay opciones. El esquema de sets/reps sigue convenciones
  estándar de entrenamiento de fuerza por meta: cut 3x12-15, maintain 3x10-12, bulk 4x6-10,
  performance 4x5-8. **Nunca se prescribe un peso en kg** — depende de la fuerza real de
  cada persona, que no se conoce hasta que registra series de verdad; eso es la siguiente
  extensión natural (sugerir peso según lo que registró la última vez en ese mismo
  ejercicio), no algo que se dejó roto hoy.
- **`programs` → `program_days` → `program_exercises` (migración 7)**: mismo patrón
  append-only; `applyOnboarding()` genera el programa inicial como best-effort (un fallo
  aquí nunca bloquea el guardado del perfil/metas, que es lo crítico) y
  `POST /api/v1/programs/generate` permite regenerarlo cuando cambien el equipo o la meta.
- **Frontend**: `Workouts.tsx` muestra el programa con tabs por día y un botón "Empezar
  día X" que arma el entrenamiento con esos ejercicios ya cargados (en vez de empezar
  vacío); sigue siendo editable libremente desde Gym Mode.

**Amigos y metas conjuntas:**

- **`friendships` y `shared_goals` (migración 8)**: solicitud → aceptar/rechazar antes de
  cualquier visibilidad; una meta conjunta cuelga de una amistad ya aceptada.
- **Endpoints**: `POST/GET /friends/requests`, `POST /friends/requests/:id/accept|reject`,
  `GET /friends`, `GET /friends/:id/profile` (plan, check-ins recientes, entrenamientos de
  la semana y totales), `POST/GET /friends/:id/goals`, `POST /goals/:goalId/complete`.
  Toda ruta que toca una amistad verifica que quien pregunta es realmente uno de los dos
  participantes — un tercero recibe 404, nunca una pista de que la amistad existe
  (`TestFriendRequestAndSharedGoalFlow` prueba esto con un tercer usuario real).
- **Frontend**: `Friends.tsx` (agregar por email, solicitudes entrantes, lista de amigos) y
  `FriendProfile.tsx` (perfil compartido + crear/ver/completar metas conjuntas con el
  progreso de ambos lado a lado).
- Verificado en vivo contra el stack real en Docker: generación de programa al
  onboarding, flujo completo de solicitud → aceptar → ver perfil → crear meta → completar,
  y el bloqueo a un tercer usuario — los tres sin errores en los logs del backend.

### Qué quedó implementado en la Fase 7 (diario de comidas + rebrand "Sami" + paleta premium)

**Diario de comidas (pedido explícito: "poder ingresar lo que has comido para llevar
también un conteo"):**

- **`food_log_entries` (migración 9)**: lo que la persona realmente registró que comió,
  distinto del plan sugerido (`nutrition/mealplan.go`, que sigue existiendo como
  sugerencia secundaria). `food_id` siempre referencia la tabla curada `peruvian_foods` —
  las macros de cada entrada se recalculan en cada lectura (`servings × macros del
  alimento`), nunca se copian ni se guarda un número aparte que pueda desalinearse.
- **Decisión deliberada: sin reconocimiento de comida por foto.** La referencia visual que
  mandaste incluye una cámara que "detecta" el plato y sus macros automáticamente — eso
  significa una IA inventando valores nutricionales de una foto, exactamente lo que
  §5.3 prohíbe desde el diseño original ("la IA no inventa valores nutricionales"). V1
  del diario es búsqueda + selección manual del catálogo curado — correcto siempre, sin
  excepción. Si más adelante se quiere una cámara, el diseño que respeta la regla es que
  la IA *sugiera* a qué ítem del catálogo se parece la foto y la persona *confirme*, nunca
  que la IA calcule macros nuevas.
- **Endpoints**: `GET /foods` (catálogo completo, el frontend filtra por texto en el
  cliente — el catálogo es chico, no hace falta búsqueda en el servidor), `POST /food-log`,
  `GET /food-log?date=` (entradas del día + totales consumidos + meta actual, todo en una
  llamada), `DELETE /food-log/:id` (con verificación de dueño — un segundo usuario nunca
  puede borrar una entrada ajena, probado en `TestFoodLogFlow`).
- **Frontend**: `Home` (antes `Dashboard.tsx`) ahora es principalmente el diario — un
  anillo de progreso (`CalorieRing.tsx`, SVG puro) con calorías restantes/de más, secciones
  por comida (desayuno/almuerzo/cena/snacks) con lo realmente registrado y un botón "+" que
  abre `FoodPicker.tsx` (buscar, elegir porciones, agregar).
- **Ajuste tras feedback**: la sugerencia de Sami empezó colapsada detrás de un toggle en
  Home — el usuario pidió que tuviera **su propia sección** además de un apartado visible
  en Home, no escondida. Ahora es `pages/Diet.tsx` (`/diet`, con su propio tab en
  `BottomNav`) con el detalle completo día por comida, y Home muestra una tarjeta siempre
  visible (resumen de kcal por comida) que lleva ahí — nunca oculta por defecto. La página
  de dieta también deja registrar un ítem sugerido directo al diario de hoy con un toque
  (reusa `POST /food-log`, mismo single-source-of-truth que el resto del diario).

**Rebrand "CoachIA" → "Sami":**

- Nombre elegido con el usuario entre tres opciones quechua (Kori/Nina/Sami) — ver nota al
  inicio del documento. Cambiado en toda la UI (títulos, headers, `index.html`) y en los
  system prompts del chat (`backend/controllers/chat.go`): el coach ahora se presenta como
  "Sami" en primera persona, no como un asistente sin nombre.

**Paleta premium crema/dorado (pedido explícito: "más sofisticada... colores premium"):**

- Reemplaza la paleta naranja/verde de la Fase 5. Dorado real (`yellow-700`/`yellow-800`
  de Tailwind, no `yellow-500`/`600` — esos fallan el mínimo de contraste 4.5:1 con texto
  blanco encima) sobre fondo crema (`--color-cream: #fbf6ec`, más cálido que el
  `neutral-50` original). Tipografía Playfair Display (headers/números grandes) + Inter
  (texto/UI) en vez de Barlow — como son tokens semánticos (`font-display`/`font-sans`)
  definidos una sola vez en `index.css`, cambiar la pareja tipográfica no tocó ni un
  componente.
- Verificado en vivo: catálogo de comida, registro de 2 alimentos con porciones distintas,
  totales del día correctos, protección contra borrar la entrada de otro usuario — todo
  sin errores en los logs del backend.

### Ajuste tras feedback: gramos reales en vez de "× porciones"

El usuario pidió cantidades/porciones o, mejor, gramos exactos en la dieta — "× 2" no dice
nada sin saber el tamaño de la porción base.

- **`peruvian_foods.serving_grams` (migración 10)**: no es un dato nuevo inventado, es el
  mismo gramaje que ya estaba escrito dentro de `serving_desc` desde la migración 3 (ej.
  "1 taza (158g)" → 158) movido a su propia columna numérica para poder calcular con él.
  Única excepción: el huevo no traía gramos en el texto — se usó 50g, el peso estándar de
  referencia pública de un huevo grande, misma fuente que el resto de la tabla.
- **`total_grams` calculado** (`servings × serving_grams`, redondeado) en los tres lugares
  donde se muestra un alimento: `MealItem` (plan sugerido), `FoodLogEntry` (diario) y el
  selector de `FoodPicker.tsx` — ahí el stepper de porciones ahora muestra el gramaje en
  grande como número principal ("225g") en vez de un multiplicador ("1.5×").
- Verificado en vivo: catálogo con `serving_grams` correcto, plan sugerido con
  `total_grams` por ítem, y un registro de 1.5 porciones de pollo (150g de referencia)
  calculando 225g exactos — sin errores en los logs.

---

*v0.1 — actualizado tras completar Fases 0 y 1.*
