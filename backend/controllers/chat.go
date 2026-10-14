package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"coachia-backend/ai"
	"coachia-backend/middleware"
	"coachia-backend/models"
	"coachia-backend/stores"
	"coachia-backend/utils"
)

type ChatController struct {
	profiles  *stores.ProfileStore
	chats     *stores.ChatStore
	changes   *stores.PendingChangeStore
	exercises *stores.ExerciseStore
	programs  *stores.ProgramStore
	ai        *ai.Client
}

func NewChatController(profiles *stores.ProfileStore, chats *stores.ChatStore, changes *stores.PendingChangeStore, exercises *stores.ExerciseStore, programs *stores.ProgramStore, aiClient *ai.Client) *ChatController {
	return &ChatController{profiles: profiles, chats: chats, changes: changes, exercises: exercises, programs: programs, ai: aiClient}
}

type chatRequest struct {
	Message string `json:"message" binding:"required"`
}

type chatResponse struct {
	Reply               string                   `json:"reply"`
	OnboardingCompleted bool                     `json:"onboarding_completed"`
	Targets             *models.NutritionTargets `json:"targets,omitempty"`
	PendingChange       *models.PendingChange    `json:"pending_change,omitempty"`
}

var onboardingToolSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"age": {"type": "integer", "minimum": 13, "maximum": 100},
		"sex": {"type": "string", "enum": ["male", "female"]},
		"height_cm": {"type": "number", "minimum": 100, "maximum": 250},
		"weight_kg": {"type": "number", "minimum": 30, "maximum": 300},
		"body_fat_pct": {"type": "number", "minimum": 3, "maximum": 70},
		"activity_level": {"type": "string", "enum": ["sedentary", "light", "moderate", "active", "very_active"]},
		"goal": {"type": "string", "enum": ["cut", "maintain", "bulk", "performance"]},
		"training_location": {"type": "string", "enum": ["gym", "home", "both"]},
		"equipment": {"type": "array", "items": {"type": "string"}},
		"likes": {"type": "array", "items": {"type": "string"}},
		"dislikes": {"type": "array", "items": {"type": "string"}},
		"restrictions": {"type": "array", "items": {"type": "string"}}
	},
	"required": ["age", "sex", "height_cm", "weight_kg", "activity_level", "goal", "training_location"]
}`)

var nutritionChangeToolSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"calories": {"type": "integer", "minimum": 800, "maximum": 6000},
		"protein_g": {"type": "integer", "minimum": 0, "maximum": 400},
		"carbs_g": {"type": "integer", "minimum": 0, "maximum": 800},
		"fat_g": {"type": "integer", "minimum": 0, "maximum": 300},
		"reason": {"type": "string", "description": "Explicación corta, para el usuario, de por qué se propone este cambio."}
	},
	"required": ["calories", "protein_g", "carbs_g", "fat_g", "reason"]
}`)

const onboardingSystemPrompt = `Te llamas Sami. Eres el coach personal de nutrición y fitness de esta persona, parte de una app self-hosted para una pareja. Esta es la primera conversación — tu objetivo es conocerla antes de armarle un plan de calorías y macros.

Necesitas estos datos, en algún momento de la charla:
- Edad, sexo biológico (para calcular bien las calorías), talla en cm, peso en kg
- % de grasa corporal (opcional — si no lo sabe, sigue sin problema)
- Nivel de actividad diaria: sedentary, light, moderate, active o very_active
- Meta principal: cut (bajar grasa), maintain (mantenerse), bulk (subir masa) o performance (rendimiento)
- Dónde entrena: gym, home o both — y si es home/both, qué equipo tiene en casa
- Gustos, no-gustos y restricciones de comida (piensa en comida peruana: pollo, arroz, lentejas, papa, etc.)

Sé cálido y cercano, como un coach real en la primera consulta — nunca sueltes todas las preguntas de golpe ni parezca un formulario. Ve conversando naturalmente, de a una o dos cosas por mensaje, y reacciona a lo que te cuenten. Responde siempre en español.

Cuando ya tengas los datos obligatorios (edad, sexo, talla, peso, nivel de actividad, meta, dónde entrena), llama a la herramienta complete_onboarding con todo lo que tengas — los gustos/no-gustos/restricciones pueden ir vacíos si la persona no mencionó nada. No llames la herramienta antes de tener los datos obligatorios.`

