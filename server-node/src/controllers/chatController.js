const { ErrNotFound } = require('../errors');
const { ok, badRequest, internalErrorf } = require('../utils/response');
const { textFrom } = require('../ai/claude');
const { applyOnboarding, hasRequiredOnboardingFields } = require('./shared');

const onboardingToolSchema = {
  type: 'object',
  properties: {
    age: { type: 'integer', minimum: 13, maximum: 100 },
    sex: { type: 'string', enum: ['male', 'female'] },
    height_cm: { type: 'number', minimum: 100, maximum: 250 },
    weight_kg: { type: 'number', minimum: 30, maximum: 300 },
    body_fat_pct: { type: 'number', minimum: 3, maximum: 70 },
    activity_level: { type: 'string', enum: ['sedentary', 'light', 'moderate', 'active', 'very_active'] },
    goal: { type: 'string', enum: ['cut', 'maintain', 'bulk', 'performance'] },
    training_location: { type: 'string', enum: ['gym', 'home', 'both'] },
    equipment: { type: 'array', items: { type: 'string' } },
    likes: { type: 'array', items: { type: 'string' } },
    dislikes: { type: 'array', items: { type: 'string' } },
    restrictions: { type: 'array', items: { type: 'string' } },
  },
  required: ['age', 'sex', 'height_cm', 'weight_kg', 'activity_level', 'goal', 'training_location'],
};

const nutritionChangeToolSchema = {
  type: 'object',
  properties: {
    calories: { type: 'integer', minimum: 800, maximum: 6000 },
    protein_g: { type: 'integer', minimum: 0, maximum: 400 },
    carbs_g: { type: 'integer', minimum: 0, maximum: 800 },
    fat_g: { type: 'integer', minimum: 0, maximum: 300 },
    reason: { type: 'string', description: 'Explicación corta, para el usuario, de por qué se propone este cambio.' },
  },
  required: ['calories', 'protein_g', 'carbs_g', 'fat_g', 'reason'],
};

const onboardingSystemPrompt = `Te llamas Sami. Eres el coach personal de nutrición y fitness de esta persona, parte de una app self-hosted para una pareja. Esta es la primera conversación — tu objetivo es conocerla antes de armarle un plan de calorías y macros.

Necesitas estos datos, en algún momento de la charla:
- Edad, sexo biológico (para calcular bien las calorías), talla en cm, peso en kg
- % de grasa corporal (opcional — si no lo sabe, sigue sin problema)
- Nivel de actividad diaria: sedentary, light, moderate, active o very_active
- Meta principal: cut (bajar grasa), maintain (mantenerse), bulk (subir masa) o performance (rendimiento)
- Dónde entrena: gym, home o both — y si es home/both, qué equipo tiene en casa
- Gustos, no-gustos y restricciones de comida (piensa en comida peruana: pollo, arroz, lentejas, papa, etc.)

Sé cálido y cercano, como un coach real en la primera consulta — nunca sueltes todas las preguntas de golpe ni parezca un formulario. Ve conversando naturalmente, de a una o dos cosas por mensaje, y reacciona a lo que te cuenten. Responde siempre en español.

Cuando ya tengas los datos obligatorios (edad, sexo, talla, peso, nivel de actividad, meta, dónde entrena), llama a la herramienta complete_onboarding con todo lo que tengas — los gustos/no-gustos/restricciones pueden ir vacíos si la persona no mencionó nada. No llames la herramienta antes de tener los datos obligatorios.`;

// stripThinkingBlocks drops thinking/redacted_thinking blocks from history
// loaded from storage. Their signature is cryptographically bound to the
// exact system prompt + tools list in effect when Claude produced them —
// onboarding and post-onboarding turns use different system prompts/tools,
// so replaying an old thinking block under the new context gets rejected
// with "Invalid signature in thinking block. The block is bound to a
// different conversation." Thinking only needs to survive the immediate
// tool_use -> tool_result round trip built fresh within a single request;
// it serves no purpose once that round trip is persisted and reloaded later.
function stripThinkingBlocks(messages) {
  const out = [];
  for (const m of messages) {
    const blocks = m.content.filter((b) => b.type !== 'thinking' && b.type !== 'redacted_thinking');
    if (blocks.length === 0) continue;
    out.push({ role: m.role, content: blocks });
  }
  return out;
}

function fallbackReplyFor(toolName) {
  if (toolName === 'propose_nutrition_target_change') {
    return 'Te propongo un cambio de plan — revísalo abajo y confírmalo o recházalo cuando quieras.';
  }
  return '¡Listo! Ya guardé tu perfil y calculé tu plan. Vamos a tu dashboard.';
}

