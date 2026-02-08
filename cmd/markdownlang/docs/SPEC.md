# markdownlang Specification

## Overview

markdownlang is an AI agent language that doesn't suck. It lets you write executable programs in markdown with YAML front matter. LLMs (specifically OpenAI's Chat Completions API) execute these programs and return structured JSON output matching a schema.

### Core Philosophy

- **Markdown as the interface**: Programs are written in markdown, the universal documentation format
- **Minimal abstraction**: The LLM does the actual work, no complex DSL to learn
- **Type safety through JSON Schema**: Input and output are validated against schemas
- **Composable**: Agents can import and call other agents as tools
- **Tool ecosystem**: Integrates with MCP (Model Context Protocol) for external tools

### High-Level Architecture

```
markdownlang program (.md)
       |
       v
    Parser (YAML front matter + JSON Schema validation)
       |
       v
    Executor (orchestrates execution)
       |
       v
    Agent Loop (max 10 iterations)
       |
       +-- LLM API Call (OpenAI Responses API v3)
       +-- Tool Execution (MCP servers, Python interpreter, imported agents)
       +-- Schema Validation (input/output)
       |
       v
    Result (JSON matching output schema)
```

### Key Design Decisions

1. **Markdown + YAML**: Simplicity over full DSL - use existing formats
2. **Iterative LLM loop**: Guarantees output schema compliance through validation feedback
3. **Blocking agent calls**: Predictable execution, easier debugging (mailboxes planned for future)
4. **Wasm Python**: Security over performance for code execution
5. **MCP as tools**: Standardized tool ecosystem integration
6. **No agent mailboxes yet**: Simplest-first design, async communication planned

## Program Structure

A markdownlang program consists of two parts:

1. **YAML Front Matter**: Metadata and configuration between `---` delimiters
2. **Markdown Content**: Instructions for the LLM, with Go template interpolation

### Front Matter Fields

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

#### Required Fields

- **`name`**: Program identifier (used in imports and metrics)
- **`description`**: What the program does (included in system message)
- **`input`**: JSON Schema Draft 2020-12 for input validation
- **`output`**: JSON Schema Draft 2020-12 for output validation

#### Optional Fields

- **`imports`**: Array of paths to other markdownlang programs (.md files)
  - Imported agents become available as tools
  - Paths are resolved relative to the importing program
  - Circular dependencies are detected and rejected
- **`mcp_servers`**: Array of MCP server configurations
  - Each server has a `name` and either `command`/`args` or `url`
  - Tools from servers are namespaced as `mcp__{server}__{tool}`
- **`model`**: Override the default model for this program
  - Uses the globally configured model if not specified

### Markdown Content

The content after the front matter contains instructions for the LLM. It supports:

- **Go template interpolation**: `{{ .variable }}` syntax
- **Template functions**: `upper`, `lower`, `title`, `default`, `len`, `slice`, `join`, `split`
- **Arbitrary markdown**: Any markdown formatting is valid

### Complete Example

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

### JSON Schema Requirements

The `input` and `output` schemas must:

1. Follow JSON Schema Draft 2020-12
2. Include the `$schema` keyword (optional but recommended)
3. Define `type` for all schemas
4. Specify `required` arrays for objects when needed
5. Use supported types: `null`, `boolean`, `object`, `array`, `number`, `integer`, `string`

## Agent Execution

### The Agent Loop

markdownlang uses an iterative agent loop with a maximum of 10 iterations:

```
┌─────────────────────────────────────────────────────┐
│  1. Render template with input data                  │
│  2. Build system message (description + schema)     │
│  3. Call LLM API (with available tools)             │
│  4. Extract and validate response against schema    │
│  5. If valid: return result                         │
│  6. If invalid: add error feedback, retry (max 10)  │
└─────────────────────────────────────────────────────┘
```

### Request Construction

Each iteration sends:

