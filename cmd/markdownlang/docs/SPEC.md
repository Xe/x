# markdownlang: An AI Agent Language That Doesn't Suck

Let's be honest: most AI agent frameworks are over-engineered garbage. They want you to learn some bespoke DSL, wrangle a mountain of YAML, or — god forbid — write Python in production. I wanted something that actually works, so I built markdownlang.

## The Problem I'm Trying to Solve

I write a lot of agents. Like, a _lot_. And every time I had to touch LangChain or AutoGen or whatever the Framework Du Jour was, I wanted to scream. Why does everything need to be so complicated? Why can't I just write down what I want the LLM to do in plain English and have it work?

markdownlang is my answer to that frustration. It's a language for writing AI agents that:

- Uses markdown as the interface — you already know it
- Lets the LLM do the actual work — no complex DSL to learn
- Validates everything with JSON Schema — type safety without the headache
- Composes naturally — agents can import and call other agents
- Plays nice with the ecosystem — MCP tools, OpenAI's Responses API, the whole deal

It's not revolutionary. It's just what happens when you strip away all the cruft and focus on what actually matters: getting an LLM to do what you want, reliably.

## How It Works

Here's the whole idea, top to bottom:

```text
markdownlang program (.md)
       |
       v
    Parser (YAML front matter + JSON Schema validation)
       |
       v
    Executor (orchestrates execution)
       |
       v
    Agent Loop (max 69 iterations)
       |
       +-- LLM API Call (OpenAI Responses API v3)
       +-- Tool Execution (MCP servers, Python interpreter, imported agents)
       +-- Schema Validation (input/output)
       |
       v
    Result (JSON matching output schema)
```

The LLM runs in a loop. If it returns invalid JSON, we tell it what went wrong and try again. Maximum 69 iterations — after that, we bail out because something has gone horribly wrong.

This is the whole trick, really. The validation feedback loop means we _always_ get output that matches the schema. The LLM learns from its mistakes. It's almost elegant.

## Design Decisions I Made (And Why)

I made some choices here. You might disagree with them. That's fine.

**Markdown + YAML over a custom DSL** — I didn't want to invent Yet Another Syntax. Markdown is universal. YAML is... well, YAML, but at least people already hate it for specific reasons rather than new reasons.

**Iterative LLM loop** — This guarantees output schema compliance through validation feedback. The LLM gets told when it messed up and tries again. It's brute-force, but it works.

**Blocking agent calls** — Right now, when one agent calls another, it waits for the result. This makes debugging way easier and execution predictable. Async mailboxes are planned for later, but I wanted to get the basics right first.

**Wasm Python** — The Python interpreter runs in a WebAssembly sandbox. Yeah, it's slower than native. No, I don't care. Security matters more.

**MCP for tools** — Model Context Protocol is becoming the standard for LLM tools. Why reinvent the wheel? Just use MCP servers.

**No agent mailboxes yet** — I went with the simplest thing that could work. Async communication is planned, but I wanted to ship something that actually works before I built the fancy stuff.

## Program Structure

A markdownlang program is dead simple:

1. **YAML Front Matter** — Metadata and configuration between `---` delimiters
2. **Markdown Content** — Instructions for the LLM, with Go template interpolation

Here's what the front matter looks like:

```yaml
---
name: my-program # Required: Program identifier
description: Does stuff # Required: Human-readable description
input: # Required: JSON Schema for input validation
  type: object
  properties:
    url: { type: string }
  required: [url]
output: # Required: JSON Schema for output validation
  type: object
  properties:
    summary: { type: string }
  required: [summary]
imports: # Optional: Other .md programs this can call
  - ./helper.md
  - ../tools/analyzer.md
mcp_servers: # Optional: MCP servers for tool access
  - name: filesystem
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "."]
model: gpt-4o # Optional: Override default model
---
```

### Required Fields

- **`name`** — The program's identifier. Used in imports and metrics.
- **`description`** — What the program actually does. Gets included in the system message.
- **`input`** — JSON Schema Draft 2020-12 for input validation.
- **`output`** — JSON Schema Draft 2020-12 for output validation.

### Optional Fields

- **`imports`** — Paths to other markdownlang programs (`.md` files). Imported agents become available as tools. Paths are resolved relative to the importing program. Circular dependencies are detected and rejected because I'm not dealing with that nonsense.
- **`mcp_servers`** — MCP server configurations. Each server has a `name` and either `command`/`args` or `url`. Tools from servers get namespaced as `mcp__{server}__{tool}`.
- **`model`** — Override the default model for this program. Uses the globally configured model if you don't specify one.

