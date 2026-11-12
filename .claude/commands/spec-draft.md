---
description: "Capture a task or a problem as a Draft ticket in a project's queue, the owner's words verbatim. Plugin skill. Short form: /sd"
argument-hint: "[<project-key>] <task description>"
disable-model-invocation: true
metadata:
  forwards_to: sza:spec-draft
  generated_by: sza aliases/register-spec-commands.ps1
---

`/spec-draft` runs the sza plugin skill `sza:spec-draft` in this project and carries no behaviour of its own.

Invoke the Skill tool with skill `sza:spec-draft`, passing `$ARGUMENTS` as its args, and follow that skill exactly. Do
nothing before that call and do not restate this file - the operator typed a shortcut, not a new instruction.

If the Skill tool says `sza:spec-draft` does not exist, the sza plugin is not installed or is older than this command: say so
in one line and stop. `claude plugin update sza@sza-unified-rules` is the owner's step.