func (cc *ChatController) SendMessage(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "message is required")
		return
	}

	history, err := cc.chats.List(userID)
	if err != nil {
		utils.InternalErrorf(c, "could not load chat history", err)
		return
	}
	history = stripThinkingBlocks(history)

	userBlocks := []ai.ContentBlock{{Type: "text", Text: req.Message}}
	if err := cc.chats.Append(userID, "user", userBlocks); err != nil {
		utils.InternalErrorf(c, "could not save your message", err)
		return
	}
	history = append(history, ai.Message{Role: "user", Content: userBlocks})

	profile, profileErr := cc.profiles.GetProfile(userID)
	hasProfile := false
	switch {
	case profileErr == nil:
		hasProfile = true
	case errors.Is(profileErr, stores.ErrNotFound):
		hasProfile = false
	default:
		utils.InternalErrorf(c, "could not load profile", profileErr)
		return
	}

	system := onboardingSystemPrompt
	var tools []ai.Tool
	if hasProfile {
		targets, _ := cc.profiles.GetLatestNutritionTargets(userID)
		system = fmt.Sprintf(
			"Te llamas Sami, el coach personal de nutrición y fitness de esta persona. Ya completó su perfil. Su plan actual es %d kcal, %dg proteína, %dg carbos, %dg grasa, con meta \"%s\". Resuélvele dudas, dale ánimo y comenta sobre lo que te cuente. Responde siempre en español, cálido y breve (4-5 líneas salvo que la pregunta pida más detalle).",
			targets.Calories, targets.ProteinG, targets.CarbsG, targets.FatG, profile.Goal,
		)
		if latest, err := cc.profiles.GetLatestBodyMeasurement(userID); err == nil && latest.Note != nil && *latest.Note != "" {
			system += fmt.Sprintf(" En su check-in del %s contó cómo se sintió esa semana: \"%s\". Tenlo en cuenta si es relevante para lo que te esté preguntando — no lo repitas a menos que venga al caso.", latest.LoggedOn, *latest.Note)
		}

		pendingChanges, err := cc.changes.ListPending(userID)
		if err != nil {
			utils.InternalErrorf(c, "could not load pending changes", err)
			return
		}
		if len(pendingChanges) > 0 {
			system += " Ya existe una propuesta de cambio de plan esperando que el usuario la confirme o la rechace — no propongas otra hasta que esa se resuelva. Si te pregunta por ella, recuérdale que puede confirmarla o rechazarla desde el chat."
		} else {
			system += " Tienes la herramienta propose_nutrition_target_change para proponer un cambio de calorías/macros cuando el progreso real lo justifique (por ejemplo, check-ins que muestran que no avanza o que el hambre es mucha) — no la uses solo porque te lo pidan sin motivo real. Nunca digas que ya cambiaste el plan ni menciones un número nuevo de calorías/macros como si fuera una propuesta ya hecha SIN haber llamado antes a la herramienta en este mismo turno — si no la llamaste, no existe ninguna propuesta todavía, así que no inventes que \"queda pendiente\". Cuando sí decidas proponer algo, llama primero a la herramienta y después explica el por qué en tu texto."
			tools = []ai.Tool{{
				Name:        "propose_nutrition_target_change",
				Description: "Propone un cambio a las metas de calorías/macros del usuario. Queda pendiente de confirmación — nunca se aplica directo.",
				InputSchema: nutritionChangeToolSchema,
			}}
		}
	} else {
		tools = []ai.Tool{{
			Name:        "complete_onboarding",
			Description: "Guarda el perfil inicial de la persona y calcula su plan de calorías y macros. Llamar solo cuando ya se tienen los datos obligatorios.",
			InputSchema: onboardingToolSchema,
		}}
	}

	ctx := context.Background()
	content, stopReason, err := cc.ai.Send(ctx, system, history, tools)
	if err != nil {
		utils.InternalErrorf(c, "tu coach no está disponible ahora mismo, inténtalo de nuevo en un momento", err)
		return
	}
	if err := cc.chats.Append(userID, "assistant", content); err != nil {
		utils.InternalErrorf(c, "could not save coach reply", err)
		return
	}

	resp := chatResponse{Reply: ai.TextFrom(content)}

	if stopReason == "tool_use" {
		for _, block := range content {
			if block.Type != "tool_use" {
				continue
			}

			result, err := cc.handleToolCall(userID, block)
			if err != nil {
				utils.InternalErrorf(c, "could not process the coach's tool call", err)
				return
			}
			if result.OnboardingCompleted {
				resp.OnboardingCompleted = true
				resp.Targets = result.Targets
			}
			if result.PendingChange != nil {
				resp.PendingChange = result.PendingChange
			}

			toolResultBlocks := []ai.ContentBlock{result.ToolResult}
			if err := cc.chats.Append(userID, "user", toolResultBlocks); err != nil {
				utils.InternalErrorf(c, "could not save tool result", err)
				return
			}

			history = append(history, ai.Message{Role: "assistant", Content: content}, ai.Message{Role: "user", Content: toolResultBlocks})
			finalContent, finalStopReason, err := cc.ai.Send(ctx, system, history, tools)
			switch {
			case err != nil:
				log.Printf("POST /api/v1/chat: follow-up Claude call after tool_result failed: %v", err)
				resp.Reply = fallbackReplyFor(block.Name)
			default:
				if saveErr := cc.chats.Append(userID, "assistant", finalContent); saveErr != nil {
					log.Printf("POST /api/v1/chat: could not save follow-up reply: %v", saveErr)
				}
				text := ai.TextFrom(finalContent)
				if text == "" {
					log.Printf("POST /api/v1/chat: follow-up reply had no text block (stop_reason=%s)", finalStopReason)
					text = fallbackReplyFor(block.Name)
				}
				resp.Reply = text
			}
		}
	}

	if resp.Reply == "" {
		// Extended thinking can burn the whole token budget reasoning before
		// it ever gets to the visible reply (seen live on a "give me a full
		// day's menu" prompt — stop_reason max_tokens with zero text, no
		// error, just an empty bubble). Never show the user nothing.
		log.Printf("POST /api/v1/chat: empty reply from Claude (stop_reason=%s)", stopReason)
		resp.Reply = "Se me cortó la respuesta pensando la pregunta — ¿puedes repetirla más corta, o pedirme que vaya por partes?"
	}

	utils.OK(c, resp)
}

