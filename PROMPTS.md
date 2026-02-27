# Writing prompts for modern LLMs

## Three layers, zero overlap

| Layer | Teaches | Where in code |
|-------|---------|---------------|
| Tool schema | Format, arg names, types | `nabu-theatron/app/lib/agent/executors/*.ts` — zod schemas with `.describe()` |
| Tool prompt | Tool-specific patterns | `hermes-logos/prompts/tools/**/*.md` — gated by filename convention |
| Domain prompt | Domain intent and judgment | `hermes-logos/prompts/shared/**/*.md` — composed by agent manifests |

Each layer trusts the one below. Never re-teach what a lower layer already covers.

## Rules

### Layering

**Tool schema teaches format.** The model knows how to call a tool from its JSON schema. Do not restate arg names, types, or call structure in prose. Do not wrap domain examples in full tool call JSON — show the value shape only.

**Domain prompts describe what and when, not how.** "Add an annotation when the passage expresses frustration" — good. Wrapping that in `{ path, language, operations: [{ op, path, value }] }` — bad. The model fixates on reproducing your format instead of reasoning about the domain.

**Redundancy is worse than omission.** When the same information appears in multiple layers, modern models waste reasoning tokens reconciling the copies. If they conflict even slightly, performance degrades sharply. Say it once, in the right layer.

**Contradictions are the biggest enemy.** GPT-5 and Claude 4.x both expend reasoning tokens reconciling contradictions — worse than verbosity, worse than poor position. One clear rule beats two rules that mostly agree. When rules span layers or blocks, audit for conflicts.

### Writing style

**Direct rules for behavior, narrative for domain knowledge.** Behavioral instructions (how to talk to the user, when to stop, what to include) should be direct and imperative — the model follows them with precision. Domain knowledge (methodology, analytical frameworks) can stay narrative — the model internalizes it as understanding, not as rules to pattern-match.

Bad (behavioral instruction as narrative):
> "The plan's primary job is encoding when the user is consulted and what units of work exist. It is not a detailed work breakdown."

Good (same instruction, direct):
> "Encode when the user is consulted and what work units exist. Not a detailed work breakdown."

**One sentence of why, not a paragraph.** Context behind a rule helps the model generalize, but keep it brief. "Never use ellipses" → "Never use ellipses — the text-to-speech engine can't pronounce them." One line of motivation, not a paragraph of rationale.

**Principles + explicit rules together.** Principles let the model reason about novel situations. Literal rules catch known failure modes. Use both — the principle handles edge cases, the rule prevents known mistakes. An if/then tree for every scenario is brittle. A principle alone misses specific traps.

**Normal language, not aggressive.** Claude 4.6 and GPT-5.2 overtrigger on "CRITICAL: You MUST" language that was needed for older models. Both have precise instruction following — use clear, direct language without shouting. "Use this tool when..." not "CRITICAL: You MUST ALWAYS use this tool when...".

### Structure

**XML tags for semantic sections.** Both Claude and GPT-5.x benefit from tagged sections (`<planning>`, `<boundaries>`, `<interpretation>`) as organizational anchors. The model references and respects instruction categories better when they're semantically labeled.

**Front-load behavioral rules within each block.** Lead each section with constraints and rules, follow with context. The model weighs the opening of each tagged section more heavily.

**Named frameworks activate training knowledge.** "As per constant comparative method" compresses a textbook into three words. Reference methodologies by name — the model brings depth from training. Don't re-explain what the name already teaches.

### Prompt types by purpose

**Domain knowledge** (methodology, analytical frameworks): narrative prose, principles, named frameworks. The model internalizes these as understanding. Can be longer — they're reference material consulted when relevant.

**Behavioral instructions** (chat style, planning process, execution discipline): direct rules, short sentences, concrete. These govern every turn. Keep them tight.

**Value shapes** (annotation format, plan JSON, codebook structure): show the shape once with a judgment-laden example. No tool call wrapper.

### System architecture

**Tool results steer more than prompts.** A tool that returns current state — progress, next item, what changed — keeps the model oriented without runtime reminders. Invest in informative tool results before adding nudges.

**Side-effects belong at exit, not mid-loop.** Capture durable side-effects (memory updates, observations) as data on the resolve call and dispatch a focused agent. Don't nudge the main agent to do it mid-work.

**Few-shot examples show domain judgment, not format.** Modern models understand tool calling from schema alone. Examples help when they show *what to decide*, not *how to format the call*.

**Over-prompting causes overtriggering.** Replace "always use this tool when..." with targeted instructions. Tools that undertriggered on older models trigger correctly on Claude 4.x and GPT-5.x.

## What goes where

**In the tool schema (function definition):**
- Parameter names, types, required/optional
- One-line description of what the tool does

**In the tool prompt (e.g. selectors.patch_json_block.md):**
- Patterns specific to the tool: selector syntax, batching, `"""` content
- When to use this tool vs another tool
- Validation behavior, error recovery

**In the domain prompt (e.g. qualitative-researcher/coding.md):**
- Domain concepts: what an annotation is, what codes mean
- Decision criteria: when review is warranted, when to merge
- Value shapes: the data structure the domain works with (without the tool call wrapper)
- Workflow: sequences of domain actions (resolve ambiguity → record feedback → optionally merge)
- Examples that show judgment calls, not format

