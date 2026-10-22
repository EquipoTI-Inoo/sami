// Ported from backend/ai/claude.go. JS plain objects don't have Go's
// omitempty-drops-the-empty-"thinking"-key problem, but we still serialize
// each content block to only the fields its type actually has — a thinking
// block's "thinking" key must always be present (even "") or the Messages
// API rejects the whole request with "thinking.thinking: Field required".

const API_URL = 'https://api.anthropic.com/v1/messages';
const ANTHROPIC_VERSION = '2023-06-01';
const REQUEST_TIMEOUT_MS = 30000;
// 1024 was too tight: extended thinking can consume the whole budget on its
// own, leaving zero tokens for the actual reply (an empty bubble in the chat
// UI) or cutting a Spanish response off mid-sentence.
const MAX_TOKENS = 2048;

function serializeBlock(b) {
  switch (b.type) {
    case 'thinking':
      return { type: 'thinking', thinking: b.thinking ?? '', signature: b.signature ?? '' };
    case 'redacted_thinking':
      return { type: 'redacted_thinking', data: b.data ?? '' };
    case 'text':
      return { type: 'text', text: b.text ?? '' };
    case 'tool_use':
      return { type: 'tool_use', id: b.id, name: b.name, input: b.input };
    case 'tool_result':
      return { type: 'tool_result', tool_use_id: b.tool_use_id, content: b.content ?? '' };
    default:
      return b;
  }
}

class Client {
  constructor(apiKey, model) {
    this.apiKey = apiKey;
    this.model = model;
  }

  // Calls the Messages API and returns [contentBlocks, stopReason]. Callers
  // check stopReason === "tool_use" to know whether any block is a tool call
  // worth acting on.
  async send(system, messages, tools) {
    const body = {
      model: this.model,
      max_tokens: MAX_TOKENS,
      system,
      messages: messages.map((m) => ({ role: m.role, content: m.content.map(serializeBlock) })),
      ...(tools && tools.length > 0 ? { tools } : {}),
    };

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
    let res;
    try {
      res = await fetch(API_URL, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'x-api-key': this.apiKey,
          'anthropic-version': ANTHROPIC_VERSION,
        },
        body: JSON.stringify(body),
        signal: controller.signal,
      });
    } catch (err) {
      throw new Error(`calling Claude API: ${err.message}`);
    } finally {
      clearTimeout(timeout);
    }

    const parsed = await res.json();
    if (parsed.error) {
      throw new Error(`Claude API error: ${parsed.error.message}`);
    }
    if (res.status !== 200) {
      throw new Error(`Claude API returned status ${res.status}`);
    }

    return [parsed.content, parsed.stop_reason];
  }
}

function newClient(apiKey, model) {
  return new Client(apiKey, model);
}

// textFrom concatenates every text block — tool_use/tool_result blocks in
// the same message are ignored, which is exactly what a chat transcript wants.
function textFrom(blocks) {
  let out = '';
  for (const b of blocks) {
    if (b.type === 'text') out += b.text;
  }
  return out;
}

module.exports = { newClient, textFrom };
