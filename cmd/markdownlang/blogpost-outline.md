# Humanity's Last Programming Language

## I. Introduction

### Hook

- The premise: "What if markdown were executable?"
- The reality: We're entering an era where LLMs are the new runtime
- A counterintuitive claim: This might be the last programming language humanity creates

### What is markdownlang?

- A markdown-based AI agent language
- Write programs in markdown with YAML front matter
- LLMs execute them and return structured JSON
- "An AI agent language that doesn't suck"

---

## II. The Problem: Agent Framework Hell

### Current State of AI Agents

- Every week: New agent framework, new DSL, new paradigm
- LangChain, AutoGPT, BabyAGI, CrewAI, and counting
- Complexity creep: Endless YAML configs, brittle orchestration, debugging nightmares
- The hype cycle: "AI agents will replace programmers" → Reality: "I can't get my agent to use a tool correctly"

### The Core Issues

1. **Abstraction overload**: Frameworks invent new languages instead of using existing ones
2. **Brittle tool calling**: LLMs struggle to match function signatures
3. **No schema enforcement**: Output is whatever the LLM feels like returning
4. **Debugging hell**: When your agent loops forever, where do you even start?

### A Different Approach

- What if we used the universal documentation format as code?
- What if schemas were first-class citizens, not afterthoughts?
- What if we embraced simplicity over complexity?

---

## III. Motivation: Design Philosophy

### Core Principles

1. **Markdown as the Interface**
   - Everyone knows markdown
   - Edit in any editor, version control normally
   - Read and write by humans, for humans
   - No DSL, no syntax highlighting plugins needed

2. **Schema-First Validation**
   - Input and output schemas defined upfront in JSON Schema
   - The executor validates everything
   - Early failures, clear error messages
   - No silent JSON mismatches

3. **The LLM Does the Work**
   - No code generation
   - The LLM is the execution engine
   - Tools are just tools, not crutches
   - Minimal abstraction between intent and execution

4. **Composability**
   - Agents can import other agents
   - Build hierarchical systems from simple components
   - Circular dependency detection
   - Metrics tracking for optimization

5. **Type Safety Through Schemas**
   - JSON Schema Draft 2020-12
   - Output MUST match schema
   - Retry loop with error feedback
   - Max 69 iterations prevents infinite loops

### The "Doesn't Suck" Manifesto

- Minimal features, maximal reliability
- Explicit over implicit
- Blocking calls for predictable execution
- No magic, no hidden behavior

---

## IV. How It Works: Implementation Details

### Program Structure

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
```

### The Execution Loop

```
┌─────────────────────────────────────────────────────┐
│  1. Parse YAML front matter                          │
│  2. Validate input against schema                    │
│  3. Render template with Go templates                │
│  4. Call LLM with structured output (JSON Schema)    │
│  5. Validate output against schema                   │
│  6. If valid: return result                          │
│  7. If invalid: add error feedback, retry (max 69)  │
└─────────────────────────────────────────────────────┘
```

### Key Implementation Components

1. **Parser** (`internal/parser`)
   - Extracts YAML front matter
   - Validates JSON Schema
   - Detects circular dependencies in imports

2. **Executor** (`internal/executor`)
   - Orchestrates execution
   - Sets up tools (MCP, Python, imported agents)
   - Manages lifecycle

3. **Agent Loop** (`internal/agent`)
   - Max 69 iterations
   - Calls OpenAI Responses API v3
   - Validates and retries

4. **MCP Integration** (`internal/mcp`)
   - Manages Model Context Protocol servers
   - Two transports: stdio (command) and SSE (HTTP)
   - Namespaced tools: `mcp__{server}__{tool}`

5. **Python Interpreter** (`internal/python`)
   - Wazero WebAssembly sandbox
   - No network, limited filesystem
   - 30s timeout, 128MB memory limit

6. **Template Renderer** (`internal/template`)
   - Go `text/template`
   - Functions: upper, lower, title, default, len, slice, join, split
   - Security: sanitization and escaping

---

## V. Tool Ecosystem

### MCP Integration

- Model Context Protocol for external tools
- Command-based (stdio) or HTTP (SSE)
- Tools automatically discovered and exposed

### Built-in Python Interpreter

- Wasm sandboxed execution
- Safe for calculations and data processing
- Captured stdout/stderr

### Agent Composition

- Import other `.md` programs
- They become tools automatically
- Blocking execution for predictability

### Example: Composed Agents

```markdown
---
name: summarizer
imports:
  - ./fetcher.md
  - ./analyzer.md