## Tool errors are prompts

The LLM reads tool errors. An error message is a prompt that teaches recovery.

**Standard errors pass through unchanged.** LLMs understand Unix stderr from training. `grep: No such file or directory` is already clear. Don't wrap or transform known error formats.

**Custom errors state: what failed, why, what to do.** "Numeric index `/annotations/0` rejected. Target array items by selector: `/annotations[id=annotation_xyz]`" — the model can fix this in one retry. "Invalid path" — the model guesses.

**Show available values on constraint violations.** "Unknown code `code_xyz`. Available: `code_a1b2`, `code_c3d4`, `code_e5f6`" beats "Unknown code." The model picks from the list instead of hallucinating an ID.

**Partial success separates succeeded from failed.** When 3 of 5 operations succeed, report which ones failed. The model shouldn't retry what already worked.

**Keep errors concise.** No stack traces, no debug info, no internal function names. Every token is attention cost.

**Don't pre-prompt for errors that aren't happening.** If the model handles tool choice and recovery well on its own, don't add error guidance to prompts. Only add targeted corrections when you see a recurring failure pattern.

## Anti-patterns

- Full tool call JSON in domain prompts (model fixates on format)
- Restating tool arg types in prose ("path is a string containing...")
- Identical guidance in tool prompt and domain prompt
- Chain-of-thought examples for reasoning models
- "Always use X" blanket rules (causes overtriggering)
- Many examples showing the same thing (diminishing returns, increases fixation)
- Format examples where domain examples belong
- Blanket reminders every N turns (model fixates on the reminder instead of the work)

## Codebase architecture

Four directories, three mechanisms:

```
prompts/
├── agents/    ← manifests (compose from shared/) + config.json
├── modes/     ← mode overlays (compose from shared/, expanded at request time)
├── shared/    ← all building block prompts
└── tools/     ← tool-gated prompts
```

### Tool schemas (zod)

Location: `nabu-theatron/app/lib/agent/executors/*.ts`

Each tool is a zod schema with `.describe()` on fields. This generates the JSON schema the model sees as the tool definition. The model learns format, arg names, and types from this alone.

### Tool prompts

Location: `hermes-logos/prompts/tools/**/*.md`

Gated by filename convention: `concept.toolname.md`. The segment after the last `.` (before `.md`) is the required tool name. If the agent doesn't have that tool, the prompt is skipped. Files without a dot in the base name are always included.

Examples: `selectors.patch_json_block.md`, `grep.run_local_shell.md`, `outcomes.resolve.md`.

Loaded per-request based on the tools the frontend sends. Appended after the agent's manifest prompt.

### Shared prompts

Location: `hermes-logos/prompts/shared/**/*.md`

All building block prompts — identity, discipline, methodology, chat behavior, planning, execution, etc. Never loaded directly. Only included when a manifest or mode overlay references them.

### Agent manifests

Location: `hermes-logos/prompts/agents/**/*.md`

Each `.md` file is a manifest that composes shared prompts via `[path.md]` syntax. Includes resolve relative to `prompts/shared/`. Literal text between includes is kept as-is.

```markdown
[nabu/identity.md]
[nabu/discipline.md]
[expert/approach.md]

Glue text specific to this agent.

[chat/style.md]
```

Naming determines the registry key:
- `agents/compacter/index.md` → key: `compacter`
- `agents/qual-coder/index.md` → key: `qual-coder`

One manifest per agent — the base prompt. Mode-specific behavior comes from mode overlays.

### Mode overlays

Location: `hermes-logos/prompts/modes/*.md`

Each `.md` file is a manifest (same `[include.md]` syntax, resolved from `shared/`). Compiled at startup into `Registry.Modes` as `map[string]string` keyed by filename without extension.

The frontend pushes `<!-- prompt: planning -->` or `<!-- prompt: execution -->` as system messages during mode transitions. The server expands these markers into the compiled mode prompt before sending to the LLM.

### Config cascade

`config.json` at any directory level under `agents/`. Agents without a direct config inherit from the nearest parent directory.

### Assembly flow

1. `CompileRegistry` walks `agents/` and `modes/` at startup, resolves all manifests from `shared/`
2. Request hits backend: `POST /qual-coder`
3. URL path maps directly to registry key → compiled manifest prompt
4. Messages containing `<!-- prompt: X -->` markers are expanded to compiled mode prompts
5. Tool prompts loaded per-request, filtered by available tools, appended
6. Concatenated prompt sent as system message

### Where to edit

| Want to change | Edit |
|----------------|------|
| Tool arg names/types/descriptions | Zod schema in `nabu-theatron/.../executors/*.ts` |
| Tool-specific patterns (selectors, batching) | `hermes-logos/prompts/tools/**/*.md` |
| Domain intent (what to annotate, when to merge) | `hermes-logos/prompts/shared/**/*.md` |
| Which prompts an agent uses | Manifest in `hermes-logos/prompts/agents/**/*.md` |
| Mode-specific behavior (planning, execution) | Mode overlay in `hermes-logos/prompts/modes/*.md` |
| Which tools a mode has | Mode config in `nabu-theatron/.../modes.ts` |
| Model/reasoning settings | `config.json` in `hermes-logos/prompts/agents/` |
