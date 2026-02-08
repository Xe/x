# markdownlang Specification

## Overview

markdownlang programs are markdown files with YAML front matter that define AI agents. Each program accepts structured input, processes it with an LLM, and returns structured output.

## Structure

A program has two parts:

1. **Front Matter** (YAML between `---` delimiters) - defines metadata
2. **Description** (markdown) - tells the LLM what to do

### Example

```markdown
---
name: fizzbuzz
input:
  type: object
  properties:
    start: { type: integer, minimum: 1 }
    end: { type: integer, minimum: 1 }
  required: [start, end]
output:
  type: object
  properties:
    results:
      type: array
      items: { type: string }
  required: [results]
---

# FizzBuzz

For each number from {{ .start }} to {{ .end }}, output:

- "Fizz" if divisible by 3
- "Buzz" if divisible by 5
- "FizzBuzz" if divisible by both
- The number otherwise
```

## Front Matter

Fields:

- `name` (string, required): Program identifier
- `description` (string): Short description
- `input` (JSON Schema): Input validation schema
- `output` (JSON Schema): Output schema the LLM must follow (output is always JSON; `{}` if nothing to return)
- `imports` (array): Programs this agent can call
- `mcp_servers` (array): MCP servers for tools

```yaml
name: my-program
description: Fetch data from a URL
input:
  type: object
  properties:
    url: { type: string, format: uri }
    timeout: { type: integer, minimum: 1, maximum: 300, default: 30 }
  required: [url]
output:
  type: object
  properties:
    status: { type: string, enum: [success, error] }
    data: { type: object }
  required: [status]
```

## Imports

List other programs this agent can call:

```yaml
imports:
  - ./utils/logger.md
  - ./validators/check.md
  - stdlib:json-parser
```

Paths can be relative, absolute, or named (`stdlib:name`). Imports are not recursive - imported programs don't expose their imports. Circular dependencies are errors.

## MCP Servers

MCP servers provide tools for agents to use:

```yaml
mcp_servers:
  - name: filesystem
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/data"]
  - name: python-interpreter
    command: uvx
    args: ["mcp-server-python-interpreter"]
```

Fields:

- `name` (required): Server identifier
- `command` (required): Command to start server
- `args` (optional): Command arguments
- `env` (optional): Environment variables
- `disabled` (optional): Don't start this server

**Special servers:**

- `python-interpreter`: Prefer for calculations, data processing, algorithms
- `fetch`: HTTP requests (GET, POST, etc.)

## Program Description

The markdown content after front matter tells the LLM what to do. It receives:

- The description text
- Input values interpolated via Go templates
- Available tools from MCP servers
- Available imported programs to call

## Template Interpolation

Use Go template syntax to reference input values:

- `{{ .variable }}` - Variable reference
- `{{ .nested.field }}` - Nested field
- `{{ if .cond }}...{{ end }}` - Conditional
- `{{ range .items }}...{{ end }}` - Loop

Available functions: `upper`, `lower`, `title`, `default`, `len`, `slice`, `join`, `split`

```markdown
{{ if .debug }}
Debug mode: processing {{ .files | len }} files
{{ end }}
```

## Execution

1. Input is validated against the input schema
2. Input values are interpolated into the description
3. LLM iterates with available tools until output satisfies the output schema
4. Output is returned as JSON

**Output MUST always be valid JSON** matching the output schema. If there's nothing to return, output `{}`.

On validation failure: retry with error feedback (max 10 iterations).

## Agent Calls

Programs can call imported programs. Calls are blocking - the caller waits for completion.

Example:

```yaml
imports:
  - ./fetch.md
  - ./process.md
```

`fetch` and `process` can be called and will block until complete.

## Tool Calls

MCP tools are available to the LLM during execution. The LLM sees tool schemas and can call them. Results are returned in subsequent iterations.

## Future Expansion: Agent Mailboxes

Currently, agent calls are blocking - callers wait for completion. A future expansion will introduce **mailboxes** for non-blocking inter-agent communication:

- **Send** - Send a message to another agent's mailbox without blocking
- **Receive** - Check mailbox for incoming messages
- **Message format** - JSON with sender, timestamp, and payload
- **Persistence** - Mailboxes persist beyond agent lifetime
- **Querying** - Filter messages by sender, time, or content