---

1. Use the **fetcher** tool to fetch the URL
2. Use the **analyzer** tool to generate summary
3. Return both the content and summary
```

---

## VI. The Agreement System

### Why It Exists

- Not a joke: Trans rights are human rights
- Gates first-time use behind explicit agreement
- Random phrase selection prevents automation

### How It Works

- Stored in `~/.markdownlang-agreement.json`
- 16 pre-defined phrases
- Must type exactly

This is a non-negotiable part of markdownlang's philosophy.

---

## VII. Limitations and Trade-offs

### Current Limitations

1. **Blocking Agent Calls Only**
   - No async communication
   - No fan-out patterns
   - Planned: Agent mailboxes for future

2. **OpenAI-Dependent**
   - Uses OpenAI Responses API v3
   - Structured outputs require compatible models
   - Could be extended to other providers

3. **Max 69 Iterations**
   - Prevents infinite loops
   - May not be enough for complex tasks
   - Hard-coded limit

4. **No Native Control Flow**
   - No if/else, loops, or conditionals in the language itself
   - The LLM must figure this out
   - Can be unreliable

5. **Stdlib Not Implemented**
   - `stdlib:name` import paths reserved
   - Not yet built
   - Would enable shared agent library

### Intentional Simplifications

1. **No Code Generation**
   - markdownlang doesn't generate code
   - It executes prompts through an agent loop
   - The LLM returns structured output directly

2. **No Agent Mailboxes (Yet)**
   - Simplest-first design
   - Blocking calls are predictable
   - Async planned for future

3. **No Custom Runtime**
   - Leverages LLM APIs directly
   - No middleware or orchestration layer
   - Reduces surface area

---

## VIII. Why "Humanity's Last Programming Language"?

### The Thesis

1. **LLMs as Universal Runtime**
   - Natural language is becoming executable
   - Schemas provide the guardrails
   - The "programming language" is just description

2. **Abstraction Direction**
   - Most languages: Move toward machine (C → Rust → Assembly)
   - markdownlang: Move toward human (Python → English → Intent)
   - The final abstraction: Describe what you want, get what you asked for

3. **The End of Syntax**
   - No more fighting semicolons
   - No more memorizing APIs
   - No more compiler errors
   - Just: Here's what I want, here's the shape of the answer

4. **But It's Still Programming**
   - Schemas are the new types
   - Imports are the new dependencies
   - Composition is the new architecture
   - Debugging is... still debugging

### The Counterpoint: It's Not That Deep

- markdownlang is just a tool
- LLMs still make mistakes
- Schemas can't express everything
- Someone still needs to write the agents

---

## IX. Future Directions

### Near Term

- Agent mailboxes for async communication
- Standard library of common agents
- Support for more LLM providers
- Better error messages and debugging tools

### Long Term

- Agent marketplace / registry
- Visual editor for building composed agents
- Performance optimization (caching, parallel execution)
- Formal verification of agent behaviors

---

## X. Conclusion

### Recap

- markdownlang: Markdown + YAML + JSON Schema + LLM = Executable
- Philosophy: Minimal abstraction, schema-first, composable
- Implementation: ~2000 lines of Go, clean architecture
- Limitations: Intentional simplicity, blocking calls only

### The Vision

- Programming is becoming description
- Schemas are the new type system
- Agents are the new functions
- The LLM is the new CPU

### Final Thought

Maybe markdownlang isn't humanity's last programming language. But it points toward a future where the boundary between "describing what you want" and "programming" disappears. And that future is written in markdown.

---

## XI. Call to Action

- Try it: `go install within.website/x/cmd/markdownlang@latest`
- Read the spec: `cmd/markdownlang/docs/SPEC.md`
- Join the discussion: [GitHub Issues]
- Build an agent: Start with FizzBuzz, end with something useful