## The Markdown Content

The content after the front matter contains instructions for the LLM. It supports Go template interpolation:

```markdown
Hello, {{ .name }}!
You have {{ .count }} messages.
```

Supported syntax:

- `{{ .variable }}` — Variable reference
- `{{ .nested.field }}` — Nested field
- `{{ if .cond }}...{{ end }}` — Conditional
- `{{ range .items }}...{{ end }}` — Loop

Template functions: `upper`, `lower`, `title`, `default`, `len`, `slice`, `join`, `split`.

## A Complete Example

Here's a FizzBuzz agent because of course I'm going to use FizzBuzz:

```markdown
---
name: fizzbuzz
description: Generate FizzBuzz sequence
input:
  type: object
  properties:
    start: { type: integer }
    end: { type: integer }
  required: [start, end]
output:
  type: object
  properties:
    results:
      type: array
      items: { type: string }
  required: [results]
---

For each number from {{ .start }} to {{ .end }}, output:

- "FizzBuzz" if divisible by both 3 and 5
- "Fizz" if divisible by 3
- "Buzz" if divisible by 5
- The number itself otherwise

Return a JSON object with a "results" array containing the strings in order.
```

Run it like this:

```bash
markdownlang -program fizzbuzz.md -input '{"start":1,"end":15}'
```

And you get back valid JSON that matches the output schema. Every single time. Even if the LLM messes up the first few attempts.

## JSON Schema Requirements

The `input` and `output` schemas must follow JSON Schema Draft 2020-12. Define your types. Specify your `required` arrays. Use the supported types: `null`, `boolean`, `object`, `array`, `number`, `integer`, `string`.

Output MUST always be valid JSON matching the output schema. If there's nothing to return, output `{}`. I'm serious about this.

## Agent Execution

### The Agent Loop

markdownlang runs an iterative loop with a maximum of 69 iterations:

```text
┌─────────────────────────────────────────────────────┐
│  1. Render template with input data                  │
│  2. Build system message (description + schema)     │
│  3. Call LLM API (with available tools)             │
│  4. Extract and validate response against schema    │
│  5. If valid: return result                         │
│  6. If invalid: add error feedback, retry (max 69)  │
└─────────────────────────────────────────────────────┘
```

Each iteration sends a request to the LLM with the program's description, the rendered markdown content, available tools, and a strict JSON Schema response format. If the output doesn't validate, we tell the LLM exactly what went wrong and try again.

### Validation and Retry

1. **Input validation** — Before execution, input is validated against the `input` schema.
2. **Output validation** — After LLM response, output is validated against the `output` schema.
3. **Retry with feedback** — If validation fails, the error is included in the next iteration's system message.
4. **Max iterations** — After 69 failed iterations, execution fails with an error.

This feedback loop is the whole reason markdownlang works reliably. The LLM learns from its mistakes.

### Lifecycle

1. **Load program** — Parse and validate the markdown file.
2. **Create context** — Set up tool handlers, agent registry, MCP manager.
3. **Render template** — Substitute `{{ .variable }}` with input values.
4. **Execute loop** — Run iterations until valid output or max retries.
5. **Return result** — Output JSON matching the schema.

## MCP Integration

MCP (Model Context Protocol) is how you connect LLMs to external tools and data sources. markdownlang integrates MCP servers to give agents additional capabilities.

### What is MCP?

Think of MCP as a universal plug standard for LLM tools. Instead of every AI framework inventing its own tool format, we just use MCP servers. Want filesystem access? There's an MCP server for that. Want to run Python? There's an MCP server for that too. (Although markdownlang has a built-in Python interpreter, more on that later.)

### Server Configuration

MCP servers are configured in the front matter's `mcp_servers` array:

```yaml
mcp_servers:
  # Command-based server (stdio transport)
  - name: filesystem
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "."]

  # HTTP-based server (SSE transport)
  - name: python-interpreter
    url: "http://localhost:3000/mcp"
```

Fields:

- `name` (required) — Server identifier
- `command` (required for stdio) — Command to start server
- `args` (optional) — Command arguments
- `env` (optional) — Environment variables
- `url` (required for SSE) — HTTP endpoint URL
- `disabled` (optional) — Don't start this server

### Transport Types

**Command (stdio)** — Executes a command and communicates via stdin/stdout. Requires `command` and `args` fields. Example: `npx -y @modelcontextprotocol/server-filesystem .`

**HTTP (SSE)** — Connects to an HTTP endpoint using Server-Sent Events. Requires `url` field starting with `http://` or `https://`. Example: `http://localhost:3000/mcp`