```json
{
  "model": "gpt-4o",
  "system": "Program description and schema instructions",
  "messages": [
    {
      "role": "user",
      "content": "Rendered markdown content with interpolated values"
    }
  ],
  "tools": [...],
  "response_format": {
    "type": "json_schema",
    "json_schema": {
      "name": "output",
      "strict": true,
      "schema": <output schema from front matter>
    }
  }
}
```

### Validation and Retry

1. **Input validation**: Before execution, input is validated against the `input` schema
2. **Output validation**: After LLM response, output is validated against the `output` schema
3. **Retry with feedback**: If validation fails, the error is included in the next iteration's system message
4. **Max iterations**: After 10 failed iterations, execution fails with an error

**Output MUST always be valid JSON** matching the output schema. If there's nothing to return, output `{}`.

### Lifecycle

1. **Load program**: Parse and validate the markdown file
2. **Create context**: Set up tool handlers, agent registry, MCP manager
3. **Render template**: Substitute `{{ .variable }}` with input values
4. **Execute loop**: Run iterations until valid output or max retries
5. **Return result**: Output JSON matching the schema

## MCP Integration

### What is MCP?

MCP (Model Context Protocol) is a protocol for connecting LLMs to external tools and data sources. markdownlang integrates MCP servers to provide agents with additional capabilities.

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

- `name` (required): Server identifier
- `command` (required for stdio): Command to start server
- `args` (optional): Command arguments
- `env` (optional): Environment variables
- `url` (required for SSE): HTTP endpoint URL
- `disabled` (optional): Don't start this server

### Transport Types

1. **Command (stdio)**: Executes a command and communicates via stdin/stdout
   - Requires `command` and `args` fields
   - Example: `npx -y @modelcontextprotocol/server-filesystem .`

2. **HTTP (SSE)**: Connects to an HTTP endpoint using Server-Sent Events
   - Requires `url` field starting with `http://` or `https://`
   - Example: `http://localhost:3000/mcp`

### Tool Naming

Tools from MCP servers are namespaced to avoid conflicts:

```
mcp__{server_name}__{tool_name}
```

For example, a `read_file` tool from a `filesystem` server becomes:

```
mcp__filesystem__read_file
```

### Tool Execution

1. **Server starts**: MCP servers are started when execution begins
2. **Tools listed**: All tools from all servers are collected and converted to JSON Schema
3. **LLM can call**: Tools are included in the LLM API call's `tools` array
4. **Results returned**: Tool results are included in the next iteration's conversation
5. **Servers stop**: MCP servers are stopped when execution completes

### Example

```markdown
---
name: file-reader
mcp_servers:
  - name: filesystem
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/dir"]
input:
  type: object
  properties:
    filename: { type: string }
  required: [filename]
output:
  type: object
  properties:
    content: { type: string }
  required: [content]
---

Read the file {{ .filename }} using the mcp**filesystem**read_file tool
and return its content.
```

## Python Interpreter

### Overview

markdownlang includes a built-in Python interpreter for executing code safely. It uses Wazero (a WebAssembly runtime) to sandbox Python execution.

### Security Features

The Python interpreter runs in a secure sandbox with:

- **No network access**: Cannot make HTTP requests or network calls
- **Limited filesystem**: Access only to a temporary directory
- **Timeout**: Default 30 seconds (configurable)
- **Memory limit**: Default 128MB (configurable)
- **Wasm bytecode**: Python code is compiled to WebAssembly before execution

### Default Configuration

```go
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

**Special servers:**

- `python-interpreter`: Prefer for calculations, data processing, algorithms

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

### Interpolation Syntax

The markdown content uses Go's `text/template` syntax for interpolation:

```markdown
Hello, {{ .name }}!
You have {{ .count }} messages.
```

Supported syntax:

- `{{ .variable }}` - Variable reference
- `{{ .nested.field }}` - Nested field
- `{{ if .cond }}...{{ end }}` - Conditional
- `{{ range .items }}...{{ end }}` - Loop

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
Debug mode: processing {{ .items | len }} files
{{ end }}

You have {{ len .items }} items:
{{ join .items "\n- " }}
```

