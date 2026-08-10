# Chancery

> **chancery** *(n.)* — the office in which documents are drafted and issued.

Turns a directory of Markdown into HTTP endpoints — the path is the route, the frontmatter picks the model, the body is the prompt.

Requests and responses are [`openai-responses`](https://platform.openai.com/docs/api-reference/responses). Chancery writes `model`, `instructions`, and whatever sampling fields the frontmatter carries onto the body it received, `POST`s the result to `{RESPONSES_BASE_URL}/responses`, and relays the event stream back untouched.

Every other key survives whatever it holds. Messages, tools and response formats pass through as raw JSON.

The backend is anything serving that format: OpenAI's own `/responses`, a local server, or a router in front of several providers.

## Configuration directory

`--config` names the directory, defaulting to `./config`. The one in this repository is a working example, and the commands below run against it.

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
- `tools/` — prompts a request pulls in by naming a tool.

A Markdown file that is neither an agent nor included by one is an error:

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

A fragment is pulled in by the agent file; a tool prompt is pulled in by the request. A request's `tools` array names the tools it offers, and everything after the last dot in a filename names the tool that file belongs to:

```text
tools/
├── preamble.md                    always included
├── search/semantic.search.md      included when the request offers `search`
└── shell/grep.run_shell.md        included when the request offers `run_shell`
```

A file whose name has no dot before `.md` is unconditional. Directories under `tools/` group files and select nothing. What is selected goes into `instructions`, behind the agent's own prompt. A built-in tool the format identifies by type alone carries no name and pulls in no prompt.

## Agent frontmatter

```markdown
---
description: "condenses a document into three sentences"
model: fast
---
You summarize documents in exactly three sentences.

[voice.md]
```

The body becomes `instructions`. The rest of the frontmatter is how this agent runs:

| field | what it changes |
|---|---|
| `model` * | which alias in `models.yaml` answers the route |
| `description` | nothing at runtime — it is the line `list` prints beside the route |
| `max_tokens` | ceiling on the length of a reply |
| `reasoning_effort` | how hard the model thinks before answering |
| `reasoning_summary` | whether the stream carries an account of that thinking |
| `verbosity` | how much the model says at a given effort |
| `service_tier` | which queue the request joins, where the backend offers a choice |
| `prompt_cache_breakpoints` | whether this model accepts explicit cache breakpoints |

\* Required, and the only one that is. An agent defines exactly one of `model` or the `models` map below; both, or neither, fails validation.

An agent's setting beats the alias's. Values reach the backend as written and are its to interpret; chancery does not check them against the model.

`prompt_cache_breakpoints` is the one field that never reaches the body. Only `false` is sent, as a query parameter on the request, telling whatever serves it that this model refuses breakpoints. A backend that has never heard of the parameter answers as before.

> [!NOTE]
> `instructions` is prepended, never replaced. A caller sending its own gets the agent's prompt in front of it, separated by a blank line.

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

Every entry names a `model`. `default` is required once there is more than one entry; with a single entry it is that entry.

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

`model` travels in the body verbatim, so write whatever the backend expects — `gpt-5-mini` against OpenAI, the prefixed form above against a router. Chancery never parses the name, and one it cannot resolve comes back as a backend error.

`extends` inherits the lot and overrides what it names, up to five steps deep. An alias must end up with a `model`, its own or an inherited one; the rest is optional. Beyond the agent fields above, an alias takes `extends` and `prompt` — a filename under `shared/` whose contents go in front of every agent's prompt on that alias.

## Quick start

Requirements: Go `1.26.5`, a configuration directory, and something serving `openai-responses`.

### 1. Validate the configuration

```console
$ go run ./cmd/chancery validate
✓ config valid (0 warnings)
```

`validate` checks that aliases resolve, includes exist and nothing is orphaned. A broken directory reports every problem at once:

```console
$ go run ./cmd/chancery --config ./broken validate
✗ research/notes.md: orphaned Markdown file has no frontmatter and is not included by an agent
✗ summarize.md: unknown model alias "quick"
Error: config invalid (2 errors · 0 warnings)
```

### 2. Inspect the routes

```console
$ go run ./cmd/chancery list
PATH                MODEL                      REASONING
research                                       
  .deep             anthropic/claude-opus-4-6  high
  .quick (default)  openai/gpt-5-mini          
summarize           openai/gpt-5-mini          
2 agents · 3 models
```

### 3. Call an agent from the terminal

```console
$ RESPONSES_BASE_URL=http://localhost:8080 \
    go run ./cmd/chancery call summarize --input "When did Rome fall?"
Rome fell in 476.
```

`call` reads stdin when `--input` is omitted, and `--input @file` reads a file. It renders the stream as text and exits non-zero when the backend reports a failure.

### 4. Start the server

```console
$ RESPONSES_BASE_URL=http://localhost:8080 go run ./cmd/chancery serve
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

## HTTP API

| route | body | response |
|---|---|---|
| `GET /health` | — | `ok` |
| `POST /<agent-path>` | an `openai-responses` request | the backend's event stream, relayed per event |
| `POST /<agent-path>.<model>` | the same, on a named model | the same |
| `POST /<agent-path>/responses` | the same | the same |
| `POST /<agent-path>.<model>/responses` | the same | the same |

A stock OpenAI SDK appends `/responses` to whatever `base_url` it is given, so the suffixed forms make any agent path a valid `base_url`. The suffix comes off only when the literal path matches no route: an agent whose own path ends in `/responses` keeps it.

Bodies are capped at 10 MB. An unknown route is `404`, an undecodable body is `400`, and a backend that never answered is `503`.

A `429` from the backend costs three attempts at most — the first and two retries — honouring `Retry-After`, with a per-model cooldown shared across requests in this process. A backend that accepts a connection and then goes silent for 90 seconds ends the stream rather than holding the caller open.

Chancery adds `X-Request-ID`, `X-Agent` and `X-Subject` for the backend's record, and passes on everything the caller sent except the headers that describe the request it rebuilt: `Content-Type` and `Content-Length`, because the body is recomposed; `Authorization`, because it carries `RESPONSES_AUTH_TOKEN` rather than the caller's token; `Accept-Encoding` and the hop-by-hop headers, because the connection to the backend is chancery's own. The three identity headers are chancery's too, so a caller sending its own copy of one does not append to the backend's record.

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
| `LOG_REQUEST_HEADERS` | `X-Session-ID,X-Project-ID` | Caller headers to put on every log line for the request. A value over 512 characters is truncated and the line says so. |

Logs are JSON on stdout:

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

`Dockerfile` builds a static Go binary on `scratch` — one binary, no shell, answering its own `/health` through the `healthcheck` subcommand, which is what a container runtime can execute there. Point it at a backend with `RESPONSES_BASE_URL` and mount a configuration directory at `/config`.

`compose.yaml` does that alongside [dragoman](https://github.com/mdijkstra-oss/dragoman), which reaches several providers behind one Responses surface:

```sh
git clone https://github.com/mdijkstra-oss/chancery && cd chancery
cp .env.example .env       # one provider key
docker compose up
```

dragoman holds every provider key and publishes no host port — publishing one would hand those keys to anyone who can reach the host — while chancery publishes `8081` and reaches it by service name. `CHANCERY_PORT` moves the host side of that.

`config/` and `dragoman.yaml` mount read-only, so editing either and restarting is the whole edit cycle. `dragoman.yaml` lists the prefixes a `models.yaml` alias may name, each with its endpoint, the protocol it speaks, and the environment variable holding its key.

Compose builds dragoman from a git URL, so cloning chancery is enough to bring up both services. `DRAGOMAN_REPO` replaces that URL with a local path.

Two details fail quietly if missed. dragoman must bind `0.0.0.0` inside its container, or the other container cannot reach it and the symptom reads as a wrong hostname. Chancery waits on dragoman's healthcheck rather than `depends_on` alone, which waits for start and not for listening. `compose.yaml` states both at the line that carries them.

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

`CONFIG` defaults to `./config`, and the targets that read a configuration take it: `make start CONFIG=~/prompts`. Environment for a local run comes from `.env.local`, not `.env` — that one holds the compose stack's provider keys.

`make dev` needs [`watchexec`](https://github.com/watchexec/watchexec). Everything else is Go and the module's own dependencies.

`internal/prompts` is the largest package, because turning a directory into a route table is most of the work. `internal/responses` composes the body and relays the stream.

## CLI

Every command takes `--config`, defaulting to `./config`.

| command | does |
|---|---|
| `validate` | Load the configuration and report every diagnostic. Exits non-zero on any error. |
| `list` | Print agents, routes and resolved models. `--json` prints the same as a document, descriptions included. |
| `call <agent-path>` | Send one turn to an agent and render the stream as text. `--input TEXT`, `--input @FILE`, or stdin. |
| `serve` | Serve every agent over HTTP. |
| `healthcheck` | Ask a running instance whether it is serving. Exits non-zero when it is not. `--addr` defaults to `127.0.0.1:$PORT`. |

`call` and `serve` need `RESPONSES_BASE_URL`; `validate`, `list` and `healthcheck` do not.

## License

AGPL-3.0. The full text is in [LICENSE](./LICENSE).

AGPL-3.0 is strong copyleft, and its network clause applies where no copy is distributed at all: anyone who runs a modified chancery as a network service must offer the source of that version to the users that service reaches. Read the terms before deploying a modified copy, not after.

## See also

- [dragoman](https://github.com/mdijkstra-oss/dragoman) — one Responses endpoint over several providers, which is what a prefixed model name in `models.yaml` is for. One `RESPONSES_BASE_URL` is one target for the whole instance, so reaching more than one provider's catalogue means putting something like this in front.
