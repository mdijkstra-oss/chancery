# Migrating a configuration directory

Chancery no longer reaches a provider, so a configuration directory no longer describes one. This is
what a live directory has to become, verified against
`/Users/matthijn/Desktop/hermes-logos-config-full-2026-07-17` — 63 Markdown files, 16 agents, 25
model aliases across four providers. Twenty-four survive; the embedding alias is the one that
leaves.

The conversion was performed on a copy. The Desktop original is untouched.

## What a human must do

1. **Rename `providers.yaml` to `models.yaml` and flatten it.** Provider blocks go; every alias
   moves to the top level of one `models:` map. Each alias's `name:` becomes `model:` carrying the
   provider prefix — `openai/`, `anthropic/`, `gemini/`, `deepseek/`, matching the service names in
   `dragoman.yaml`. Alias keys are unchanged, so no agent file changes.
2. **Drop the endpoint fields.** `protocol`, `base_url`, `api_key_env`, and `strict` are dragoman's
   service table; they belong in `dragoman.yaml` and nowhere else.
3. **Delete the embedding alias and its agent.** `text-embedding-3-large` carried `type: embedding`
   and `dimensions: 1024`, neither of which is a field any more, and `embeddings.md` is the route
   that went with it.
4. **Delete `legacy_thinking: true`** from `gemini-2.5-flash-lite` and `gemini-2.5-flash`.
5. **Delete `modes/` entirely.** It is no longer a reserved directory, so a leftover `modes/` is
   walked as ordinary Markdown and every file in it becomes an orphan error. Ignoring it is not an
   option; it has to go.
6. **Delete `seed: true` and `temperature: 0` from `hyde-generator/index.md` and
   `topic-assigner/index.md`.** See below.
7. **Run `validate`.** It must come back `✓ config valid (0 warnings)`.

## Two agents break

The specification assumed no agent set a deleted field. Two do. Before the `seed:` lines were
removed, the converted copy reported:

```text
✗ hyde-generator/hyde-generator.md: orphaned Markdown file has no frontmatter and is not included by an agent
✗ hyde-generator/index.md: malformed YAML frontmatter: line 4: unknown field "seed"
✗ topic-assigner/index.md: malformed YAML frontmatter: line 4: unknown field "seed"
✗ topic-assigner/topic-assigner.md: orphaned Markdown file has no frontmatter and is not included by an agent
```

Both set `seed: true` beside `temperature: 0`, and both run on `gemini-2.5-flash-lite`. The two
orphan errors are collateral: frontmatter that fails to parse never marks its includes referenced,
so each agent's sibling fragment is reported as unreachable as well.

`seed` was a Gemini-only switch consumed by the deleted adapter, with no position in an
`openai-responses` body. `temperature` has a position and no longer has a frontmatter field: a
reasoning model rejects it or ignores it, and an agent file that pins it is pinning a value most of
the catalogue will not take. **These two agents lose deterministic sampling entirely.** Whether
that is acceptable for HyDE generation and topic assignment is a question for whoever owns those
agents; nothing in the conversion can answer it.

A caller that needs a temperature still sends one: it is a field of the request body and travels
untouched. What is gone is the ability to fix it in the agent file.

With those four lines removed, the whole directory is clean:

```console
$ chancery --config ./config validate
✓ config valid (0 warnings)
```

```console
$ chancery --config ./config list
PATH                      MODEL                         REASONING
commit-agent              openai/gpt-5-mini             off
compacter                 gemini/gemini-3.5-flash       minimal
corpus-describer          gemini/gemini-3.1-flash-lite  minimal
cv                        gemini/gemini-3.5-flash       none
deep-analysis-adjudicate  anthropic/claude-opus-4-6     medium
deep-analysis-filter                                    
  .deep                   anthropic/claude-opus-4-6     low
  .fast (default)         openai/gpt-5.5                low
file-hyde                 gemini/gemini-3.1-flash-lite  minimal
generic-hyde              gemini/gemini-3.1-flash-lite  minimal
hyde-generator            gemini/gemini-2.5-flash-lite  minimal
qual-coder                deepseek/deepseek-v4-pro      high
refine-code               anthropic/claude-opus-4-6     medium
scout-filter              deepseek/deepseek-v4-flash    none
section-labeler           gemini/gemini-3.1-flash-lite  minimal
semantic-filter           deepseek/deepseek-v4-pro      none
topic-assigner            gemini/gemini-2.5-flash-lite  minimal
15 agents · 16 models
```