type toolCallResult struct {
	ToolResult          ai.ContentBlock
	OnboardingCompleted bool
	Targets             *models.NutritionTargets
	PendingChange       *models.PendingChange
}

// handleToolCall processes a single tool_use block, applying whatever store
// change it implies, and always returns a ToolResult for it — even when the
// call is rejected as invalid. A tool_use with no following tool_result
// permanently breaks the next request that replays this history (the
// Messages API rejects the whole call), so every path here must set one;
// none of them may skip without producing a result.
func (cc *ChatController) handleToolCall(userID int64, block ai.ContentBlock) (toolCallResult, error) {
	switch block.Name {
	case "complete_onboarding":
		var onboardingReq models.OnboardingRequest
		if err := json.Unmarshal(block.Input, &onboardingReq); err != nil {
			return toolCallResult{}, err
		}

		if !hasRequiredOnboardingFields(onboardingReq) {
			return toolCallResult{ToolResult: ai.ContentBlock{
				Type:      "tool_result",
				ToolUseID: block.ID,
				Content:   "Todavía faltan datos obligatorios — no se guardó nada. Sigue preguntando lo que falte antes de volver a llamar a esta herramienta.",
			}}, nil
		}

		_, targets, err := applyOnboarding(cc.profiles, cc.exercises, cc.programs, userID, onboardingReq)
		if err != nil {
			return toolCallResult{}, err
		}

		return toolCallResult{
			ToolResult: ai.ContentBlock{
				Type:      "tool_result",
				ToolUseID: block.ID,
				Content:   fmt.Sprintf("Guardado. Plan calculado: %d kcal, %dg proteína, %dg carbos, %dg grasa.", targets.Calories, targets.ProteinG, targets.CarbsG, targets.FatG),
			},
			OnboardingCompleted: true,
			Targets:             &targets,
		}, nil

	case "propose_nutrition_target_change":
		var proposal struct {
			models.NutritionTargetChangePayload
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal(block.Input, &proposal); err != nil {
			return toolCallResult{}, err
		}

		if proposal.Calories <= 0 || proposal.Reason == "" {
			return toolCallResult{ToolResult: ai.ContentBlock{
				Type:      "tool_result",
				ToolUseID: block.ID,
				Content:   "La propuesta vino incompleta (faltan calorías o la razón) — no se guardó. Vuelve a proponerla con todos los campos.",
			}}, nil
		}

		payloadJSON, err := json.Marshal(proposal.NutritionTargetChangePayload)
		if err != nil {
			return toolCallResult{}, err
		}
		change, err := cc.changes.Create(userID, "nutrition_target", payloadJSON, proposal.Reason)
		if err != nil {
			return toolCallResult{}, err
		}

		return toolCallResult{
			ToolResult: ai.ContentBlock{
				Type:      "tool_result",
				ToolUseID: block.ID,
				Content:   "Propuesta guardada como pendiente. Todavía no está aplicada — el usuario tiene que confirmarla o rechazarla desde el chat.",
			},
			PendingChange: &change,
		}, nil

	default:
		return toolCallResult{ToolResult: ai.ContentBlock{
			Type:      "tool_result",
			ToolUseID: block.ID,
			Content:   "Esta herramienta no existe — ignora esta llamada.",
		}}, nil
	}
}

func fallbackReplyFor(toolName string) string {
	if toolName == "propose_nutrition_target_change" {
		return "Te propongo un cambio de plan — revísalo abajo y confírmalo o recházalo cuando quieras."
	}
	return "¡Listo! Ya guardé tu perfil y calculé tu plan. Vamos a tu dashboard."
}

// stripThinkingBlocks drops thinking/redacted_thinking blocks from history
// loaded from storage. Their signature is cryptographically bound to the
// exact system prompt + tools list in effect when Claude produced them —
// onboarding and post-onboarding turns use different system prompts/tools,
// so replaying an old thinking block under the new context gets rejected
// with "Invalid signature in thinking block. The block is bound to a
// different conversation." Thinking only needs to survive the immediate
// tool_use -> tool_result round trip built fresh within a single request;
// it serves no purpose once that round trip is persisted and reloaded later.
func stripThinkingBlocks(messages []ai.Message) []ai.Message {
	out := make([]ai.Message, 0, len(messages))
	for _, m := range messages {
		blocks := make([]ai.ContentBlock, 0, len(m.Content))
		for _, b := range m.Content {
			if b.Type == "thinking" || b.Type == "redacted_thinking" {
				continue
			}
			blocks = append(blocks, b)
		}
		if len(blocks) == 0 {
			continue
		}
		out = append(out, ai.Message{Role: m.Role, Content: blocks})
	}
	return out
}

func hasRequiredOnboardingFields(req models.OnboardingRequest) bool {
	return req.Age > 0 && req.Sex != "" && req.HeightCm > 0 && req.WeightKg > 0 &&
		req.ActivityLevel != "" && req.Goal != "" && req.TrainingLocation != ""
}

type chatHistoryMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

func (cc *ChatController) GetHistory(c *gin.Context) {
	userID := c.GetInt64(middleware.UserIDKey)

	history, err := cc.chats.List(userID)
	if err != nil {
		utils.InternalErrorf(c, "could not load chat history", err)
		return
	}

	out := []chatHistoryMessage{}
	for _, m := range history {
		text := ai.TextFrom(m.Content)
		if text == "" {
			continue // tool_use/tool_result turns have no display text
		}
		out = append(out, chatHistoryMessage{Role: m.Role, Text: text})
	}

	utils.OK(c, out)
}