This enables asynchronous workflows, fan-out patterns, and long-running agent coordination.

## SKILL.md to markdownlang Conversion

SKILL.md files (Claude Code skills) can be converted to markdownlang programs:

### Mapping

| SKILL.md                 | markdownlang                   |
| ------------------------ | ------------------------------ |
| Description (markdown)   | Program description (markdown) |
| Parameters (in prompt)   | `input` schema (JSON Schema)   |
| Return value (implied)   | `output` schema (JSON Schema)  |
| Tool usage               | `mcp_servers` list             |
| `/skill-name` invocation | `imports` reference            |

### Conversion Process

1. **Extract parameters** from the skill description (e.g., "takes a URL" → `url: {type: string, format: uri}`)
2. **Define output schema** based on what the skill returns
3. **Create front matter** with `name`, `input`, `output`, `mcp_servers`
4. **Copy description** as-is (it's already markdown)
5. **Update template references** - replace placeholders with `{{ .param }}`
6. **Add imports** if the skill calls other skills

### Example

Original SKILL.md:

```markdown
# URL Fetcher

Fetches a URL and returns the status and content.

Parameters:

- url (required): The URL to fetch
- format (optional): Response format (json, xml, html, text)

Uses the fetch tool to make HTTP requests.
```

Converted markdownlang:

```markdown
---
name: url-fetcher
input:
  type: object
  properties:
    url: { type: string, format: uri }
    format: { type: string, enum: [json, xml, html, text], default: json }
  required: [url]
output:
  type: object
  properties:
    status: { type: integer }
    content: { type: string }
  required: [status, content]
mcp_servers:
  - name: fetch
    command: npx
    args: ["-y", "@modelcontextprotocol/server-fetch"]
---

# URL Fetcher

Fetches {{ .url }} and returns the status and content.

Uses the fetch tool to make HTTP requests. Response format: {{ .format | default "json" }}.
```

## Examples

### FizzBuzz

```markdown
---
name: fizzbuzz
input:
  type: object
  properties:
    start: { type: integer, minimum: 1 }
    end: { type: integer, minimum: 1 }
  required: [start, end]
output:
  type: object
  properties:
    results: { type: array, items: { type: string } }
  required: [results]
---

For each number from {{ .start }} to {{ .end }}:

- "FizzBuzz" if divisible by 3 and 5
- "Fizz" if divisible by 3
- "Buzz" if divisible by 5
- The number itself
```

### Data Fetcher

```markdown
---
name: fetch-data
input:
  type: object
  properties:
    url: { type: string, format: uri }
    format: { type: string, enum: [json, xml, html, text], default: json }
  required: [url]
output:
  type: object
  properties:
    status: { type: string, enum: [success, error] }
    data: { type: object }
  required: [status]
mcp_servers:
  - name: fetch
    command: npx
    args: ["-y", "@modelcontextprotocol/server-fetch"]
---

Fetch {{ .url }} and parse as {{ .format }}. Use the fetch tool, then return the parsed data or an error.
```

### Text Processor

```markdown
---
name: text-processor
input:
  type: object
  properties:
    text: { type: string }
    operations:
      type: array
      items:
        {
          type: string,
          enum: [word_count, sentiment, extract_entities, summarize],
        }
  required: [text, operations]
output:
  type: object
  properties:
    results: { type: object }
  required: [results]
mcp_servers:
  - name: python-interpreter
    command: uvx
    args: ["mcp-server-python-interpreter"]
---

Process text using python-interpreter for: {{ .operations | join ", " }}.
```

## Implementation Notes

- Input and output MUST be validated against their schemas
- Output MUST always be valid JSON (use `{}` when there's nothing to return)
- Template rendering should prevent injection attacks
- MCP servers start on agent init, stop on completion
- Retry on transient errors (default max 10)
- Limit resource usage and execution time

## References

- **JSON Schema**: Draft 2020-12. Types: string, number, integer, boolean, array, object, null. Validation: required, enum, minimum/maximum, minLength/maxLength, pattern, format
- **Go Templates**: `{{ .var }}`, `{{ if }}`, `{{ range }}`. Functions: upper, lower, title, len, default, slice, join, split
- **MCP**: Model Context Protocol for exposing tools to LLMs. See https://modelcontextprotocol.io

---

**Version:** 1.0.0
