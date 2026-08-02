# Chancery

> **chancery** *(n.)* — the office in which documents are drafted and issued.

Turns a directory of Markdown into HTTP endpoints — the path is the route, the frontmatter picks the model, the body is the prompt.

What separates this from a server that merely holds prompts is that the file also chooses the model behind it. A `.md` file is the complete definition of an endpoint: its route is the path, its model is the frontmatter, its behaviour is the Markdown body, and its per-request settings sit beside them. Nothing about that endpoint lives anywhere else.

The nearest familiar thing is a static site generator, and the resemblance is mechanical rather than rhetorical: a file's path is its URL down to the `index.md` rule, frontmatter configures the page, `shared/` holds partials pulled in by name, and `validate` is the build — a broken include fails it, and so does a fragment no agent includes. The difference is that the output is a live endpoint, so the build runs at boot instead of ahead of time.

Requests and responses are [`openai-responses`](https://platform.openai.com/docs/api-reference/responses). Chancery writes `model`, `instructions`, and whatever sampling fields the frontmatter carries onto the body it received, `POST`s the result to `{RESPONSES_BASE_URL}/responses`, and relays the event stream back untouched. It never parses the message array and never decodes an event.

## Configuration directory

Everything an agent is lives outside the binary, in a directory named by `--config`. `./config` in this repository is a working one, and every command below runs against it.

```text
config/
├── models.yaml
├── summarize.md
├── research/
│   ├── index.md
│   └── method.md
└── shared/
    └── voice.md
```

That directory serves two routes:

| file | route |
|---|---|
| `summarize.md` | `POST /summarize` |
| `research/index.md` | `POST /research` |

A file's path minus `.md` is its route. `index.md` takes the route of its directory, so `research/index.md` answers `/research` rather than `/research/index`. Nesting is free: `deep/analysis/filter.md` answers `/deep/analysis/filter`.

Two directory names are reserved and hold no routes:

- `shared/` — fragments any agent can include by name.
- `tools/` — reserved. Answers no route; holds prompts a request pulls in by naming a tool.

A Markdown file that is not an agent and is not included by one is an error, so a fragment that stopped being used is reported rather than sitting there:

```text
✗ research/notes.md: orphaned Markdown file has no frontmatter and is not included by an agent
```

### Fragments

A line that is nothing but a bracketed `.md` filename is an include:

```markdown
You are a research assistant.

[method.md]
```

The name resolves against the agent's own directory first, then `shared/`. `research/index.md` gets `research/method.md`; anything not found beside the agent comes from `shared/`. An include that resolves to neither fails validation.

### Tool prompts

A fragment is pulled in by the agent file. A tool prompt is pulled in by the request, which is what `tools/` is for: how to use a tool is worth saying only to a caller that can call it.

The filename carries the condition. Everything after the last dot names the tool the prompt belongs to:

```text
tools/
├── preamble.md                    always included
├── search/semantic.search.md      included when a tool named `search` is offered
└── shell/grep.run_shell.md        included when a tool named `run_shell` is offered
```

A file whose name has no dot before `.md` is unconditional. Directories are walked, so the grouping is yours to choose and means nothing to the loader. What is selected is appended to `instructions`, behind the agent's own prompt.

The name is read from the `tools` array of the request — a sibling of `input`, never inside it — and nothing else about a tool is looked at. A built-in tool the format identifies by type alone carries no name and asks for no prompt. The array itself travels to the backend byte for byte.

## Agent frontmatter

```markdown
---
description: "condenses a document into three sentences"
model: fast
---
You summarize documents in exactly three sentences.

[voice.md]
```

The body becomes `instructions`. Every other field is a body field or the alias that supplies one:

| field | body position |
|---|---|
| `model` | an alias defined in `models.yaml` |
| `description` | none — it is what `list` prints |
| `max_tokens` | `max_output_tokens` |
| `reasoning_effort` | `reasoning.effort` |
| `reasoning_summary` | `reasoning.summary` |
| `verbosity` | `text.verbosity` |
| `service_tier` | `service_tier` |

An agent's setting beats the alias's.

> [!NOTE]
> `instructions` is prepended, never replaced. A caller sending its own gets the agent's prompt in front of it, separated by a blank line — overwriting would silently discard something the caller wrote, and refusing would make an ordinary Responses body an error.

### Named models

An agent can offer the same prompt on several models. `models` replaces `model`, and `default` names the one the bare route uses:

```markdown
---
description: "answers questions about a corpus"
models:
  quick:
    model: fast
  deep:
    model: smart-prio
default: quick
---
You are a research assistant.

[method.md]
```

That is three routes: `POST /research` and `POST /research.quick` reach `fast`, `POST /research.deep` reaches `smart-prio`. Frontmatter set outside `models` applies to every entry; an entry overrides it.

## models.yaml

One flat map. A key is an alias an agent names; underneath it are the body fields that alias runs with.

```yaml
models:
  fast:
    model: openai/gpt-5-mini
    verbosity: low
  smart:
    model: anthropic/claude-opus-4-6
    reasoning_effort: high
  smart-prio:
    extends: smart
    service_tier: priority
```

`model` is the value that travels in the body, prefix included. The prefix belongs to whoever serves the request — a backend fronting several providers reads it, a backend that is one provider is handed a name without one. Chancery never reads it and never checks it; an unresolvable name comes back as an error from the backend.

`extends` inherits the lot and overrides what it names, up to five steps deep. Aliases accept `model`, `extends`, `max_tokens`, `reasoning_effort`, `reasoning_summary`, `verbosity`, `service_tier`, and `prompt` — a filename under `shared/` whose contents go in front of every agent's prompt on that alias.

## Three files, three questions

| file | answers |
|---|---|
| `<agent>.md` | what this agent is — route, model alias, settings, prompt |
| `models.yaml` | which model an alias names, prefix included, and the settings it runs with |
| `dragoman.yaml` | where a prefix points, what it speaks, and which variable holds its key |

The third file is the backend's, and `dragoman.yaml` in this repository is the deployment's copy of it: one entry per prefix, naming that provider's base URL, the dialect it speaks, and the environment variable holding its key.

Placement and credentials belong to the thing that reaches the network; capability and intent belong to the thing that composes the request. Chancery cannot express an endpoint at all — no API key resolves inside this process, from `models.yaml` or anywhere else, and `auth` names a variable rather than holding a key even on the side that does.

## The backend

`RESPONSES_BASE_URL` is the whole coupling. Chancery builds one URL, `{base}/responses`, and writes only fields the format already has, so anything serving `openai-responses` answers it.

[dragoman](https://github.com/mdijkstra-oss/dragoman) is the recommended target: it reaches several providers behind a single Responses surface, which is what makes a prefixed model name mean something. OpenAI's own `/responses` works identically, and so does anything else mirroring that surface. Nothing here names dragoman, checks for it, or depends on it existing.

Two limits are worth knowing before pointing at a provider directly. Positions travel but values do not — a reasoning level one provider accepts another may reject — and one base URL is one target for the whole instance, so every agent shares that target's catalogue and that target's key.

## Quick start

Requirements: Go `1.26.5`, a configuration directory, and something serving `openai-responses`.

### 1. Validate the configuration

```console
$ go run ./cmd/chancery --config ./config validate
✓ config valid (0 warnings)
```

Nothing here reaches the network. `validate` checks the config it owns — aliases resolve, includes exist, nothing is orphaned — and stays offline. A broken directory reports every problem at once:

```console
$ go run ./cmd/chancery --config ./broken validate
✗ research/notes.md: orphaned Markdown file has no frontmatter and is not included by an agent
✗ summarize.md: unknown model alias "quick"
Error: config invalid (2 errors · 0 warnings)
```

### 2. Inspect the routes

```console
$ go run ./cmd/chancery --config ./config list
PATH                MODEL                      REASONING
research                                       
  .deep             anthropic/claude-opus-4-6  high
  .quick (default)  openai/gpt-5-mini          
summarize           openai/gpt-5-mini          
2 agents · 3 models
```

`list --json` prints the same thing as a document, descriptions included:

```console
$ go run ./cmd/chancery --config ./config list --json
{
  "agents": [
    {
      "path": "research",
      "description": "answers questions about a corpus",
      "models": [
        {
          "name": "deep",
          "model": "anthropic/claude-opus-4-6",
          "reasoning_effort": "high"
        },
        {
          "name": "quick",
          "model": "openai/gpt-5-mini",
          "default": true
        }
      ]
    },
    {
      "path": "summarize",
      "description": "condenses a document into three sentences",
      "model": "openai/gpt-5-mini"
    }
  ],
  "summary": {
    "agents": 2,
    "models": 3
  }
}
```

### 3. Call an agent from the terminal

```console
$ RESPONSES_BASE_URL=http://localhost:8080 \
    go run ./cmd/chancery --config ./config call summarize --input "When did Rome fall?"
Rome fell in 476.
```

`call` reads stdin when `--input` is omitted, and `--input @file` reads a file. It renders the stream as text and exits non-zero when the backend reports a failure.

### 4. Start the server

```console
$ RESPONSES_BASE_URL=http://localhost:8080 \
    go run ./cmd/chancery --config ./config serve
{"time":"2026-08-01T00:05:16.047414+02:00","level":"WARN","msg":"auth disabled — all requests accepted","environment":"development"}
{"time":"2026-08-01T00:05:16.047498+02:00","level":"INFO","msg":"config loaded","environment":"development","component":"startup","data":{"agents":2}}
{"time":"2026-08-01T00:05:16.047644+02:00","level":"INFO","msg":"server starting","environment":"development","component":"startup","data":{"port":"8081"}}
```

```console
$ curl -N -X POST http://localhost:8081/summarize \
    -H 'Content-Type: application/json' \
    -d '{"input":[{"type":"message","role":"user","content":"When did Rome fall?"}],"stream":true}'
event: response.created
data: {"type": "response.created"}

event: response.output_text.delta
data: {"type": "response.output_text.delta", "delta": "Rome "}

event: response.output_text.delta
data: {"type": "response.output_text.delta", "delta": "fell in 476."}

event: response.completed
data: {"type": "response.completed", "response": {"usage": {"input_tokens": 41, "output_tokens": 7}}}
```

Nothing in that request is chancery's own dialect — it is the JSON a stock Responses client already writes, which is what makes the seam a public format rather than a shape a caller has to learn.

## CLI

Every command takes `--config`.

| command | does |
|---|---|
| `validate` | Load the configuration and report every diagnostic. Exits non-zero on any error. |
| `list` | Print agents, routes and resolved models. `--json` prints a document. |
| `call <agent-path>` | Send one turn to an agent and render the stream as text. `--input TEXT`, `--input @FILE`, or stdin. |
| `serve` | Serve every agent over HTTP. |

`call` and `serve` need `RESPONSES_BASE_URL`; `validate` and `list` do not.

## HTTP API

| route | body | response |
|---|---|---|
| `GET /health` | — | `ok` |
| `POST /<agent-path>` | an `openai-responses` request | the backend's event stream, relayed per event |
| `POST /<agent-path>.<model>` | the same, on a named model | the same |

Two query parameters override a body field for one request, for callers that cannot edit the body they send:

- `?tool_choice=` → `tool_choice`
- `?reasoning_summary=` → `reasoning.summary`

Every other key of the request survives whatever it holds. Messages, tools and response formats are raw JSON on the way through — chancery reads none of them.

Bodies are capped at 10 MB. An unknown route is `404`, an undecodable body is `400`, and a backend that never answered is `503`.

### Outbound identity

The forwarded request carries `X-Request-ID`, `X-Agent`, and `X-Subject` — the JWT subject where authentication is on — plus whatever `LOG_REQUEST_HEADERS` names. All of it is a log label at the far end. No credential travels: the caller's `Authorization` header is consumed by the middleware here and never forwarded.

## Runtime environment

| variable | default | meaning |
|---|---|---|
| `RESPONSES_BASE_URL` | none | Required. Scheme, host and port of the backend. Missing, and `serve` refuses at boot rather than failing per request. |
| `RESPONSES_AUTH_TOKEN` | empty | Optional bearer token on every outbound request, for whatever fronts the backend. |
| `PORT` | `8081` | Listen port. |
| `CORS_ORIGINS` | empty | Comma-separated allowed origins. Empty denies every cross-origin request. |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error`. |
| `ENV` | `development` | Recorded on every log line. |
| `SHUTDOWN_TIMEOUT` | `60s` | Grace period for in-flight requests on `SIGTERM`. |
| `LOG_REQUEST_HEADERS` | `X-Session-ID,X-Project-ID` | Caller headers to log and forward. At most sixteen, each canonicalizing to `X-*`, none containing a credential-like term. |

Logs are JSON on stdout. A request line carries the endpoint, the resolved model, the request ID, and the allowlisted headers:

```json
{"time":"2026-08-01T00:02:22.944642+02:00","level":"INFO","msg":"request forwarded","environment":"development","component":"chat","data":{"endpoint":"summarize","model":"openai/gpt-5-mini"},"request_id":"5b3e6b1ee9613282","headers":{"x-session-id":"s-42"},"user":""}
```

### Authentication

JWT validation is off unless a key source is configured, and `serve` says so at boot.

| variable | meaning |
|---|---|
| `AUTH_JWT_JWKS_URL` | JWKS endpoint. Mutually exclusive with the file below. |
| `AUTH_JWT_PUBLIC_KEY_FILE` | PEM public key on disk. |
| `AUTH_JWT_ISSUER` | Required once either source is set. |
| `AUTH_JWT_AUDIENCE` | Required once either source is set. |
| `AUTH_JWT_ALGORITHMS` | Required once either source is set. Asymmetric algorithms only. |

`/health` is always open; every agent route is behind the middleware. A rejected request is `401` with `WWW-Authenticate: Bearer` and a reason in the logs, never in the response.

## Deployment

Two containers under one compose file: dragoman holds every provider key and publishes no host port, chancery publishes `8081` and reaches it by service name.

```sh
git clone https://github.com/mdijkstra-oss/chancery && cd chancery
cp .env.example .env       # one provider key
docker compose up
```

The two binaries live in two repositories and a reader should not have to know that: Docker takes a git URL as a build context, so `compose.yaml` fetches dragoman itself. Both images are static Go binaries on `scratch` — a few MB each, seconds to build. Once images are published the `build:` keys become `image:` references and nothing compiles on the reader's machine at all.

Four files carry the deployment:

- `compose.yaml` — the two services and the network between them.
- `Dockerfile` — chancery's image.
- `.env.example` — the provider keys, all of them on the dragoman service.
- `dragoman.yaml` — the service table, mounted read-only. `--config` replaces dragoman's embedded table wholesale rather than merging, which makes this file exactly the one to read when asking what a deployment can reach.

`config/` and `dragoman.yaml` mount read-only, so editing a prompt file and restarting is enough; nothing is baked into an image.

> [!IMPORTANT]
> Publishing dragoman's port would hand every provider key to anyone who can reach the host. It performs no client authentication by design, and that is safe only because it is unreachable from outside the compose network.

Two more details fail quietly if missed, and `compose.yaml` states both at the line that carries them. dragoman must bind `0.0.0.0` inside its container, since its default is loopback and the symptom of getting it wrong is a connection refused that reads like a wrong hostname. And chancery gates on a dragoman healthcheck rather than on `depends_on` alone, which waits for start and not for listening — without the gate the first requests after `up` fail against a process that has not finished binding.

> [!NOTE]
> Both images are `scratch`: one binary, no shell, no `wget`. A healthcheck naming any command would name one the image cannot execute, so a one-shot `probe` service copies a static busybox into a volume that dragoman mounts read-only. It runs once and exits.

## Development

```sh
make validate            # validate a configuration directory
make list                # list its routes
make start               # serve it, freeing port 8081 first
make dev                 # the same under watchexec
make build               # bin/chancery
make test                # go test ./...
make test-race           # go test -race ./...
make vet fmt-check lint  # go vet, gofmt -l, golangci-lint
make cover               # coverage profile and per-function report
```

`CONFIG` defaults to `./config` and every target takes it: `make start CONFIG=~/prompts`. Environment for a local run comes from `.env.local` — not `.env`, which holds the compose stack's provider keys and has no business in this process.

`make dev` needs [`watchexec`](https://github.com/watchexec/watchexec). Everything else is Go and the module's own dependencies.

## Repository layout

```text
cmd/chancery/            Entry point
internal/prompts/            Markdown discovery, frontmatter, aliases, fragments, validation
internal/responses/          Body composition, the backend client, stream relay
internal/handlers/http/      Routes, middleware, the agent handler
internal/server/             Listener, signal handling, graceful shutdown
internal/cli/                Commands and terminal rendering
internal/auth/               JWT validation
internal/config/             Environment configuration
internal/ratelimit/          Retry and per-model cooldown
internal/fn/                 Generic slice helpers
internal/logging/            Context-carried log attributes
internal/bootstrap/          Logger construction
```

`internal/prompts` is roughly half the non-test code in the repository, which is the shape to expect: composing a request is a handful of fields, and turning a directory into a route table is the work.

## Operational notes

- A `429` from the backend costs three attempts at most — the first and two retries — honouring `Retry-After`, with a per-model cooldown shared across requests. Cooldowns are process-local and are not shared between replicas.
- A backend that accepts a connection and then goes silent for 90 seconds ends the stream rather than holding the caller open.
- Usage counts arrive on `response.completed` where the format puts them; chancery relays them and does not read them.