### Tool Naming

Tools from MCP servers are namespaced to avoid conflicts:

```
mcp__{server_name}__{tool_name}
```

So a `read_file` tool from a `filesystem` server becomes `mcp__filesystem__read_file`.

### Tool Execution

1. **Server starts** — MCP servers are started when execution begins.
2. **Tools listed** — All tools from all servers are collected and converted to JSON Schema.
3. **LLM can call** — Tools are included in the LLM API call's `tools` array.
4. **Results returned** — Tool results are included in the next iteration's conversation.
5. **Servers stop** — MCP servers are stopped when execution completes.

## Python Interpreter

markdownlang includes a built-in Python interpreter for executing code safely. It uses Wazero (a WebAssembly runtime) to sandbox Python execution.

### Security Features

The Python interpreter runs in a secure sandbox with:

- **No network access** — Cannot make HTTP requests or network calls
- **Limited filesystem** — Access only to a temporary directory
- **Timeout** — Default 30 seconds (configurable)
- **Memory limit** — Default 128MB (configurable)
- **Wasm bytecode** — Python code is compiled to WebAssembly before execution

Is it slower than native Python? Yeah. Do I care? No. Security is more important than raw speed for this use case.

### Default Configuration

```
Timeout:   30 seconds
MaxMemory: 128 MB
```

### Tool Interface

The Python interpreter is available as a tool named `python`:

**Input:**

```json
{
  "code": "print('Hello, World!')"
}
```

**Output:**

```json
{
  "stdout": "Hello, World!\n",
  "stderr": "",
  "error": null
}
```

### Usage

Agents can use the Python tool for calculations, data processing, or any computation:

```markdown
---
name: calculate-stats
input:
  type: object
  properties:
    numbers:
      type: array
      items: { type: number }
  required: [numbers]
output:
  type: object
  properties:
    mean: { type: number }
    median: { type: number }
  required: [mean, median]
---

Calculate the mean and median of {{ .numbers }}.
Use the python tool to perform the calculations.
```

### Error Handling

If Python code fails, the error is returned in the `error` field:

```json
{
  "stdout": "",
  "stderr": "Traceback (most recent call last):\n  ...",
  "error": "execution failed: ..."
}
```

## Template System

The markdown content uses Go's `text/template` syntax for interpolation. This lets you dynamically insert input values into the prompt.

### Interpolation Syntax

```markdown
Hello, {{ .name }}!
You have {{ .count }} messages.
```

Supported syntax:

- `{{ .variable }}` — Variable reference
- `{{ .nested.field }}` — Nested field
- `{{ if .cond }}...{{ end }}` — Conditional
- `{{ range .items }}...{{ end }}` — Loop

### Available Functions

| Function  | Description               | Example                  |
| --------- | ------------------------- | ------------------------ | ----------------- |
| `upper`   | Convert to uppercase      | `{{ upper .name }}`      |
| `lower`   | Convert to lowercase      | `{{ lower .name }}`      |
| `title`   | Capitalize words          | `{{ title .name }}`      |
| `default` | Provide default value     | `{{ .value               | default "N/A" }}` |
| `len`     | Get length                | `{{ len .items }}`       |
| `slice`   | Slice array/string        | `{{ slice .items 0 5 }}` |
| `join`    | Join array with separator | `{{ join .items ", " }}` |
| `split`   | Split string by separator | `{{ split .text "," }}`  |

### Examples

```markdown
---
name: greet
input:
  type: object
  properties:
    name: { type: string }
    items: { type: array, items: { type: string } }
    debug: { type: boolean }
  required: [name, items]
---

Hello {{ .name | title }}!

{{ if .debug }}
Debug mode: processing {{ len .items }} files
{{ end }}

You have {{ len .items }} items:
{{ join .items "\n- " }}
```

### Security

The template renderer includes security features:

1. **Map key sanitization** — Only alphanumeric, underscore, and hyphen allowed
2. **Template syntax escaping** — Prevents injection attacks
3. **Recursive sanitization** — Nested structures are sanitized

Don't try to hack the template system. I've tried. It's annoying.

## Agent Composition

Agents can import other agents as tools using the `imports` field. This is how you build complex behaviors from simple pieces.

### Imports

```yaml
imports:
  - ./helper.md
  - ../tools/analyzer.md
  - /absolute/path/to/agent.md
  - stdlib:json-parser # Reserved for future standard library
```

Paths can be relative, absolute, or named (`stdlib:name`). Imports are not recursive — imported programs don't expose their imports. Circular dependencies are errors because I refuse to deal with that chaos.