function createChatController(profiles, chats, changes, exercises, programs, aiClient) {
  // handleToolCall processes a single tool_use block, applying whatever
  // store change it implies, and always returns a ToolResult for it — even
  // when the call is rejected as invalid. A tool_use with no following
  // tool_result permanently breaks the next request that replays this
  // history (the Messages API rejects the whole call), so every path here
  // must set one; none of them may skip without producing a result.
  function handleToolCall(userId, block) {
    switch (block.name) {
      case 'complete_onboarding': {
        const onboardingReq = block.input;

        if (!hasRequiredOnboardingFields(onboardingReq)) {
          return {
            toolResult: {
              type: 'tool_result',
              tool_use_id: block.id,
              content: 'Todavía faltan datos obligatorios — no se guardó nada. Sigue preguntando lo que falte antes de volver a llamar a esta herramienta.',
            },
          };
        }

        const { targets } = applyOnboarding(profiles, exercises, programs, userId, onboardingReq);

        return {
          toolResult: {
            type: 'tool_result',
            tool_use_id: block.id,
            content: `Guardado. Plan calculado: ${targets.calories} kcal, ${targets.protein_g}g proteína, ${targets.carbs_g}g carbos, ${targets.fat_g}g grasa.`,
          },
          onboardingCompleted: true,
          targets,
        };
      }

      case 'propose_nutrition_target_change': {
        const proposal = block.input;

        if (!proposal.calories || proposal.calories <= 0 || !proposal.reason) {
          return {
            toolResult: {
              type: 'tool_result',
              tool_use_id: block.id,
              content: 'La propuesta vino incompleta (faltan calorías o la razón) — no se guardó. Vuelve a proponerla con todos los campos.',
            },
          };
        }

        const payload = { calories: proposal.calories, protein_g: proposal.protein_g, carbs_g: proposal.carbs_g, fat_g: proposal.fat_g };
        const change = changes.create(userId, 'nutrition_target', payload, proposal.reason);

        return {
          toolResult: {
            type: 'tool_result',
            tool_use_id: block.id,
            content: 'Propuesta guardada como pendiente. Todavía no está aplicada — el usuario tiene que confirmarla o rechazarla desde el chat.',
          },
          pendingChange: change,
        };
      }

      default:
        return {
          toolResult: { type: 'tool_result', tool_use_id: block.id, content: 'Esta herramienta no existe — ignora esta llamada.' },
        };
    }
  }

  return {
    async sendMessage(req, res) {
      const userId = req.userId;
      const message = req.body ? req.body.message : null;
      if (!message) {
        badRequest(res, 'message is required');
        return;
      }

      let history;
      try {
        history = chats.list(userId);
      } catch (err) {
        internalErrorf(req, res, 'could not load chat history', err);
        return;
      }
      history = stripThinkingBlocks(history);

      const userBlocks = [{ type: 'text', text: message }];
      try {
        chats.append(userId, 'user', userBlocks);
      } catch (err) {
        internalErrorf(req, res, 'could not save your message', err);
        return;
      }
      history.push({ role: 'user', content: userBlocks });

      let profile = null;
      try {
        profile = profiles.getProfile(userId);
      } catch (err) {
        if (err !== ErrNotFound) {
          internalErrorf(req, res, 'could not load profile', err);
          return;
        }
      }

      let system = onboardingSystemPrompt;
      let tools = [];
      if (profile) {
        let targets = { calories: 0, protein_g: 0, carbs_g: 0, fat_g: 0 };
        try {
          targets = profiles.getLatestNutritionTargets(userId);
        } catch {
          // best-effort, matches Go's `targets, _ := ...` (zero value on error)
        }
        system = `Te llamas Sami, el coach personal de nutrición y fitness de esta persona. Ya completó su perfil. Su plan actual es ${targets.calories} kcal, ${targets.protein_g}g proteína, ${targets.carbs_g}g carbos, ${targets.fat_g}g grasa, con meta "${profile.goal}". Resuélvele dudas, dale ánimo y comenta sobre lo que te cuente. Responde siempre en español, cálido y breve (4-5 líneas salvo que la pregunta pida más detalle).`;

        try {
          const latest = profiles.getLatestBodyMeasurement(userId);
          if (latest.note) {
            system += ` En su check-in del ${latest.logged_on} contó cómo se sintió esa semana: "${latest.note}". Tenlo en cuenta si es relevante para lo que te esté preguntando — no lo repitas a menos que venga al caso.`;
          }
        } catch {
          // no measurements yet — fine, nothing to mention
        }

        let pendingChanges;
        try {
          pendingChanges = changes.listPending(userId);
        } catch (err) {
          internalErrorf(req, res, 'could not load pending changes', err);
          return;
        }
        if (pendingChanges.length > 0) {
          system += ' Ya existe una propuesta de cambio de plan esperando que el usuario la confirme o la rechace — no propongas otra hasta que esa se resuelva. Si te pregunta por ella, recuérdale que puede confirmarla o rechazarla desde el chat.';
        } else {
          system += ' Tienes la herramienta propose_nutrition_target_change para proponer un cambio de calorías/macros cuando el progreso real lo justifique (por ejemplo, check-ins que muestran que no avanza o que el hambre es mucha) — no la uses solo porque te lo pidan sin motivo real. Nunca digas que ya cambiaste el plan ni menciones un número nuevo de calorías/macros como si fuera una propuesta ya hecha SIN haber llamado antes a la herramienta en este mismo turno — si no la llamaste, no existe ninguna propuesta todavía, así que no inventes que "queda pendiente". Cuando sí decidas proponer algo, llama primero a la herramienta y después explica el por qué en tu texto.';
          tools = [
            {
              name: 'propose_nutrition_target_change',
              description: 'Propone un cambio a las metas de calorías/macros del usuario. Queda pendiente de confirmación — nunca se aplica directo.',
              input_schema: nutritionChangeToolSchema,
            },
          ];
        }
      } else {
        tools = [
          {
            name: 'complete_onboarding',
            description: 'Guarda el perfil inicial de la persona y calcula su plan de calorías y macros. Llamar solo cuando ya se tienen los datos obligatorios.',
            input_schema: onboardingToolSchema,
          },
        ];
      }

      let content, stopReason;
      try {
        [content, stopReason] = await aiClient.send(system, history, tools);
      } catch (err) {
        internalErrorf(req, res, 'tu coach no está disponible ahora mismo, inténtalo de nuevo en un momento', err);
        return;
      }
      try {
        chats.append(userId, 'assistant', content);
      } catch (err) {
        internalErrorf(req, res, 'could not save coach reply', err);
        return;
      }

      const resp = { reply: textFrom(content) };

      if (stopReason === 'tool_use') {
        for (const block of content) {
          if (block.type !== 'tool_use') continue;

          let result;
          try {
            result = handleToolCall(userId, block);
          } catch (err) {
            internalErrorf(req, res, "could not process the coach's tool call", err);
            return;
          }
          if (result.onboardingCompleted) {
            resp.onboarding_completed = true;
            resp.targets = result.targets;
          }
          if (result.pendingChange) {
            resp.pending_change = result.pendingChange;
          }

          const toolResultBlocks = [result.toolResult];
          try {
            chats.append(userId, 'user', toolResultBlocks);
          } catch (err) {
            internalErrorf(req, res, 'could not save tool result', err);
            return;
          }

          history.push({ role: 'assistant', content }, { role: 'user', content: toolResultBlocks });

          try {
            const [finalContent, finalStopReason] = await aiClient.send(system, history, tools);
            try {
              chats.append(userId, 'assistant', finalContent);
            } catch (saveErr) {
              console.error('POST /api/v1/chat: could not save follow-up reply:', saveErr);
            }
            let text = textFrom(finalContent);
            if (!text) {
              console.error(`POST /api/v1/chat: follow-up reply had no text block (stop_reason=${finalStopReason})`);
              text = fallbackReplyFor(block.name);
            }
            resp.reply = text;
          } catch (err) {
            console.error('POST /api/v1/chat: follow-up Claude call after tool_result failed:', err);
            resp.reply = fallbackReplyFor(block.name);
          }
        }
      }

      if (!resp.reply) {
        // Extended thinking can burn the whole token budget reasoning
        // before it ever gets to the visible reply — never show the user
        // nothing.
        console.error(`POST /api/v1/chat: empty reply from Claude (stop_reason=${stopReason})`);
        resp.reply = 'Se me cortó la respuesta pensando la pregunta — ¿puedes repetirla más corta, o pedirme que vaya por partes?';
      }

      ok(res, resp);
    },

    getHistory(req, res) {
      let history;
      try {
        history = chats.list(req.userId);
      } catch (err) {
        internalErrorf(req, res, 'could not load chat history', err);
        return;
      }

      const out = [];
      for (const m of history) {
        const text = textFrom(m.content);
        if (!text) continue; // tool_use/tool_result turns have no display text
        out.push({ role: m.role, text });
      }

      ok(res, out);
    },
  };
}

module.exports = { createChatController };