### Security

The template renderer includes security features:

1. **Map key sanitization**: Only alphanumeric, underscore, and hyphen allowed
2. **Template syntax escaping**: Prevents injection attacks
3. **Recursive sanitization**: Nested structures are sanitized

## Agent Composition

### Imports

Agents can import other agents as tools using the `imports` field:

```yaml
imports:
  - ./helper.md
  - ../tools/analyzer.md
  - /absolute/path/to/agent.md
  - stdlib:json-parser # Reserved for future standard library
```

Paths can be relative, absolute, or named (`stdlib:name`). Imports are not recursive - imported programs don't expose their imports. Circular dependencies are errors.

### Registry Pattern

The `Registry` manages imported agents:

1. **LoadImport**: Loads and parses a program by path
2. **CreateToolHandlers**: Creates tool handlers for imported agents
3. **CallAgent**: Executes an imported agent with input

### Imported Agents as Tools

Each imported agent becomes a tool with:

- **Name**: The agent's `name` field from front matter
- **Input schema**: The agent's `input` schema
- **Execution**: Runs the agent's loop with the provided input
- **Output**: Returns the agent's validated output

### Blocking Execution

Agent calls are **blocking** - the calling agent waits for the imported agent to complete:

```
Main Agent
    |
    +-- calls fizzbuzz (waits for result)
    |       |
    |       +-- fizzbuzz completes
    |
    +-- uses fizzbuzz result
```

This provides predictable execution and easier debugging.

### Circular Dependency Detection

The registry detects circular dependencies:

```
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

```
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

- `OPENAI_API_KEY`: API key for OpenAI (or compatible service)
- `OPENAI_BASE_URL`: Base URL for API requests

### Validation Rules

1. **Program must exist**: `-program` must point to a valid `.md` file
2. **Input must be valid JSON**: `-input` must parse as valid JSON
3. **Agreement required**: First-time users must run `markdownlang agree`

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

### Purpose

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

### Verification

Before any program execution, markdownlang checks for agreement:

```go
if err := agreement.Check(); err != nil {
    fmt.Fprintln(os.Stderr, "\n"+err.Error())
    fmt.Fprintln(os.Stderr, "\nTo accept the agreement, run: markdownlang agree")
    os.Exit(1)
}
```

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

### Agent Call Metrics

When agents call other agents, detailed metrics are tracked:

```json
{
  "agent_calls": {
    "total_calls": 5,
    "calls_by_agent": {
      "helper": 3,
      "analyzer": 2
    },
    "total_duration": "5.2s",
    "average_duration": "1.04s",
    "tokens_used": 1250
  }
}
```

### Enabling Summaries

Use the `-summary` flag to enable summary output:

```bash
markdownlang -program agent.md -input '{"data":[1,2,3]}' -summary
```

The summary is written to stderr, while the result goes to stdout or the specified output file.

### Error Summaries

Even on failure, summaries are produced if `-summary` is enabled:

```json
{
  "program": "broken.md",
  "success": false,
  "error": "validation failed: ...",
  "iterations": 10,
  "tokens": {
    "input": 500,
    "output": 250,
    "total": 750,
    "cost": 0.00375
  },
  "duration": "12.5s",
  "model": "gpt-4o"
}
```

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

## Future Expansion: Agent Mailboxes

Currently, agent calls are blocking - callers wait for completion. A future expansion will introduce **mailboxes** for non-blocking inter-agent communication:

- **Send** - Send a message to another agent's mailbox without blocking
- **Receive** - Check mailbox for incoming messages
- **Message format** - JSON with sender, timestamp, and payload
- **Persistence** - Mailboxes persist beyond agent lifetime
- **Querying** - Filter messages by sender, time, or content

This enables asynchronous workflows, fan-out patterns, and long-running agent coordination.

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

_This specification describes markdownlang as implemented in the codebase. The code is law._