### Registry Pattern

The `Registry` manages imported agents:

1. **LoadImport** — Loads and parses a program by path
2. **CreateToolHandlers** — Creates tool handlers for imported agents
3. **CallAgent** — Executes an imported agent with input

### Imported Agents as Tools

Each imported agent becomes a tool with:

- **Name** — The agent's `name` field from front matter
- **Input schema** — The agent's `input` schema
- **Execution** — Runs the agent's loop with the provided input
- **Output** — Returns the agent's validated output

### Blocking Execution

Agent calls are **blocking** — the calling agent waits for the imported agent to complete:

```text
Main Agent
    |
    +-- calls fizzbuzz (waits for result)
    |       |
    |       +-- fizzbuzz completes
    |
    +-- uses fizzbuzz result
```

This provides predictable execution and easier debugging. Async mailboxes are planned for later.

### Circular Dependency Detection

The registry detects circular dependencies:

```text
a.md imports [b.md]
b.md imports [c.md]
c.md imports [a.md]  # ERROR: circular dependency detected
```

### Metrics Tracking

Agent calls are tracked with metrics:

```json
{
  "agent_calls": {
    "total_calls": 5,
    "calls_by_agent": {
      "fizzbuzz": 2,
      "word-count": 3
    },
    "total_duration": "5.2s",
    "average_duration": "1.04s",
    "tokens_used": 1250
  }
}
```

### Complete Example

**fizzbuzz.md:**

```markdown
---
name: fizzbuzz
input:
  type: object
  properties:
    start: { type: integer }
    end: { type: integer }
  required: [start, end]
output:
  type: object
  properties:
    results:
      type: array
      items: { type: string }
  required: [results]
---

Generate FizzBuzz from {{ .start }} to {{ .end }}.
```

**word-count.md:**

```markdown
---
name: word-count
input:
  type: object
  properties:
    text: { type: string }
  required: [text]
output:
  type: object
  properties:
    count: { type: integer }
  required: [count]
---

Count the words in: {{ .text }}
```

**fizzbuzz-word-count.md:**

```markdown
---
name: fizzbuzz-word-count
imports:
  - ./fizzbuzz.md
  - ./word-count.md
input:
  type: object
  properties:
    start: { type: integer }
    end: { type: integer }
  required: [start, end]
output:
  type: object
  properties:
    fizzbuzz_results:
      type: array
      items: { type: string }
    total_words: { type: integer }
  required: [fizzbuzz_results, total_words]
---

1. Call the fizzbuzz tool with start={{ .start }} and end={{ .end }}
2. For each result, call word-count to count words
3. Use the python tool to sum all word counts
4. Return both the FizzBuzz results and total word count
```

## CLI and Configuration

### Commands

```bash
markdownlang <command> [flags]
```

| Command | Description                              |
| ------- | ---------------------------------------- |
| `run`   | Execute a markdownlang program (default) |
| `agree` | Accept the trans rights agreement        |

### Run Command Flags

| Flag        | Description                   | Default            |
| ----------- | ----------------------------- | ------------------ |
| `-program`  | Path to the .md file          | _required_         |
| `-input`    | JSON input for the program    | `{}`               |
| `-output`   | Path to write output JSON     | stdout             |
| `-model`    | OpenAI model to use           | `gpt-4o`           |
| `-api-key`  | OpenAI API key                | `$OPENAI_API_KEY`  |
| `-base-url` | LLM base URL                  | `$OPENAI_BASE_URL` |
| `-debug`    | Enable verbose debug logging  | `false`            |
| `-summary`  | Output JSON execution summary | `false`            |
| `-agree`    | Accept agreement (deprecated) | -                  |

### Environment Variables

- `OPENAI_API_KEY` — API key for OpenAI (or compatible service)
- `OPENAI_BASE_URL` — Base URL for API requests

### Validation Rules

1. **Program must exist** — `-program` must point to a valid `.md` file
2. **Input must be valid JSON** — `-input` must parse as valid JSON
3. **Agreement required** — First-time users must run `markdownlang agree`

### Usage Examples

```bash
# Basic execution
markdownlang run -program fizzbuzz.md -input '{"start":1,"end":15}'

# Implicit run (default command)
markdownlang -program fizzbuzz.md -input '{"start":1,"end":15}'

# Save output to file
markdownlang -program agent.md -input '{"data":[1,2,3]}' -output result.json

# With debug logging
markdownlang -program myagent.md -input '{"url":"https://example.com"}' -debug

# With execution summary
markdownlang -program fizzbuzz.md -input '{"start":1,"end":10}' -summary

# Custom model
markdownlang -program agent.md -model claude-3-5-sonnet-20241022
```

