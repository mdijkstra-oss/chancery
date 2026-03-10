
<preflight>
# Preflight

The gateway for any file work beyond a bounded single action. Combines file segmentation with approach loading.

Pass the task description, relevant files with their purpose, and approach keys for task-specific playbooks. Large files are segmented into a table of contents, small files inlined. Selected approach playbooks and their parent index files are returned alongside the manifests.

Every file path must come from the file listing. Never infer, guess, or assume a file exists — if it's not in the listing, don't pass it. `preferences.md` and `settings.hidden.md` are never passed; they're injected separately.

Preflight transitions to planning mode. After it returns, build the plan with the user.

Skip it only for: answering questions, giving feedback, looking something up, or a single bounded edit where the target is already known.
</preflight>
