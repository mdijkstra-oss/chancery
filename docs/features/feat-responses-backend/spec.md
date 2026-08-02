---
slug: responses-backend
type: feat
status: draft
created: 2026-07-31
repos:
  chancery: /Users/matthijn/Documents/dev.nosync/hermes/hermes-logos
  dragoman: /Users/matthijn/Documents/dev.nosync/open-source/dragoman
live_config: /Users/matthijn/Desktop/hermes-logos-config-full-2026-07-17
---

------

# Feature: Any `openai-responses` backend

## Purpose

Chancery does two unrelated things. It compiles a directory of Markdown into addressable
agents — routes, model aliases, composed system prompts, validation — and it translates a
canonical request into four provider wire formats and their streams back again. The second job is
the whole of [dragoman](https://github.com/mdijkstra-oss/dragoman), which does nothing else and has
5,725 lines of tests behind it.

So chancery stops translating. It resolves an agent, writes a handful of fields onto an
`openai-responses` body, forwards it, and relays the event stream back.

**It forwards to whatever serves that format.** Dragoman is the recommended target because it is
the one that speaks four dialects behind a single Responses surface, but OpenAI's own `/responses`
works identically, and so does anything else mirroring it. Nothing in this repository names
dragoman, checks for it, or depends on it existing — a base URL is the entire coupling. That is
the point of the cut, not a side effect of it: the seam is a public format, so the set of things
chancery can talk to grows without chancery changing.

The gain is not line count. It is that each repository has one subject, and the boundary between
them is a body a stock SDK already knows how to write.

## What chancery becomes

A request arrives on an agent's route carrying an `openai-responses` body. Chancery resolves the path
segment to an agent, resolves the agent's model alias to a model name, writes `model` and
`instructions` and whatever sampling fields the frontmatter carries, and `POST`s the result to
`{backend}/responses`. The backend answers with the `openai-responses` event sequence, which chancery
relays to the client and does not otherwise read.

**The request field becomes `input`, matching the format.** It is `messages` today, which made an
almost-Responses body that no stock client could produce. Renaming it means an OpenAI SDK pointed
at an agent route works with nothing in between, which is the plainest possible demonstration that
the seam is a public format.

Everything chancery writes is a field the format already has, so there is no canonical request type
and no adapter. The message array is never parsed — it arrives as JSON and leaves as the same
JSON.

**Relaying an event stream does not require parsing one.** The backend emits the `openai-responses`
event sequence and the client expects that same sequence, so the server path is headers, copy,
flush per event — nothing between the two is read. The one place chancery still decodes events is
`call`, which has to turn a stream into terminal text, and that is a CLI concern rather than a
provider one. It moves accordingly.

What survives is the part that was never a translator:

| package | src | test | fate |
|---|---:|---:|---|
| `internal/prompts` | 1,130 | 505 | kept — this is the product |
| `internal/{auth,config,server,cli,handlers,ratelimit,fn,…}` | ~1,300 | ~1,400 | kept, mostly untouched |
| `internal/providers` | 3,568 | 5,238 | **deleted in full**; the terminal renderer moves to `internal/cli` |
| `internal/protocol` | 242 | 487 | **deleted** |
| `internal/messages` | 178 | 475 | **deleted** |
| `internal/quota` | 174 | 138 | **deleted** |
| `internal/telemetry` | 143 | 162 | **deleted** |
| `internal/tokens` | 19 | 47 | **deleted** |
| `internal/invoke`, `internal/pipeline` | 116 | 122 | **folded into their callers** |

`invoke` and `pipeline` exist to stand between a resolved agent and one of four adapters. With one
backend and no canonical request there is not enough left in either to justify a package boundary,
and a package that only forwards is a name pretending to be a seam.

The `prompts` row is the one to read twice: Markdown discovery, frontmatter validation, model
alias inheritance, fragment includes, orphan detection. Roughly 2,600 non-test lines survive in
total, and that package is nearly half of them.

## What leaves

**The three translating adapters and the canonical request.** With them goes
`google.golang.org/genai` and roughly twenty indirect dependencies — gRPC, OpenCensus, the Google
Cloud auth tree — pulled in to reach one endpoint that dragoman reaches with `net/http`.

**Both in-band markers, and modes with them.** Chancery stops reading `<!-- prompt: … -->` and
`<!-- cache -->`. Modes go entirely: the `modes/` reserved directory, the last-marker-wins
expansion, the removal of earlier markers. Cache breakpoints are a placement concern and dragoman
already owns them per dialect; a caller who wants one sets `prompt_cache_breakpoint` on the
content part, exactly as a caller talking to dragoman directly would. Once neither marker is read,
chancery never inspects the message array at all, which is what deletes `internal/messages` rather
than shrinking it.

**`temperature`.** It names a body field, which is why a caller may still send one and have it
travel untouched. It stops being something an agent file can pin: a reasoning model rejects the
field or ignores it, so an alias fixing it fixes a value most of the catalogue will not take.

**Frontmatter that names no field in the format.** `seed`, `legacy_thinking`, and `cache_ttl` have
no position in an `openai-responses` body and die with the adapters that consumed them, taking
Gemini explicit caching with them — dragoman drops Gemini cache markers deliberately, because both
of that provider's caching routes are stateful and a stateless proxy reaches neither.

Two agents set `seed`, so this costs something: `hyde-generator` and `topic-assigner` lose
deterministic sampling and there is nothing to replace it with. That could not be checked from
this repository — `config/` is empty by design and the live configuration lives outside the
workspace — so it is a done criterion rather than an assumption: `validate` against the real
config directory must report no unknown fields after the strip. Where it does, an agent depends on
something being deleted, and what that agent loses is stated rather than absorbed.

**Usage accounting entirely** — quota reservation and settlement, the call records, and the token
estimator. The quota client goes with its four environment variables and with the `429` and `503`
responses that reservation produced; the call record goes with the endpoint, model, reasoning,
service tier, trigger, duration, and the five token counts it carried; the estimator goes with the
`bytes ÷ 4` heuristic that fed reservation and the embeddings batch guard.

Two reasons beyond the obvious one. Quota reserves and settles against an external service this
repository does not ship, so nobody cloning it can exercise the path at all. And the call record
is the last thing in chancery that reads message content: deriving the trigger means walking the
array backwards for the most recent user message or tool result and resolving that result's call
back to the tool that produced it. Removing it is what makes "chancery never reads a message" a fact
rather than an aspiration — modes and markers were only two thirds of that claim.

The accounting itself is not lost, it relocates — see below. Dragoman already normalises usage
across all four dialects into one shape, and a client wanting the numbers reads them off
`response.completed` where they have always been.

**Embeddings.** Dragoman serves `/responses` and nothing else, so the route has no destination.
Keeping it would mean the chancery container holds one provider key and one direct HTTP path, which
is the entire credential story undone for one endpoint. It goes, and something small gets built
elsewhere later.

## Where configuration lands

`providers.yaml` currently owns `protocol`, `base_url`, and `api_key_env`. All three are
dragoman's — they are its service table, and duplicating them is how two tables drift. Chancery's
file keeps only what the prompt layer needs: a model alias and the upstream model name it stands
for, plus the inheritance and per-agent overrides that already exist.

The split is clean because it follows the same rule dragoman states about itself: placement and
credentials belong to the thing that reaches the network, capability and intent to the thing that
composes the request. Chancery cannot express an endpoint after this change, and that is the point.

**The file becomes `models.yaml`, a flat map of aliases.** Once the endpoint fields go there is
nothing left for a provider block to hold, and the loader already collapses every block into one
namespace and rejects an alias defined twice — so the grouping's last job is naming a provider,
which the model name now carries itself. A file named for providers that cannot express one is the
drift the split exists to prevent.

```yaml
models:
  smart-model:
    model: openai/gpt-5.5
    reasoning_effort: high
    verbosity: low
  smart-model-prio:
    extends: smart-model
    service_tier: priority
```

There is no `service` field. The alias sets `model` — the body field it writes, prefix included —
and whatever sampling fields it wants, and `extends` inherits the lot. That is the whole schema:
one key naming an endpoint, a map of body fields underneath it.

Agents name aliases unqualified today, so no agent file changes — the live configuration is one
file rewritten by hand alongside the change.

**`instructions` is prepended, never replaced.** A caller may send their own; the agent's composed
prompt goes in front of it, separated by a blank line. Overwriting silently discards something the
caller wrote, and refusing the request makes an ordinary Responses body an error.

### Reaching the backend

Chancery gains one required environment variable, `RESPONSES_BASE_URL` — the backend's scheme, host,
and port — and one optional one, `RESPONSES_AUTH_TOKEN`, holding a bearer token attached to every
outbound request. Both name the format: `DRAGOMAN_URL` would put a target's name in the one place
every operator reads, contradicting a coupling that is a base URL and nothing more. No default for
the first: a gateway that silently assumes `localhost` fails as a connection refused deep in a
request rather than at boot, where the mistake actually is.

**The optional token is for whatever sits in front of dragoman, not for dragoman.** Dragoman
performs no client authentication by design, on the argument that a shared bearer token guarding a
process that holds provider keys is theatre — the token stops nobody who is not already on the
network, and anyone on the network who can reach the chancery container can read it out of that
container's environment. Under the deployment described below, where dragoman publishes no host
port, it protects nothing the network topology does not already protect.

It exists because that topology need not hold forever. A dragoman on a shared cluster, reachable
by workloads that are not chancery, wants a proxy doing authentication as its actual job — and then
chancery needs a credential for the proxy. Supporting that costs one environment variable here and
no change at all in dragoman, which is the whole reason to draw the line this way rather than
teaching dragoman about tokens.

This token is chancery's own credential for the next hop and is unrelated to the caller's. A caller's
`Authorization` header is consumed by chancery's JWT middleware and never forwarded.

### The backend is any Responses endpoint

Chancery writes only fields the format already has, so it does not know or care what serves them.
Dragoman is the recommended target because it is the one that speaks four dialects; OpenAI's own
`/responses` is equally valid, and so is anything else mirroring that surface. The frontmatter is
unchanged either way — `model`, `reasoning_effort`, `max_tokens`, `verbosity`, and `service_tier`
all travel in the body and are read by whoever receives it.

**The URL is always `{base}/responses`.** There is no path template, no segment, and no
target-specific branch — chancery builds one URL shape and every Responses endpoint answers it.

Which provider serves a request is a property of the model name, prefixed as OpenRouter spells it:
`openai/gpt-5.5`, `anthropic/claude-opus-4-6`. A backend fronting several providers reads the
prefix; a backend that is one provider is handed a name without one. Either way it is a value in
the `model` field, which is the field every Responses endpoint already requires — so selecting a
provider costs chancery no field, no route, and no knowledge of what it is talking to.

Two limits, both real. **Positions travel, values do not** — a reasoning level one provider accepts
another may reject, and the model name itself is now target-specific, since `openai/gpt-5.5` is a
name dragoman resolves and OpenAI does not. Both come back as an error from the target naming the
field. And **one base URL is one target for the whole instance**, so pointing directly at a
provider limits every agent to that provider's catalogue, and puts that provider's key in the
chancery container. Mixing providers is the thing dragoman exists to do.

## What the repository is afterwards

The README describes a program that will not exist. A configuration-driven gateway across five
upstream API shapes is the deleted half; documenting the survivor as a diminished version of that
would bury the part worth reading.

**Prompts as endpoints, where the file also chooses the model behind it.** That second clause is
what separates this from a server that merely holds prompts: a `.md` file is the complete
definition of an endpoint — its route is the path, its model is the frontmatter, its behaviour is
the body, and its overrides sit beside them. Nothing about the endpoint lives anywhere else.

Which is what makes the layering legible. Three files, three questions, no overlap:

| file | answers |
|---|---|
| `<agent>.md` | what this agent is — route, model alias, settings, prompt |
| `models.yaml` | which model an alias names, prefix included, and the settings it runs with |
| `dragoman.yaml` | where a prefix points, what it speaks, and which variable holds its key |

The nearest familiar thing is a static site generator, and the resemblance is mechanical rather
than rhetorical: a file's path is its URL down to the `index.md` rule, frontmatter configures the
page, `shared/` holds partials pulled in by name, and `validate` is the build — failing on a
broken include and warning on an orphaned fragment exactly as one would. The difference is that
the output is a live endpoint, so the build runs at boot instead of ahead of time.

Written as an opening line: *turns a directory of Markdown into HTTP endpoints — the path is the
route, the frontmatter picks the model, the body is the prompt.*

Neither "gateway" nor "router" survives. The first names the half being deleted. The second names
choosing, and nothing here chooses: the filename fixes the route and the frontmatter fixes the
model, so dispatch is three map lookups against a table the rest of the repository exists to
build. Downstream, dragoman genuinely routes.

## Deployment

Two containers under one compose file. Dragoman binds `0.0.0.0` inside its network and publishes
no host port; chancery publishes `8081` and reaches dragoman by service name. Every provider key is
an environment variable on the dragoman container and on no other.

That last sentence is the property worth protecting: dragoman has no client authentication by
design, on the argument that a shared bearer token in front of a process holding provider keys is
authentication theatre. It is safe here precisely because it is not reachable from outside the
compose network. Publishing dragoman's port would hand every provider key to anyone who can reach
the host.

Dragoman's `--config` replaces its embedded service table wholesale rather than merging, so the
mounted `dragoman.yaml` is the complete list of services chancery may name — which makes it exactly
the file to read when asking what this deployment can reach.

### One clone, three commands

The two binaries live in two repositories, and a reader should not have to know that. Docker
accepts a git URL as a build context, so the compose file lives here and fetches dragoman itself:

```sh
git clone …/chancery && cd chancery
cp .env.example .env       # one provider key
docker compose up
```

Four files carry it: a `Dockerfile` in each repository, plus `compose.yaml` and `.env.example`
here. Both images are multi-stage onto `scratch` or distroless — static Go binaries, a few MB
each, so building from source on a stranger's machine costs seconds rather than being the reason
they give up.

One process per container, two containers. Compose has no pods, so the sidecar shape is two
services on a network addressing each other by name; putting both binaries in one image would mean
adopting a supervisor and owning restart, reaping, and log separation that the runtime already
does.

Three details are load-bearing and each fails quietly if missed:

- **Dragoman binds `0.0.0.0` inside its container.** Its default is loopback, which the other
  container cannot reach — the symptom is a connection refused that looks like a wrong hostname.
- **`depends_on` alone is not enough.** It waits for start, not for listening. Dragoman needs a
  healthcheck and chancery a `condition: service_healthy`, or the first requests after `up` fail
  against a process that has not finished binding. Both images are `scratch` and hold no shell,
  so each binary answers its own `/health` through a `healthcheck` subcommand — a check naming
  any other command names one the image cannot execute.
- **Dragoman publishes no port.** This is what makes it safe for it to hold every key without
  client authentication, and it is one line away from not being true.

Configuration directories mount read-only: the Markdown config for chancery, `dragoman.yaml` for
dragoman. Neither is baked into an image, so a reader edits a prompt file and restarts rather than
rebuilding.

Once images are published the `build:` keys become `image:` references and nothing compiles on the
reader's machine at all. Not a prerequisite — the git-context build works from the first commit,
and needing a registry account before anyone can run the thing defeats the point.

## What dragoman changes

Two things: how a request names its service, and what a completed request logs.

### The service moves from the path into the model

**Dragoman serves `POST /responses` and nothing else.** The service is the first segment of the
`model` value — `openai/gpt-5.5` resolves service `openai` and forwards model `gpt-5.5`. Split on
the first `/` only, so OpenRouter's own nested identifiers survive: `openrouter/anthropic/claude-3`
is service `openrouter`, model `anthropic/claude-3`. A name with no prefix, or a prefix naming no
service, is the `404` the unknown-service path already returns.

`POST /<service>/responses` goes rather than staying beside it. Two routing schemes against one
table is the drift the single table exists to prevent, and a request that can name its service two
ways has to define what happens when the two disagree.

This is what lets chancery hold one URL and no service field, and it costs dragoman a stated
property: the `openai-responses` path forwards a body byte for byte today, and rewriting `model`
makes every request a re-serialisation. Nothing is lost in the re-encode — the body decodes to raw
per-key JSON, so nested values and unmodelled top-level keys survive and only key order moves —
but the guarantee is now "equivalent" rather than "unchanged", and the comment claiming otherwise
goes with it.

The `--backend` flag on the pipe is untouched. It names a service directly because it has no body
to read a prefix from, and the pipe is not the surface chancery uses.

### Accounting

Deleting the call record from chancery without replacing it anywhere loses per-request accounting
outright.

Dragoman writes nothing on success today. Its four log calls are a failed stream, a failed
request, and the drop diff at warn and debug — so nothing records that a request happened, what it
cost, or how long it took. It already holds the numbers: usage is normalised out of every dialect
into one shape and returned in the body. The missing piece is not data, it is a line.

**Dragoman gains a success log line and an env var naming which request headers to copy into it.**
Chancery attaches its identity as `X-*` headers on the forwarded request; dragoman copies the
configured ones onto the record beside the service, the model, the status, the duration, and the
token counts. What chancery logs today is then reconstructable at the point of spend, and direct
callers get the same accounting for free.

The header allowlist is not a new idea, which is the argument for it. Chancery already has one:
`LOG_REQUEST_HEADERS`, defaulting to `X-Session-ID,X-Project-ID`, capped at sixteen names, each
required to canonicalize to `X-*` and rejected if it contains a credential-like term such as
`auth`, `api-key`, `secret`, or `password`. Those constraints are the right ones and they should
travel with the feature — an operator who can name any header to log can name `Authorization`.

**An allowlist, never a passthrough.** Dragoman logs the headers it was configured to log and no
others. A request carrying fifty `X-` headers produces entries for the configured names or
nothing. Unconfigured means unlogged, so the default is silence rather than whatever arrived.

Two identity values are worth calling out. **The request ID is the load-bearing one**: chancery
still logs its own request line — a server logging what it served — while dragoman logs what that
request cost, and a shared ID is what makes those two lines one story. **The JWT subject is safe
to forward and must never be trusted.** Dragoman performs no authentication and must not begin to;
the subject is a label on a log entry. It is only trustworthy because dragoman is unreachable from
outside the compose network, which is the same property that lets it hold provider keys without
client auth.

Two fields do not survive the move intact.

**Duration changes meaning.** Chancery measured its own handling; dragoman measures its upstream
call. Over one container hop the difference is small, but it is a different measurement and should
not be read as the old one.

**The trigger is dropped, not moved.** Deriving it means walking the message array backwards for
the most recent user message or tool result and resolving that result's call back to the tool that
produced it — reading meaning out of content, which is the one thing dragoman is built to refuse
and the thing chancery is shedding. Neither side does it. What the accounting actually needs is
which agent ran, and that travels as a header like the rest of the identity. A caller wanting the
trigger back knows its own without anyone parsing anything, and can send one more header.

**Dragoman's logger writes text to stderr.** If it becomes the accounting surface it should emit
JSON, matching what chancery already emits on stdout; two formats do not aggregate.

A route and a log line are the whole of the cross-repo work. Everything else is: strip chancery, add
a client, write a compose file. The dragoman half is small enough to state here and self-contained
enough to lift into its own spec in that repository, which is where its done criteria will need to
live — and the routing change lands first, because chancery cannot reach a backend that still
requires a segment.

## Consumers this breaks

`nabu-theatron` calls `POST /embeddings` on chancery at `app/lib/embeddings/client.ts`, sending
`{"input": [...]}` and reading `data[].embedding` and `usage.total_tokens`. Its semantic search
stops working the moment the route is removed, and stays broken until a replacement exists. That
replacement is out of scope here, but the breakage is not a discovery to be made later — it is a
known cost of taking the cut now, and nabu-theatron's search should be considered down for the
interval.

No other repository in the workspace calls chancery.

## Out of scope

- **Building the embeddings replacement.** Named as a consequence above, specified elsewhere.
- **Extracting dragoman's packages out of `internal/`.** The HTTP seam is the chosen shape; a
  library dependency is a different feature with a different argument.
- **Rebuilding modes on top of the new shape.** Deleted, not deferred. If prompt modes return they
  return as a design, not as a port.
- **A chancery-side model capability table.** Which reasoning values a provider accepts is the
  provider's business and reaches chancery as an error from dragoman.
- **Retrying across services.** A failure returns; chancery does not try the same agent against a
  second backend.
- **Changing what `validate` can prove.** It checks the config it owns and stays offline. It will
  not learn to ask a backend whether a model name resolves: this repository does not know what a
  dragoman is, and a validator that reaches out for an answer only one particular backend can give
  is that knowledge arriving through the back door. A prefix is an opaque part of a model name
  here, never a thing to be checked.
- **Retry moving to the backend.** A backend that retries is deciding to spend money and holding
  per-model cooldown state, which is a different kind of program from one that forwards a request.
  Retrying a `429` stays here, where it needs to know nothing about who served it.

## Constraints

- Bound by `chancery/CLAUDE.md`.
- **The SSE event sequence a client receives is unchanged.** Same event names, same order, same
  framing. A client written against chancery today needs no edit, embeddings aside.
- No provider API key resolves inside the chancery process. Not from `models.yaml`, not from the
  environment, not for one route.
- Chancery reads no message content — not to expand a mode, not to place a cache breakpoint, not to
  derive a trigger. Messages, tools, and response formats reach the backend as raw JSON. A tool's
  name is read to select its prompt from `tools/`, which is a sibling of `input` and not content.
- Nothing on the serving path decodes an event. Terminal rendering in `call` is the sole decoder.
- No adapter, no canonical request type, and no branch on a provider name anywhere in chancery.
- `go.mod` gains at most nothing. The strip removes dependencies; it adds none.
- Chancery attaches identity as `X-*` request headers and forwards no credential of any kind on
  them — not the caller's bearer token, not a provider key.
- Dragoman logs only headers it was configured to log, under the same rules chancery applies today:
  at most sixteen, each canonicalizing to `X-*`, none containing a credential-like term.
- Dragoman still performs no authentication. A forwarded subject is a log label and grants nothing.

## Done criteria

### Offline

| check | expected |
|---|---|
| `go build ./...` | succeeds with `google.golang.org/genai` absent from `go.mod` and `go.sum` |
| `go test ./...` | passes; no test names or imports reference gemini, anthropic, or completions |
| `internal/providers` | gone as a directory |
| grep the tree for `<!--` | no match outside documentation |
| grep the tree for `protocol`, `base_url`, `api_key_env` as config fields | no match |
| grep the tree for `providers.yaml`, a `providers:` key, or a per-model `name:` field | no match |
| `RESPONSES_BASE_URL` set, `RESPONSES_AUTH_TOKEN` unset | `serve` starts; outbound requests carry no `Authorization` header |
| grep the tree for `QUOTA_`, `Reserve`, `Settle`, `CallRecord` | no match |
| grep the tree for `json.Unmarshal` over a request message | no match |
| `validate` against a config setting `seed`, `legacy_thinking`, or `cache_ttl` | reports each as an unknown field, by name, without leaking a Go type |
| `validate` against the **live** config directory named in the frontmatter | clean, and warning-free — the one warning it carries today is `embeddings.md`, which goes with the route. A hit means a real agent used something deleted |
| grep the tree for `service` as a config field or a URL segment | no match; the only URL chancery builds is `{base}/responses` |
| `validate` against a config where an agent names an alias `models.yaml` does not define | reports it as an error, naming the alias |
| `validate` against a config defining one alias twice | reports it as an error |
| `RESPONSES_BASE_URL` unset | `serve` refuses at boot, naming the variable; it does not start and fail per request |
| a request with no agent-level overrides | the body chancery sends differs from the body it received only by `model` and `instructions` |
| `list` and `list --json` | still enumerate every route and model alias |
| the compose file | no provider key appears under the chancery service |
| `README.md` mentions of protocols, adapters, modes, embeddings, quota | none, including the pipeline diagram |
| `README.md` opening | states the Markdown-to-endpoint mechanism; the configuration directory and agent frontmatter are the first sections a reader meets |
| every `README.md` route, flag, field, and environment variable | present in the built binary — checked by running each documented command against the example config |

Dragoman side, in that repository:

| check | expected |
|---|---|
| `POST /responses` with `model: openai/gpt-5.5` | reaches the `openai` service with `model: gpt-5.5` on the forwarded body |
| `POST /responses` with `model: openrouter/anthropic/claude-3` | service `openrouter`, forwarded model `anthropic/claude-3` — split on the first `/` only |
| `POST /responses` with an unprefixed model, or a prefix naming no service | `404`, exit code `2`, naming what it could not resolve |
| `POST /openai/responses` | `404` — the segment route is gone, and the unrouted handler names the one route that exists |
| `dragoman --backend openai` on the pipe | unchanged; it names a service directly and reads no prefix |
| `GET /services` | unchanged |
| a completed request with the env var unset | one success line: service, model, status, duration, token counts; no header fields |
| the env var naming two headers, both present | both appear on the record, values intact |
| the env var naming a header the request omits | logged without it; no empty key, no error |
| a request carrying headers the env var does not name | absent from the record |
| the env var naming `Authorization`, `X-Api-Key`, or seventeen headers | rejected at load: exit `2`, naming the offending entry |
| a failed request | still logs the existing failure line; the success line does not fire |
| log output | JSON |

### Live

Needs `docker compose up` and one real provider key; skipped, not failed, without one.

| check | expected |
|---|---|
| `POST /<agent>` with a minimal Responses body | `text/event-stream`, ending `response.completed`, flushed per event |
| the same request against an agent with a distinctive persona | the answer reflects the Markdown body, proving the composed prompt travelled |
| the same request carrying its own `instructions` | the agent prompt precedes the caller's; neither is lost |
| a request naming a named model (`/<agent>.<model>`) | reaches the alias's prefixed model, resolving to that provider upstream |
| agents spanning four providers in one config | each reaches its own provider through one `RESPONSES_BASE_URL` |
| the client's own view of `usage` | present on `response.completed`, unchanged — chancery stopped reading it, not stripping it |
| `call` against the same agent | renders text to the terminal and exits non-zero on `response.failed` |
| the two containers' logs for one request | correlate on a shared request ID; chancery's line says what it served, dragoman's says what it cost |
| dragoman's record for an agent request | carries the agent route and session, and token counts matching `response.completed` |
| dragoman container stopped mid-flight | chancery answers `502`/`503` promptly; no hang, no partial stream presented as complete |
| dragoman's port from the host | refused — it is not published |
| the chancery container's environment | contains no provider API key |
| `git clone` then `cp .env.example .env` then `docker compose up`, on a machine holding neither repository nor a Go toolchain | both images build and both services come up |
| the first request issued immediately after `up` returns | succeeds — the healthcheck gate held, not a connection refused |
| editing a prompt file and restarting chancery | the change takes effect without rebuilding an image |
| both image sizes | single-digit MB |

The persona check is the one that cannot be dropped. Every other check passes on a chancery that
forwards the body untouched and composes nothing, which is precisely the failure this change is
capable of introducing.

## Deliberately undecided

**Whether the backend is one-per-chancery or shared.** Not deferred for lack of an answer — the
answer is that it must not matter, and it does not. Nothing here knows what it is talking to, the
backend holds no per-request state, and the difference is one line of compose. Both topologies
deploy from the same binaries and the same configuration.

Guessing which one a future deployment wants would be worse than the indifference, because the
guess would have to be built into something to be worth making. The one place a future topology
could have reached back into the design is a credential for whatever fronts a shared backend, and
that is already an optional environment variable costing five lines. Statelessness is what buys
the rest.

Recorded here so a later reader does not mistake an open question for an oversight and go fix it.

## Open questions

None. `nabu-theatron` is the only caller, it touches embeddings and no agent route, and it is
being edited for embeddings regardless — so the `messages` to `input` rename breaks nothing that
was not already breaking.