## Agreement System

Before first use, users must accept the trans rights agreement. This gate ensures users support transgender rights before using the software.

### Storage

Agreement is stored in `~/.markdownlang-agreement.json`:

```json
{
  "agreed_at": "2024-01-15T10:30:00Z",
  "phrase_index": 7
}
```

### Random Phrase Selection

On first run, a random phrase is selected from 16 pre-defined phrases. Users must type the phrase exactly to accept the agreement.

### Acceptable Phrases

All phrases affirm support for trans rights and commit to not harming transgender people. Examples include:

- "Trans rights are human rights"
- "Trans women are women"
- "Trans men are men"
- "I support transgender people"

### Acceptance Flow

```bash
$ markdownlang agree

markdownlang Trans Rights Agreement

Before using markdownlang, you must agree to the following:

Trans rights are human rights. I will not harm transgender people.

Type the above phrase exactly to accept: [user types phrase]

Agreement accepted. You can now use markdownlang.
```

This is non-negotiable. If you don't support trans rights, this tool isn't for you.

## Execution Metrics and Summaries

### Tracked Metrics

The executor tracks the following metrics:

| Metric          | Description                 |
| --------------- | --------------------------- |
| `iterations`    | Number of loop iterations   |
| `tokens_input`  | Input tokens used           |
| `tokens_output` | Output tokens used          |
| `tokens_total`  | Total tokens used           |
| `tool_calls`    | Number of tool calls made   |
| `duration`      | Execution duration          |
| `errors`        | Number of validation errors |
| `start_time`    | Execution start time        |
| `end_time`      | Execution end time          |

### Execution Summary

When `-summary` flag is used, execution outputs a JSON summary:

```json
{
  "program": "fizzbuzz.md",
  "success": true,
  "iterations": 1,
  "tokens": {
    "input": 150,
    "output": 75,
    "total": 225,
    "cost": 0.0001125
  },
  "tools_called": 0,
  "duration": "1.2s",
  "model": "gpt-4o",
  "agent_calls": {
    "total_calls": 2,
    "calls_by_agent": {
      "fizzbuzz": 1,
      "word-count": 1
    },
    "total_duration": "2.5s",
    "average_duration": "1.25s",
    "tokens_used": 450
  }
}
```

### Token Cost Calculation

Costs are calculated using OpenAI's pricing:

| Model  | Input per 1M tokens | Output per 1M tokens |
| ------ | ------------------- | -------------------- |
| GPT-4o | $2.50               | $10.00               |

Example calculation:

- Input: 150 tokens = $0.000375
- Output: 75 tokens = $0.00075
- Total: $0.001125

### Enabling Summaries

Use the `-summary` flag to enable summary output:

```bash
markdownlang -program agent.md -input '{"data":[1,2,3]}' -summary
```

The summary is written to stderr, while the result goes to stdout or the specified output file.

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
5. **Update template references** — replace placeholders with `{{ .param }}`
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

## Future Expansion: Agent Mailboxes

Currently, agent calls are blocking — callers wait for completion. A future expansion will introduce **mailboxes** for non-blocking inter-agent communication:

- **Send** — Send a message to another agent's mailbox without blocking
- **Receive** — Check mailbox for incoming messages
- **Message format** — JSON with sender, timestamp, and payload
- **Persistence** — Mailboxes persist beyond agent lifetime
- **Querying** — Filter messages by sender, time, or content

This enables asynchronous workflows, fan-out patterns, and long-running agent coordination. I haven't built this yet because I wanted to get the basics right first.

## Implementation Notes

- Input and output MUST be validated against their schemas
- Output MUST always be valid JSON (use `{}` when there's nothing to return)
- Template rendering should prevent injection attacks
- MCP servers start on agent init, stop on completion
- Retry on transient errors (default max 69)
- Limit resource usage and execution time

## References

- **JSON Schema** — Draft 2020-12. Types: string, number, integer, boolean, array, object, null. Validation: required, enum, minimum/maximum, minLength/maxLength, pattern, format
- **Go Templates** — `{{ .var }}`, `{{ if }}`, `{{ range }}`. Functions: upper, lower, title, len, default, slice, join, split
- **MCP** — Model Context Protocol for exposing tools to LLMs. See https://modelcontextprotocol.io

---

**Version:** 1.0.0

_This specification describes markdownlang as implemented in the codebase. The code is law._

If you find bugs or have ideas for improvements, file an issue or send a pull request. I'm actually pretty responsive.
