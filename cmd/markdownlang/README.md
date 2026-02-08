# markdownlang

An AI agent language that doesn't suck.

> "The code you write is the code you deserve." - probably someone smarter than me

markdownlang is a markdown-based AI agent language. You write markdown files, and
AI agents make them do things. It's like having a very expensive intern who
actually reads documentation.

## Why

Because existing AI agent frameworks are over-engineered nightmares built by
people who think adding more abstraction layers makes them smarter. We just
want to write markdown and have LLMs do the rest.

Also, I was bored and wanted to see if I could build an entire programming
language out of markdown and spite.

## Features

- **Markdown-based**: Programs are just markdown files with YAML front matter
- **Type-safe**: JSON Schema validation for inputs and outputs
- **Composable**: Agents can call other agents (recursively, if you're brave)
- **Tool support**: MCP (Model Context Protocol) for external tools
- **Python**: Built-in Python interpreter running in wasm (because security matters)
- **Snark**: Error messages that will hurt your feelings

## Installation

```bash
go install within.website/x/cmd/markdownlang@latest
```

Or build it yourself like a real programmer:

```bash
git clone https://github.com/Xe/x.git
cd x/cmd/markdownlang
go build
```

## Usage

### Basic Example

Create a file `fizzbuzz.md`:

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

For each number from {{ .start }} to {{ .end }}:

- "FizzBuzz" if divisible by 3 and 5
- "Fizz" if divisible by 3
- "Buzz" if divisible by 5
- The number itself
```

Run it:

```bash
markdownlang -program fizzbuzz.md -input '{"start":1,"end":15}'
```

Output:

```json
{
  "results": [
    "1",
    "2",
    "Fizz",
    "4",
    "Buzz",
    "Fizz",
    "7",
    "8",
    "Fizz",
    "Buzz",
    "11",
    "Fizz",
    "13",
    "14",
    "FizzBuzz"
  ]
}
```

### With Python

```markdown
---
name: calculate-primes
input:
  type: object
  properties:
    limit: { type: integer }
  required: [limit]
output:
  type: object
  properties:
    primes:
      type: array
      items: { type: integer }
  required: [primes]
mcp_servers:
  - name: python-interpreter
    command: uvx
    args: ["mcp-server-python-interpreter"]
---

Use the Python interpreter to calculate all prime numbers up to {{ .limit }}.
```

## Configuration

### Flags

- `-program`: Path to the markdownlang program (required)
- `-input`: JSON input string (default: `{}`)
- `-output`: Output file path (default: stdout)
- `-model`: OpenAI model (default: `gpt-4o`)
- `-api-key`: OpenAI API key (default: `$OPENAI_API_KEY`)
- `-base-url`: LLM base URL (default: `$OPENAI_BASE_URL`)
- `-debug`: Enable verbose logging
- `-summary`: Output JSON execution summary with metrics

### Environment Variables

```bash
export OPENAI_API_KEY="sk-..."
export OPENAI_BASE_URL="https://api.openai.com/v1"  # or your local LLM
```

### Agent Imports and Calls

Agents can import and call other agents, making them composable building blocks.

Create `fizzbuzz.md`:

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

For each number from {{ .start }} to {{ .end }}:

- "FizzBuzz" if divisible by 3 and 5
- "Fizz" if divisible by 3
- "Buzz" if divisible by 5
- The number itself
```

Create `word-count.md`:

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
    word_count: { type: integer }
    character_count: { type: integer }
    longest_word: { type: string }
  required: [word_count, character_count, longest_word]
---

Count the statistics of the given text:

1. Count the number of words (separated by whitespace)
2. Count the number of characters (including spaces)
3. Find the longest word

Text: {{ .text }}
```

Create `fizzbuzz-word-count.md` that uses both:

```markdown
---
name: fizzbuzz-word-count
description: Runs FizzBuzz on a range and then counts the total words across all results
input:
  type: object
  properties:
    start: { type: integer }
    end: { type: integer }
  required: [start, end]
output:
  type: object
  properties:
    total_words: { type: integer }
    total_characters: { type: integer }
    results: { type: array, items: { type: string } }
  required: [total_words, total_characters, results]
imports:
  - ./fizzbuzz.md
  - ./word-count.md
---

First, run fizzbuzz to get all results from {{ .start }} to {{ .end }}.

Then, for each result, count its words and characters.

Finally, sum up all the word counts and character counts into totals.

Return the totals along with the original fizzbuzz results array.
```

### Execution Summary

Use the `-summary` flag to get detailed metrics about your agent execution:

```bash
markdownlang -program fizzbuzz.md -input '{"start":1,"end":15}' -summary
```

Output includes:

```json
{
  "program": "fizzbuzz.md",
  "success": true,
  "iterations": 1,
  "tokens": {
    "total": 42,
    "input": 20,
    "output": 22,
    "cost": 0.0003
  },
  "tools_called": 0,
  "duration": "1.2s",
  "model": "gpt-4o"
}
```

For agents that call other agents, the summary includes:

```json
{
  "agent_calls": {
    "total_calls": 2,
    "calls_by_agent": {
      "fizzbuzz": 1,
      "word-count": 1
    },
    "total_duration": "2.5s",
    "average_duration": "1.25s",
    "tokens_used": 84
  }
}
```

## Language Reference

See [SPEC.md](docs/SPEC.md) for the full specification. If you don't read it,
don't expect me to explain why your programs don't work.

### Program Structure

Every markdownlang program has:

1. **Front matter** (YAML between `---` delimiters)
   - `name`: Program identifier
   - `description`: What it does
   - `input`: JSON Schema for input validation
   - `output`: JSON Schema for output validation
   - `imports`: Other programs this can call
   - `mcp_servers`: MCP servers for tools

2. **Description** (markdown content)
   - Tells the LLM what to do
   - Supports Go templates: `{{ .variable }}`
   - Template functions: `upper`, `lower`, `title`, `default`, `len`, `slice`, `join`, `split`

### Template Functions

```
{{ .variable }}        - Variable reference
{{ .nested.field }}    - Nested field access
{{ upper .name }}      - Convert to uppercase
{{ .value | default "fallback" }} - Default value
{{ len .items }}       - Get length
{{ .items | slice 0 5 }} - Slice array/string
```

## Architecture

The system consists of:

1. **Parser**: Extracts YAML front matter from markdown
2. **Agent Loop**: Iterates with LLM until output matches schema
3. **Template Renderer**: Interpolates input into description
4. **MCP Client**: Manages tool servers
5. **Python Interpreter**: Wasm-based Python execution

Because who doesn't want to run Python in a wasm sandbox inside their Go
program that's calling an LLM? We live in the future, I guess.

## Development

### Running Tests

```bash
go test ./...
```

### Code Style

- Tabs for indentation (because spaces are for people who hate their wrists)
- `camelCase` for variables
- `PascalCase` for exported types
- Snarky error messages (because they're more memorable)

## Contributing

Contributions are welcome, but:

1. Don't be a bigot (see LICENSE)
2. Don't add unnecessary complexity
3. Keep error messages snarky
4. Write tests for new features
5. Be excellent to each other

If your PR adds corporate over-engineering, I will reject it with extreme
prejudice.

## License

Be Gay Do Crimes License (BSD-3-Clause-Be-Gay-Do-Crimes)

See [LICENSE](LICENSE) for details. TL;DR: Do whatever you want with this code,
but don't be a bigot. Trans rights are human rights.

## Acknowledgments

- OpenAI for the API that makes this possible
- The MCP protocol for tool standardization
- Wasm for making Python safe(r)
- Corporate AI frameworks for showing us what NOT to do

## Author

Built with [Claude Code](https://claude.ai/code) and a team of AI agents.
If that doesn't concern you, it should.

---

<3,

Xe Iaso and the markdownlang contributors

P.S. If you're still reading this, you clearly have too much time on your
hands. Go write a markdownlang program or something.
