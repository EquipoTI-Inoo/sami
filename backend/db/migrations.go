package db

import (
	"database/sql"
	"fmt"
)

// Append-only: once a migration has shipped, never edit its SQL — add a new
// entry instead. The version number is what's recorded as applied.
var migrations = []struct {
	version int
	sql     string
}{
	{
		version: 1,
		sql: `
			CREATE TABLE users (
				id            INTEGER PRIMARY KEY AUTOINCREMENT,
				email         TEXT NOT NULL UNIQUE,
				password_hash TEXT NOT NULL,
				created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
			);
		`,
	},
	{
		version: 2,
		sql: `
			CREATE TABLE user_profile (
				user_id            INTEGER PRIMARY KEY REFERENCES users(id),
				age                INTEGER NOT NULL,
				sex                TEXT NOT NULL CHECK (sex IN ('male', 'female')),
				height_cm          REAL NOT NULL,
				body_fat_pct       REAL,
				activity_level     TEXT NOT NULL CHECK (activity_level IN ('sedentary', 'light', 'moderate', 'active', 'very_active')),
				goal               TEXT NOT NULL CHECK (goal IN ('cut', 'maintain', 'bulk', 'performance')),
				training_location  TEXT NOT NULL CHECK (training_location IN ('gym', 'home', 'both')),
				equipment          TEXT NOT NULL DEFAULT '[]',
				likes              TEXT NOT NULL DEFAULT '[]',
				dislikes           TEXT NOT NULL DEFAULT '[]',
				restrictions       TEXT NOT NULL DEFAULT '[]',
				created_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
				updated_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
			);

			CREATE TABLE body_measurements (
				id            INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id       INTEGER NOT NULL REFERENCES users(id),
				logged_on     TEXT NOT NULL,
				weight_kg     REAL NOT NULL,
				waist_cm      REAL,
				hip_cm        REAL,
				arm_cm        REAL,
				body_fat_pct  REAL,
				note          TEXT,
				created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
			);

			CREATE INDEX idx_body_measurements_user_date ON body_measurements(user_id, logged_on);

			CREATE TABLE nutrition_targets (
				id          INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id     INTEGER NOT NULL REFERENCES users(id),
				calories    INTEGER NOT NULL,
				protein_g   INTEGER NOT NULL,
				carbs_g     INTEGER NOT NULL,
				fat_g       INTEGER NOT NULL,
				updated_by  TEXT NOT NULL DEFAULT 'user' CHECK (updated_by IN ('user', 'ai')),
				reason      TEXT,
				created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
			);

			CREATE INDEX idx_nutrition_targets_user ON nutrition_targets(user_id, created_at);
		`,
	},
	{
		// Seed macros are standard reference values for the raw/base ingredient,
		// not AI-estimated recipe totals — see CoachIA PRD §5.3 for why composite
		// dishes are deliberately left out of v1 and built from these instead.
		version: 3,
		sql: `
			CREATE TABLE peruvian_foods (
				id            INTEGER PRIMARY KEY AUTOINCREMENT,
				name          TEXT NOT NULL,
				category      TEXT NOT NULL CHECK (category IN ('protein', 'carb', 'legume', 'vegetable', 'fruit', 'fat', 'dairy')),
				serving_desc  TEXT NOT NULL,
				calories      INTEGER NOT NULL,
				protein_g     REAL NOT NULL,
				carbs_g       REAL NOT NULL,
				fat_g         REAL NOT NULL,
				tags          TEXT NOT NULL DEFAULT '[]'
			);

			INSERT INTO peruvian_foods (name, category, serving_desc, calories, protein_g, carbs_g, fat_g, tags) VALUES
				('Pollo (pechuga a la plancha)', 'protein', '150g', 248, 46.5, 0, 5.4, '[]'),
				('Pollo (pierna con piel, al horno)', 'protein', '150g', 324, 39, 0, 16.5, '[]'),
				('Carne de res (lomo)', 'protein', '150g', 375, 39, 0, 22.5, '[]'),
				('Pescado (bonito o jurel, a la plancha)', 'protein', '150g', 210, 39, 0, 4.5, '[]'),
				('Atún en lata (en agua)', 'protein', '1 lata (100g)', 116, 26, 0, 1, '[]'),
				('Huevo', 'protein', '1 unidad', 72, 6, 0.4, 5, '["vegetariano"]'),
				('Queso fresco', 'dairy', '100g', 98, 11, 3, 4, '["vegetariano","con_lactosa"]'),
				('Leche entera', 'dairy', '1 taza (240ml)', 149, 8, 12, 8, '["vegetariano","con_lactosa"]'),
				('Arroz blanco cocido', 'carb', '1 taza (158g)', 205, 4.3, 45, 0.4, '["vegetariano","vegano","sin_lactosa"]'),
				('Papa sancochada', 'carb', '1 papa mediana (150g)', 130, 2.7, 30, 0.2, '["vegetariano","vegano","sin_lactosa"]'),
				('Camote sancochado', 'carb', '150g', 129, 2.4, 30, 0.1, '["vegetariano","vegano","sin_lactosa"]'),
				('Yuca sancochada', 'carb', '150g', 190, 1, 45, 0.3, '["vegetariano","vegano","sin_lactosa"]'),
				('Choclo', 'carb', '1 mazorca mediana (150g)', 123, 4.5, 27, 1.5, '["vegetariano","vegano","sin_lactosa"]'),
				('Pan francés', 'carb', '1 unidad (60g)', 165, 5, 32, 1.5, '["vegetariano","vegano","sin_lactosa"]'),
				('Avena cocida', 'carb', '1 taza (234g)', 166, 6, 28, 3.5, '["vegetariano","vegano","sin_lactosa"]'),
				('Quinua cocida', 'carb', '1 taza (185g)', 222, 8, 39, 3.5, '["vegetariano","vegano","sin_lactosa"]'),
				('Fideos cocidos', 'carb', '1 taza (140g)', 220, 8, 43, 1.3, '["vegetariano","vegano","sin_lactosa"]'),
				('Lentejas cocidas', 'legume', '1 taza (198g)', 230, 18, 40, 0.8, '["vegetariano","vegano","sin_lactosa"]'),
				('Frijoles o pallares cocidos', 'legume', '1 taza (182g)', 245, 15, 44, 1, '["vegetariano","vegano","sin_lactosa"]'),
				('Garbanzos cocidos', 'legume', '1 taza (164g)', 269, 14.5, 45, 4.2, '["vegetariano","vegano","sin_lactosa"]'),
				('Lechuga', 'vegetable', '1 taza (36g)', 5, 0.5, 1, 0, '["vegetariano","vegano","sin_lactosa"]'),
				('Tomate', 'vegetable', '1 unidad mediana (123g)', 22, 1, 4.8, 0.2, '["vegetariano","vegano","sin_lactosa"]'),
				('Zanahoria', 'vegetable', '1 taza (128g)', 52, 1.2, 12, 0.3, '["vegetariano","vegano","sin_lactosa"]'),
				('Brócoli cocido', 'vegetable', '1 taza (156g)', 55, 3.7, 11, 0.6, '["vegetariano","vegano","sin_lactosa"]'),
				('Palta', 'fat', '1/2 unidad (100g)', 160, 2, 8.5, 14.7, '["vegetariano","vegano","sin_lactosa"]'),
				('Plátano', 'fruit', '1 unidad mediana (118g)', 105, 1.3, 27, 0.4, '["vegetariano","vegano","sin_lactosa"]'),
				('Manzana', 'fruit', '1 unidad mediana (182g)', 95, 0.5, 25, 0.3, '["vegetariano","vegano","sin_lactosa"]'),
				('Naranja', 'fruit', '1 unidad mediana (131g)', 62, 1.2, 15, 0.2, '["vegetariano","vegano","sin_lactosa"]'),
				('Papaya', 'fruit', '1 taza (145g)', 62, 0.7, 16, 0.4, '["vegetariano","vegano","sin_lactosa"]'),
				('Aceite vegetal', 'fat', '1 cucharada (14g)', 124, 0, 0, 14, '["vegetariano","vegano","sin_lactosa"]'),
				('Maní', 'fat', '1/4 taza (36g)', 207, 9.4, 7.7, 18, '["vegetariano","vegano","sin_lactosa"]');
		`,
	},
	{
		// One flat thread per user, not multiple named conversations — the right
		// scope for a 2-person personal coach. content is the raw Claude content
		// block array (JSON) so history can be replayed to the API byte-for-byte,
		// including tool_use/tool_result turns.
		version: 4,
		sql: `
			CREATE TABLE chat_messages (
				id         INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id    INTEGER NOT NULL REFERENCES users(id),
				role       TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
				content    TEXT NOT NULL,
				created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
			);

			CREATE INDEX idx_chat_messages_user ON chat_messages(user_id, created_at);
		`,
	},
	{
		// Every AI-proposed change to the plan lands here first — nothing the
		// chat's tool calls do ever writes to nutrition_targets directly. See
		// CoachIA PRD §5.5: "autonomía cero sin confirmación explícita".
		version: 5,
		sql: `
			CREATE TABLE pending_changes (
				id          INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id     INTEGER NOT NULL REFERENCES users(id),
				change_type TEXT NOT NULL CHECK (change_type IN ('nutrition_target')),
				payload     TEXT NOT NULL,
				reason      TEXT NOT NULL,
				status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
				created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
				resolved_at TEXT
			);

			CREATE INDEX idx_pending_changes_user_status ON pending_changes(user_id, status);
		`,
	},
	{
		// Entrenamiento, inspirado en el modelo de datos de Lyftr (Workout →
		// WorkoutExercise[] → Set[]) pero con código propio — ver CoachIA PRD
		// §5.2. `equipment`/`location` en exercises es nuevo frente a Lyftr: es
		// lo que permite filtrar el catálogo por "solo tengo esto en casa".
		// El catálogo es curado a mano (como peruvian_foods en la migración 3),
		// no importado de una base externa, para no depender de una fuente de
		// datos de terceros en el arranque.
		version: 6,
		sql: `
			CREATE TABLE exercises (
				id            INTEGER PRIMARY KEY AUTOINCREMENT,
				name          TEXT NOT NULL,
				muscle_group  TEXT NOT NULL CHECK (muscle_group IN ('piernas', 'pecho', 'espalda', 'hombros', 'brazos', 'core', 'cardio')),
				equipment     TEXT NOT NULL DEFAULT '[]',
				location      TEXT NOT NULL CHECK (location IN ('gym', 'home', 'both'))
			);

			CREATE TABLE workouts (
				id           INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id      INTEGER NOT NULL REFERENCES users(id),
				started_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
				completed_at TEXT,
				notes        TEXT
			);

			CREATE INDEX idx_workouts_user ON workouts(user_id, started_at);

			CREATE TABLE workout_exercises (
				id          INTEGER PRIMARY KEY AUTOINCREMENT,
				workout_id  INTEGER NOT NULL REFERENCES workouts(id),
				exercise_id INTEGER NOT NULL REFERENCES exercises(id),
				order_index INTEGER NOT NULL DEFAULT 0
			);

			CREATE INDEX idx_workout_exercises_workout ON workout_exercises(workout_id, order_index);

			CREATE TABLE workout_sets (
				id                  INTEGER PRIMARY KEY AUTOINCREMENT,
				workout_exercise_id INTEGER NOT NULL REFERENCES workout_exercises(id),
				set_number          INTEGER NOT NULL,
				reps                INTEGER NOT NULL,
				weight_kg           REAL,
				completed_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
			);

			CREATE INDEX idx_workout_sets_exercise ON workout_sets(workout_exercise_id);

			INSERT INTO exercises (name, muscle_group, equipment, location) VALUES
				('Sentadilla con peso corporal', 'piernas', '["bodyweight"]', 'both'),
				('Sentadilla con barra', 'piernas', '["barbell"]', 'gym'),
				('Sentadilla goblet con mancuerna', 'piernas', '["dumbbell"]', 'both'),
				('Zancadas', 'piernas', '["bodyweight"]', 'both'),
				('Zancadas con mancuernas', 'piernas', '["dumbbell"]', 'both'),
				('Sentadilla búlgara', 'piernas', '["dumbbell", "bench"]', 'both'),
				('Peso muerto con barra', 'piernas', '["barbell"]', 'gym'),
				('Peso muerto rumano con mancuernas', 'piernas', '["dumbbell"]', 'both'),
				('Hip thrust', 'piernas', '["bench"]', 'gym'),
				('Puente de glúteo', 'piernas', '["bodyweight"]', 'both'),
				('Prensa de piernas', 'piernas', '["machine"]', 'gym'),
				('Extensión de cuádriceps en máquina', 'piernas', '["machine"]', 'gym'),
				('Curl femoral en máquina', 'piernas', '["machine"]', 'gym'),
				('Elevación de talones (pantorrilla)', 'piernas', '["bodyweight"]', 'both'),
				('Flexiones de pecho', 'pecho', '["bodyweight"]', 'both'),
				('Flexiones diamante', 'pecho', '["bodyweight"]', 'both'),
				('Press de banca con barra', 'pecho', '["barbell", "bench"]', 'gym'),
				('Press de banca con mancuernas', 'pecho', '["dumbbell", "bench"]', 'both'),
				('Aperturas con mancuernas', 'pecho', '["dumbbell", "bench"]', 'both'),
				('Fondos en paralelas', 'pecho', '["dip_bars"]', 'gym'),
				('Press militar con barra', 'hombros', '["barbell"]', 'gym'),
				('Press militar con mancuernas', 'hombros', '["dumbbell"]', 'both'),
				('Elevaciones laterales con mancuerna', 'hombros', '["dumbbell"]', 'both'),
				('Elevaciones frontales con mancuerna', 'hombros', '["dumbbell"]', 'both'),
				('Face pull con banda', 'hombros', '["band"]', 'both'),
				('Pike push-up', 'hombros', '["bodyweight"]', 'both'),
				('Dominadas', 'espalda', '["pullup_bar"]', 'gym'),
				('Remo con barra', 'espalda', '["barbell"]', 'gym'),
				('Remo con mancuerna a una mano', 'espalda', '["dumbbell", "bench"]', 'both'),
				('Jalón al pecho en polea', 'espalda', '["machine"]', 'gym'),
				('Remo en máquina', 'espalda', '["machine"]', 'gym'),
				('Superman (lumbar)', 'espalda', '["bodyweight"]', 'both'),
				('Curl de bíceps con mancuerna', 'brazos', '["dumbbell"]', 'both'),
				('Curl de bíceps con barra', 'brazos', '["barbell"]', 'gym'),
				('Curl de bíceps con banda', 'brazos', '["band"]', 'both'),
				('Extensión de tríceps con mancuerna', 'brazos', '["dumbbell"]', 'both'),
				('Fondos de tríceps en banco', 'brazos', '["bench"]', 'both'),
				('Plancha abdominal', 'core', '["bodyweight"]', 'both'),
				('Crunch abdominal', 'core', '["bodyweight"]', 'both'),
				('Elevación de piernas', 'core', '["bodyweight"]', 'both'),
				('Dead bug', 'core', '["bodyweight"]', 'both'),
				('Russian twist', 'core', '["bodyweight"]', 'both'),
				('Mountain climbers', 'core', '["bodyweight"]', 'both'),
				('Burpees', 'cardio', '["bodyweight"]', 'both'),
				('Jumping jacks', 'cardio', '["bodyweight"]', 'both'),
				('Saltos de cuerda', 'cardio', '["jump_rope"]', 'both'),
				('Correr o caminar', 'cardio', '["none"]', 'both'),
				('Rodillas altas (high knees)', 'cardio', '["bodyweight"]', 'both');
		`,
	},
	{
		// "El programa sugerido en el onboarding se genera según modalidad +
		// meta" (PRD §5.2) — generado por reglas en backend/training, nunca
		// por la IA inventando pesos, igual que nutrition_targets. Append-only
		// como el resto: regenerar crea una fila nueva, no sobrescribe.
		version: 7,
		sql: `
			CREATE TABLE programs (
				id         INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id    INTEGER NOT NULL REFERENCES users(id),
				goal       TEXT NOT NULL,
				location   TEXT NOT NULL,
				created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
			);

			CREATE INDEX idx_programs_user ON programs(user_id, created_at);

			CREATE TABLE program_days (
				id         INTEGER PRIMARY KEY AUTOINCREMENT,
				program_id INTEGER NOT NULL REFERENCES programs(id),
				day_index  INTEGER NOT NULL,
				name       TEXT NOT NULL
			);

			CREATE INDEX idx_program_days_program ON program_days(program_id, day_index);

			CREATE TABLE program_exercises (
				id              INTEGER PRIMARY KEY AUTOINCREMENT,
				program_day_id  INTEGER NOT NULL REFERENCES program_days(id),
				exercise_id     INTEGER NOT NULL REFERENCES exercises(id),
				order_index     INTEGER NOT NULL,
				target_sets     INTEGER NOT NULL,
				target_reps_min INTEGER NOT NULL,
				target_reps_max INTEGER NOT NULL
			);

			CREATE INDEX idx_program_exercises_day ON program_exercises(program_day_id, order_index);
		`,
	},
	{
		// Amigos: solicitud por email + aceptar antes de ver nada (decisión
		// explícita del usuario — nunca visibilidad automática entre cuentas).
		// shared_goals cuelga de una friendship ya aceptada; su progreso se
		// calcula en caliente a partir de workouts/body_measurements, nunca
		// se guarda un número aparte que pueda desalinearse.
		version: 8,
		sql: `
			CREATE TABLE friendships (
				id           INTEGER PRIMARY KEY AUTOINCREMENT,
				requester_id INTEGER NOT NULL REFERENCES users(id),
				recipient_id INTEGER NOT NULL REFERENCES users(id),
				status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected')),
				created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
				resolved_at  TEXT
			);

			CREATE INDEX idx_friendships_requester ON friendships(requester_id, status);
			CREATE INDEX idx_friendships_recipient ON friendships(recipient_id, status);

			CREATE TABLE shared_goals (
				id            INTEGER PRIMARY KEY AUTOINCREMENT,
				friendship_id INTEGER NOT NULL REFERENCES friendships(id),
				created_by    INTEGER NOT NULL REFERENCES users(id),
				goal_type     TEXT NOT NULL CHECK (goal_type IN ('workouts_per_week', 'weight_target', 'custom')),
				description   TEXT NOT NULL,
				target_value  REAL,
				target_date   TEXT,
				status        TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'abandoned')),
				created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
				completed_at  TEXT
			);

			CREATE INDEX idx_shared_goals_friendship ON shared_goals(friendship_id, status);
		`,
	},
	{
		// Diario de comidas: lo que la persona realmente registró que comió,
		// distinto del plan sugerido (nutrition/mealplan.go). food_id siempre
		// referencia la tabla curada peruvian_foods — las macros de cada
		// entrada se recalculan desde ahí en cada lectura (servings × macros
		// del alimento), nunca se copian ni se guarda un número aparte.
		version: 9,
		sql: `
			CREATE TABLE food_log_entries (
				id         INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id    INTEGER NOT NULL REFERENCES users(id),
				food_id    INTEGER NOT NULL REFERENCES peruvian_foods(id),
				meal_type  TEXT NOT NULL CHECK (meal_type IN ('desayuno', 'almuerzo', 'cena', 'snack')),
				servings   REAL NOT NULL DEFAULT 1,
				logged_on  TEXT NOT NULL,
				created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
			);

			CREATE INDEX idx_food_log_user_date ON food_log_entries(user_id, logged_on);
		`,
	},
	{
		// Gramos reales por ración, para mostrar cantidades exactas ("300g")
		// en vez de un multiplicador ambiguo ("× 2") en la dieta y el diario.
		// Los valores vienen del mismo gramaje ya citado en serving_desc desde
		// la migración 3 (ej. "1 taza (158g)" → 158) — no son un número nuevo,
		// solo se extraen a su propia columna numérica. La única excepción es
		// el huevo, que no traía gramos en el texto: 50g es el peso estándar
		// de un huevo grande de referencia pública, igual que el resto de la
		// tabla (ver nota de la migración 3 sobre la fuente de los datos).
		version: 10,
		sql: `
			ALTER TABLE peruvian_foods ADD COLUMN serving_grams REAL;

			UPDATE peruvian_foods SET serving_grams = CASE id
				WHEN 1  THEN 150  -- Pollo (pechuga a la plancha)
				WHEN 2  THEN 150  -- Pollo (pierna con piel, al horno)
				WHEN 3  THEN 150  -- Carne de res (lomo)
				WHEN 4  THEN 150  -- Pescado (bonito o jurel, a la plancha)
				WHEN 5  THEN 100  -- Atún en lata (en agua)
				WHEN 6  THEN 50   -- Huevo
				WHEN 7  THEN 100  -- Queso fresco
				WHEN 8  THEN 240  -- Leche entera
				WHEN 9  THEN 158  -- Arroz blanco cocido
				WHEN 10 THEN 150  -- Papa sancochada
				WHEN 11 THEN 150  -- Camote sancochado
				WHEN 12 THEN 150  -- Yuca sancochada
				WHEN 13 THEN 150  -- Choclo
				WHEN 14 THEN 60   -- Pan francés
				WHEN 15 THEN 234  -- Avena cocida
				WHEN 16 THEN 185  -- Quinua cocida
				WHEN 17 THEN 140  -- Fideos cocidos
				WHEN 18 THEN 198  -- Lentejas cocidas
				WHEN 19 THEN 182  -- Frijoles o pallares cocidos
				WHEN 20 THEN 164  -- Garbanzos cocidos
				WHEN 21 THEN 36   -- Lechuga
				WHEN 22 THEN 123  -- Tomate
				WHEN 23 THEN 128  -- Zanahoria
				WHEN 24 THEN 156  -- Brócoli cocido
				WHEN 25 THEN 100  -- Palta
				WHEN 26 THEN 118  -- Plátano
				WHEN 27 THEN 182  -- Manzana
				WHEN 28 THEN 131  -- Naranja
				WHEN 29 THEN 145  -- Papaya
				WHEN 30 THEN 14   -- Aceite vegetal
				WHEN 31 THEN 36   -- Maní
			END;
		`,
	},
}

func Migrate(conn *sql.DB) error {
	if _, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		);
	`); err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	for _, m := range migrations {
		var applied int
		err := conn.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, m.version).Scan(&applied)
		if err != nil {
			return fmt.Errorf("checking migration %d: %w", m.version, err)
		}
		if applied > 0 {
			continue
		}

		tx, err := conn.Begin()
		if err != nil {
			return fmt.Errorf("starting tx for migration %d: %w", m.version, err)
		}
		if _, err := tx.Exec(m.sql); err != nil {
			tx.Rollback()
			return fmt.Errorf("applying migration %d: %w", m.version, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, m.version); err != nil {
			tx.Rollback()
			return fmt.Errorf("recording migration %d: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing migration %d: %w", m.version, err)
		}
	}

	return nil
}
