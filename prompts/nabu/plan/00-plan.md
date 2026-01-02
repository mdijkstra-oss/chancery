# Plan Phase

<task>
Generate a plan for the described task.
</task>

<output_format>
```json
{"type": "plan", "task": "what we're accomplishing", "steps": ["step 1", "step 2", "step 3"]}
```

Steps are concise descriptions. Each step should be a single verifiable action.
</output_format>

<plan_rules>
- Steps are sequential
- Each step: one action, verifiable outcome
- Include reads before writes when context needed
- Plans can be revised mid-execution
</plan_rules>

<stuck>
If you need clarification before planning:
```json
{"type": "stuck", "question": "what you need to know"}
```
</stuck>
