# External configuration

`hermes-logos` ships without agents, prompts, providers, or model names. Every command requires an external config directory:

```bash
hermes-logos --config /path/to/config serve
hermes-logos --config /path/to/config validate
hermes-logos --config /path/to/config list
hermes-logos --config /path/to/config call agent-path --input "text"
```

The binary never searches the repository or current working directory for config.

## Directory layout

```text
config/
├── providers.yaml
├── simple-agent.md
├── grouped-agent/
│   ├── index.md
│   └── local-fragment.md
├── shared/
│   └── common.md
├── modes/
│   ├── planning.md
│   └── planning/
│       └── instructions.md
└── tools/
    └── search/
        └── usage.search.md
```

Markdown files outside `shared/`, `modes/`, and `tools/` are either agents with YAML frontmatter or local fragments included by an agent. A frontmatterless file that no agent includes is an orphan and fails validation.

## Agent files

Each agent is one Markdown file. YAML frontmatter contains model settings; the Markdown body is its system prompt.

```markdown
---
description: concise purpose shown by list
model: model-fast
reasoning_effort: low
temperature: 0.2
compact_at: 100000
---
[common.md]

Agent-specific instructions.
```

Supported agent settings are:

- `model`
- `reasoning_effort`
- `reasoning_summary`
- `verbosity`
- `service_tier`
- `legacy_thinking`
- `temperature`
- `seed`
- `auto_cache`
- `cache_ttl`
- `compact_at`
- `dimensions`
- `max_tokens`
- `description`

An empty body is valid and means the prompt comes entirely from caller messages. `validate` reports it as a warning.

### Named models

Use `models` when callers need stable names for multiple model configurations:

```markdown
---
description: analysis with selectable depth
reasoning_summary: concise
models:
  fast:
    model: model-fast
    reasoning_effort: low
  deep:
    model: model-deep
    reasoning_effort: high
default: fast
---
Analyze the supplied material.
```

Top-level settings are shared by every named entry. Entry settings override shared settings.

`default` is required when `models` contains more than one entry. A single entry is implicitly the default. `default` must name an existing entry.

An agent defines either `model` or `models`, never both.

## Routing

Routes mirror paths relative to the config root:

| File | Route |
|---|---|
| `simple-agent.md` | `POST /simple-agent` |
| `grouped-agent/index.md` | `POST /grouped-agent` |
| `grouped-agent/review.md` | `POST /grouped-agent/review` |

Named entries use a suffix:

```text
POST /grouped-agent.deep
```

A bare path uses the configured default. Numeric model query indexes are not supported.

`embeddings.md` remains the explicit `POST /embeddings` endpoint. It uses the same frontmatter parser but retains the dedicated embeddings request and provider implementation.

## Prompt composition

An agent body can include local or shared Markdown:

```markdown
[local-fragment.md]
[common.md]
```

Resolution checks the agent file's directory first, then `shared/`. Literal text between includes remains in place. Includes are resolved when config loads, and missing files fail validation.

### Modes

Top-level files in `modes/` compile to mode prompts keyed by filename. They use the same include syntax.

```text
modes/planning.md → planning
```

Caller messages containing `<!-- prompt: planning -->` are expanded through the existing message pipeline.

### Tool prompts

Files in `tools/` are loaded per request according to the tools supplied by the caller. The segment after the final `.` before `.md` is the required tool name:

```text
usage.search.md → loaded when the caller provides the search tool
```

Files without a tool suffix are always included. Tool schema remains responsible for argument names, types, and call format; tool prompts should only teach tool-specific behavior.

## Providers

All providers and models live in `providers.yaml`:

```yaml
providers:
  provider-a:
    protocol: responses
    base_url: https://provider.example/v1
    api_key_env: PROVIDER_A_API_KEY
    models:
      model-fast:
        name: upstream-model-name
        reasoning_effort: low
      model-priority:
        extends: model-fast
        service_tier: priority
```

Supported protocols are `responses`, `gemini`, `completions`, and `anthropic`. Provider model inheritance, provider translation, streaming, and request conversion retain their existing semantics.

`validate` and `list` never read API-key environment variables. `serve` resolves all configured provider keys at execution time. `call` resolves only the selected agent's provider key.

## CLI

### Serve

```bash
hermes-logos --config /path/to/config serve
```

Loads config, resolves execution credentials, and starts the HTTP server.

External quota integration is optional. Set `QUOTA_RESERVE_URL` and `QUOTA_SETTLE_URL` together to enable it. `QUOTA_AUTH_TOKEN` adds bearer authentication to both calls, and `QUOTA_TIMEOUT` controls their timeout with a default of `2s`. Reserve failures fail closed; settlement failures are logged without replacing a successful provider response.

The reserve endpoint receives a `POST` request containing the request ID, optional authenticated subject, operation, endpoint, provider, resolved model, service tier, reasoning effort, estimated input tokens, and maximum output tokens. Anonymous requests omit `subject`. It returns a 2xx JSON response such as:

```json
{
  "allowed": true,
  "reservation_id": "res-123"
}
```

A denial returns a 2xx response with `allowed: false`; `reason` and `retry_after_seconds` are optional. Allowed responses require a reservation ID.

The settlement endpoint receives a `POST` request containing the reservation ID, an outcome of `completed`, `failed`, or `cancelled`, and normalized usage when available:

```json
{
  "reservation_id": "res-123",
  "outcome": "completed",
  "usage": {
    "input_tokens": 100,
    "cached_input_tokens": 20,
    "cache_write_tokens": 10,
    "output_tokens": 50,
    "reasoning_tokens": 5,
    "total_tokens": 150
  }
}
```

The settlement endpoint may return any 2xx response. Pricing, plans, balances, quota policy, and persistence remain external to `hermes-logos`.

### Validate

```bash
hermes-logos --config /path/to/config validate
```

Reports malformed YAML, invalid provider/model references, missing required frontmatter, invalid named defaults, missing includes, orphaned local Markdown, and empty-body warnings. Errors produce a nonzero exit code.

### List

```bash
hermes-logos --config /path/to/config list
hermes-logos --config /path/to/config list --json
```

Shows every agent path, upstream model, and reasoning effort. Named entries are indented and the default is marked. The human-readable output ends with agent, model, and provider counts.

### Call

```bash
hermes-logos --config /path/to/config call simple-agent --input "text"
hermes-logos --config /path/to/config call grouped-agent.deep --input @request.txt
printf 'text' | hermes-logos --config /path/to/config call simple-agent
```

Chat output streams as text to stdout. Calling `embeddings` writes the provider's embeddings JSON response to stdout.

## Prompt writing boundaries

Three layers must not overlap:

| Layer | Responsibility |
|---|---|
| Tool schema | Argument names, types, and call format |
| Tool prompt | Tool-specific patterns and selection guidance |
| Agent/shared prompt | Domain intent, judgment, and workflow |

Use direct behavioral rules, concise rationale, and semantic sections. Avoid repeating schemas in prose, duplicating guidance across layers, contradictory rules, chain-of-thought examples, and blanket tool-use reminders.
