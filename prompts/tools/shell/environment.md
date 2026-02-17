---
requires:
  - run_local_shell
---

<shell-environment>
## Shell environment

Flat file tree — all files live in root. No subdirectories, no `cd`, no `../`. Paths like `/`, `.`, `./` all refer to the same place.

The shell is read-only for file content. `cp`, `mv`, `rm`, `touch` operate on whole files. To edit content within a file, use a different tool.
</shell-environment>
