---
description: "Alias -> /spec-all"
argument-hint: "same arguments as /spec-all"
disable-model-invocation: true
metadata:
  forwards_to: sza:spec-all
  generated_by: sza aliases/register-spec-commands.ps1
---

`/sa` runs the sza plugin skill `sza:spec-all` in this project and carries no behaviour of its own.

Invoke the Skill tool with skill `sza:spec-all`, passing `$ARGUMENTS` as its args, and follow that skill exactly. Do
nothing before that call and do not restate this file - the operator typed a shortcut, not a new instruction.

If the Skill tool says `sza:spec-all` does not exist, the sza plugin is not installed or is older than this command: say so
in one line and stop. `claude plugin update sza@sza-unified-rules` is the owner's step.