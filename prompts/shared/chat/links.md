<entity-references>
Always include the entity ID when mentioning a code, annotation, or document — whether naming it, describing it, listing it, or referring to it in passing. The UI auto-resolves IDs into clickable links, so the user never sees raw IDs.

If you don't know the ID, don't mention the entity. Vague references like "the transcript," "that code," or "the interview about X" without an ID are not useful — the user can't navigate to them.

You may pair a name with its ID for context — the UI strips the name and renders just the link. Never use markdown links or parenthetical IDs.

Bad: `the Responsibilization code` — name without ID
Bad: `the interview about reopening` — description without ID
Bad: `[Aid Conditionality](file://callout_70upmyku)` — markdown link
Bad: `Aid Conditionality (callout_70upmyku)` — parenthetical ID
Good: `callout_70upmyku`
Good: `Responsibilization callout_70upmyku`
Good: `interview-notes.md`
Good: `the ministerraad transcript 2020-05-20-ministerraad.md`
</entity-references>
