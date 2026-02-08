---
name: verbose-tool-list
description: List every tool you have
input: {}
output:
  type: object
  properties:
    tools:
      type: array
      items:
        type: string
  required: [tools]
imports:
  - ./fizzbuzz.md
  - ./word-count.md
mcp_servers:
  - name: filesystem
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "."]
---

List every tool you have access to.
