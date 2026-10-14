# Sami

Tu coach y nutricionista personal con IA — self-hosted, pensado primero para 2 usuarios.
Calcula tu plan de calorías/macros, arma una dieta con comida peruana real, genera una
rutina de entrenamiento, lleva un diario de lo que comes y entrenas, y lo habla contigo
por chat (Claude) en vez de dejarte solo con formularios y gráficas.

Ver [`PRD.md`](./PRD.md) para el producto completo: visión, decisiones de arquitectura y
qué se implementó en cada fase.

## Stack

- **Backend**: Go + Gin + SQLite (`modernc.org/sqlite`, sin CGO)
- **Frontend**: React + TypeScript + Vite + Tailwind CSS v4
- **IA**: Anthropic Claude (Messages API, llamado directo por HTTP)
- **Deploy**: Docker Compose (dos contenedores: `backend` y `frontend`)

## Correr en local

Requiere Docker y Docker Compose.

```bash
cp .env.example .env
# edita .env: pon un JWT_SECRET real y tu ANTHROPIC_API_KEY (console.anthropic.com)
docker compose up -d --build
```

La app queda en `http://localhost` (puerto 80, configurable con `PORT` en `.env`).

## Variables de entorno (`.env`)

| Variable | Qué es |
|---|---|
| `JWT_SECRET` | String largo y aleatorio para firmar los tokens de sesión. |
| `CORS_ORIGIN` | Origen(es) permitidos, separados por coma (ej. `https://sami.tudominio.com`). |
| `PORT` | Puerto donde el contenedor `frontend` expone la app (default `80`). |
| `BACKEND_ORIGIN` | A dónde le pega el frontend para `/api` — normalmente no hace falta tocarlo (`backend:3000`, nombre del servicio de Docker Compose). |
| `ANTHROPIC_API_KEY` | Requerida — el chat con el coach es el centro de la app, no es opcional. |

## Desplegar en un subdominio

La app ya está pensada para correr detrás de un reverse proxy con TLS:

1. Levanta el stack con `docker compose up -d --build` en el servidor.
2. Pon tu reverse proxy (nginx, Caddy, Traefik, etc.) apuntando al puerto que expone el
   contenedor `frontend` (el de `PORT` en `.env`), con el certificado del subdominio.
3. En `.env`, pon `CORS_ORIGIN` con el subdominio real, por ejemplo:
   ```
   CORS_ORIGIN=https://sami.tudominio.com
   ```
   Si vas a entrar también desde el celular con otra URL (ej. una IP local mientras
   pruebas), se puede poner más de un origen separado por coma.
4. La base de datos SQLite vive en `./data` (bind mount al host, fuera de los
   contenedores) — haz backup de esa carpeta, es el único estado real de la app.

## Estructura

```
backend/   API en Go (Gin + SQLite)
web/       Frontend en React + Vite
data/      SQLite + WAL (no versionado, se crea al levantar el stack)
PRD.md     Producto: visión, decisiones, qué se construyó en cada fase
```