Sixteen agents became fifteen: `embeddings` is the one that left.

## Twelve tool prompts still compose

`tools/` holds twelve Markdown files, 26 KB in total, each named `<file>.<toolname>.md` across
`shell/`, `delegation/`, `search/`, `query/` and `patching/`. A file is appended to an agent's
instructions when the request declares a tool matching its suffix, and nothing about that changed.

The names are read from the `tools` array, which is a sibling of `input` rather than part of it, so
selecting a prompt costs no reading of message content. The array reaches the backend byte for
byte.

Verified against this directory: `commit-agent` offered a tool named `run_local_shell` composes
5,147 characters of instructions ending in `tools/shell/grep.run_local_shell.md`.

## What survives untouched

- **Every agent's model line.** Aliases are named unqualified and their keys did not change.
- **`extends`.** All four chains are one level deep and translate verbatim.
- **`deep-analysis-filter`'s named models.** `models:` and `default:` are unchanged.
- **Every fragment and every include.** `shared/` and `tools/` are still reserved, and both still
  resolve by name — a fragment beside the agent's own directory first, a tool prompt by the name
  the request gives the tool.
- **`cache_ttl` and `auto_cache`.** Neither was ever set — `cache_ttl` appeared only in three
  comments deliberately marking it disabled.

`modes/` is the only content that has to be deleted. Nothing in the configuration authors a `<!-- prompt: … -->`
marker; the two mode files were expanded from markers the client sent, and a client that keeps
sending one now has an ordinary system message pass through untouched. `qual-coder` is the agent
whose prompts talk about planning and execution mode, through `shared/nabu/discipline.md` and
`shared/chat/routing.md` — those fragments still compose, they just no longer have modes behind
them.

## The converted models.yaml

```yaml
models:
  gpt-5.5:
    model: openai/gpt-5.5
    verbosity: low
    reasoning_summary: concise
  gpt-5.4:
    model: openai/gpt-5.4
    verbosity: low
    reasoning_summary: concise
  gpt-5.4-mini:
    model: openai/gpt-5.4-mini
    verbosity: low
    reasoning_summary: concise
  gpt-5.2:
    model: openai/gpt-5.2
    verbosity: low
    reasoning_summary: concise
  gpt-5.2-prio:
    extends: gpt-5.2
    service_tier: priority
  gpt-5.1:
    model: openai/gpt-5.1
    verbosity: low
    reasoning_summary: concise
  gpt-5-mini:
    model: openai/gpt-5-mini
    verbosity: low
    reasoning_summary: concise
  gpt-5-mini-prio:
    extends: gpt-5-mini
    service_tier: priority
  gpt-5-nano:
    model: openai/gpt-5-nano
  gpt-5-nano-prio:
    extends: gpt-5-nano
    service_tier: priority
  gpt-5.4-nano:
    model: openai/gpt-5.4-nano
  gpt-5.4-nano-prio:
    extends: gpt-5.4-nano
    service_tier: priority
  gpt-4o-mini:
    model: openai/gpt-4o-mini

  claude-opus-4-6:
    model: anthropic/claude-opus-4-6
    reasoning_effort: minimal
  claude-sonnet-4-6:
    model: anthropic/claude-sonnet-4-6
    reasoning_effort: minimal
  claude-haiku-4-5:
    model: anthropic/claude-haiku-4-5
    reasoning_effort: none

  gemini-3.1-pro:
    model: gemini/gemini-3.1-pro
    reasoning_effort: low
  gemini-3-flash:
    model: gemini/gemini-3-flash
    reasoning_effort: minimal
  gemini-3.5-flash:
    model: gemini/gemini-3.5-flash
    reasoning_effort: minimal
  gemini-3.1-flash-lite:
    model: gemini/gemini-3.1-flash-lite
    reasoning_effort: minimal
  gemini-2.5-flash-lite:
    model: gemini/gemini-2.5-flash-lite
    reasoning_effort: minimal
  gemini-2.5-flash:
    model: gemini/gemini-2.5-flash
    reasoning_effort: minimal

  deepseek-v4-flash:
    model: deepseek/deepseek-v4-flash
  deepseek-v4-pro:
    # Temporary pricing through May 2026.
    model: deepseek/deepseek-v4-pro
```

Blank lines group the file by prefix; nothing reads them. The alias keys are globally unique
already, so flattening four provider blocks into one map loses no name.

The `deployment` half of this is `dragoman.yaml` in the repository root, which names the four
services these prefixes resolve against and the environment variable each one's key lives in.
