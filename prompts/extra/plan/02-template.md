# Plan: {concise title describing the task}

## Goal
What success looks like for this task. Taken from the outcome field, sharpened based on what you learned during planning. Specific enough that the executor can check its own work against this when done.

## Involvement
How and when the executor interacts with the user, translated from the involvement field into concrete guidance. Not "check in if ambiguous" — instead: "Code clear sections autonomously. Batch ambiguous items and present them to the user after completing each file." The executor should not have to interpret an involvement level.

## Steps

- [ ] First step — a concrete action with a clear done state
    - [ ] Branch or sub-step if needed
    - [ ] Exit condition → what to do (reject, ask user, skip)
- [ ] Next step
- [ ] For each {item} in {collection}:
    - [ ] What to do per iteration
    - [ ] What to do when {edge case}
    - [ ] What to do when {other edge case}
    - [ ] Write output as you go / collect for batch — be explicit
- [ ] After the loop or main work:
    - [ ] Collect {things that were flagged or deferred}
    - [ ] Present to user (if involvement requires it)
    - [ ] Apply user decisions
    - [ ] Final output step
- [ ] Resolve with {specific artifacts}

## Decisions
Judgment calls you made during planning that the executor should not re-litigate. These are project-specific choices about how to approach the work — not domain knowledge, but decisions shaped by the intent, involvement, and constraints of this particular task.

Each decision should be a clear statement the executor can follow without needing the reasoning behind it, though brief reasoning helps if the decision isn't obvious.

## On failure
What to do when things go wrong that isn't tied to a specific step. When to reject, when to ask the user, when to continue with a best effort. These are the fallback rules the executor follows when something unexpected happens that the steps don't cover.