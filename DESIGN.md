---
version: alpha
name: Xe Iaso Design System
description: Warm Gruvbox-rooted personal blog system — parchment surfaces, serif headlines, sticker-driven voice, magenta invert link hovers. Derived from xeiaso.net.
colors:
  # Signature
  primary: "#af3a03" # muted orange — WCAG-safe CTA surface
  # Surfaces (light)
  bg-hard: "#f9f5d7"
  bg-soft: "#f2e5bc"
  bg-0: "#fbf1c7"
  bg-1: "#ebdbb2"
  bg-2: "#d5c4a1"
  bg-3: "#bdae93"
  bg-4: "#a89984"
  # Foreground (light)
  fg-0: "#282828"
  fg-1: "#3c3836"
  fg-2: "#504945"
  fg-3: "#665c54"
  fg-4: "#7c6f64"
  # Accents — muted (for surfaces against cream body)
  red: "#9d0006"
  green: "#79740e"
  yellow: "#b57614"
  blue: "#076678"
  purple: "#8f3f71"
  aqua: "#427b58"
  orange: "#af3a03"
  # Accents — bright (for text against muted surfaces, or body text in admonitions)
  red-bright: "#cc241d"
  green-bright: "#98971a"
  yellow-bright: "#d79921"
  blue-bright: "#458588"
  purple-bright: "#b16286"
  aqua-bright: "#689d6a"
  orange-bright: "#d65d0e"
  # Link system — signature magenta invert on hover
  link: "#b80050"
  link-hover: "#fdf4c1"
  link-hover-bg: "#9e0045"
  link-visited: "#53493c"
  link-visited-hover: "#ffffff"
  link-visited-hover-bg: "#282828"
  # Code block (always dark, both modes)
  code-bg: "#1d2021"
  code-fg: "#ebdbb2"

typography:
  display:
    fontFamily: Podkova
    fontSize: 3rem
    fontWeight: 600
    lineHeight: 1.2
  h1:
    fontFamily: Podkova
    fontSize: 2.25rem
    fontWeight: 600
    lineHeight: 1.2
  h2:
    fontFamily: Podkova
    fontSize: 1.875rem
    fontWeight: 600
    lineHeight: 1.2
  h3:
    fontFamily: Podkova
    fontSize: 1.5rem
    fontWeight: 600
    lineHeight: 1.2
  h4:
    fontFamily: Podkova
    fontSize: 1.25rem
    fontWeight: 600
    lineHeight: 1.2
  h5:
    fontFamily: Podkova
    fontSize: 1.125rem
    fontWeight: 600
    lineHeight: 1.2
  h6:
    fontFamily: Podkova
    fontSize: 1rem
    fontWeight: 600
    lineHeight: 1.2
    letterSpacing: 0.04em
  body:
    fontFamily: Schibsted Grotesk
    fontSize: 1rem
    fontWeight: 400
    lineHeight: 1.55
  body-emphasis:
    fontFamily: Schibsted Grotesk
    fontSize: 1rem
    fontWeight: 600
    lineHeight: 1.55
  small:
    fontFamily: Schibsted Grotesk
    fontSize: 0.875rem
    fontWeight: 400
    lineHeight: 1.5
  code:
    fontFamily: Iosevka Curly Iaso
    fontSize: 0.95em
    fontWeight: 400

rounded:
  xs: 2px
  sm: 4px
  md: 6px
  lg: 8px
  xl: 12px

spacing:
  "1": 4px
  "2": 8px
  "3": 12px
  "4": 16px
  "5": 24px
  "6": 32px
  "7": 48px
  "8": 64px

components:
  # Page & surface ramp
  page:
    backgroundColor: "{colors.bg-hard}"
    textColor: "{colors.fg-1}"
    typography: "{typography.body}"
  surface-raised:
    backgroundColor: "{colors.bg-0}"
    textColor: "{colors.fg-1}"
    rounded: "{rounded.md}"
    padding: "{spacing.4}"
  divider-strong:
    backgroundColor: "{colors.bg-3}"
    height: 2px
  border-hairline:
    backgroundColor: "{colors.bg-4}"
    height: 1px

  # Text scale
  text-strong:
    textColor: "{colors.fg-0}"
    typography: "{typography.body-emphasis}"
  text-subtle:
    textColor: "{colors.fg-2}"
    typography: "{typography.body}"
  text-muted:
    textColor: "{colors.fg-3}"
    typography: "{typography.small}"
  text-caption:
    textColor: "{colors.fg-4}"
    typography: "{typography.small}"

  # Buttons — muted hues for WCAG AA on white text
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "#ffffff"
    typography: "{typography.body-emphasis}"
    rounded: "{rounded.xl}"
    padding: 8px 16px
  button-secondary:
    backgroundColor: "{colors.fg-0}"
    textColor: "{colors.bg-hard}"
    typography: "{typography.body-emphasis}"
    rounded: "{rounded.xl}"
    padding: 8px 16px
  button-accent:
    backgroundColor: "{colors.purple}"
    textColor: "#ffffff"
    typography: "{typography.body-emphasis}"
    rounded: "{rounded.xl}"
    padding: 8px 16px
  button-ghost:
    backgroundColor: transparent
    textColor: "{colors.fg-1}"
    typography: "{typography.body-emphasis}"
    rounded: "{rounded.xl}"
    padding: 8px 16px
  button-danger:
    backgroundColor: "{colors.red}"
    textColor: "#ffffff"
    typography: "{typography.body-emphasis}"
    rounded: "{rounded.xl}"
    padding: 8px 16px

  # Content primitives
  card:
    backgroundColor: "{colors.bg-2}"
    textColor: "{colors.fg-1}"
    rounded: "{rounded.md}"
    padding: "{spacing.4}"
  tag:
    backgroundColor: "{colors.bg-1}"
    textColor: "{colors.fg-1}"
    rounded: "{rounded.lg}"
    padding: 6px 10px
    typography: "{typography.small}"
  details:
    backgroundColor: "{colors.bg-soft}"
    textColor: "{colors.fg-1}"
    rounded: "{rounded.md}"
    padding: "{spacing.4}"
  pre:
    backgroundColor: "{colors.code-bg}"
    textColor: "{colors.code-fg}"
    rounded: "{rounded.md}"
    padding: "{spacing.4}"
    typography: "{typography.code}"
  code-inline:
    backgroundColor: "{colors.bg-1}"
    textColor: "{colors.fg-1}"
    rounded: "{rounded.sm}"
    padding: 0.1em 0.3em
    typography: "{typography.code}"
  blockquote:
    backgroundColor: "{colors.bg-2}"
    textColor: "{colors.fg-1}"
    rounded: "{rounded.lg}"
    padding: "{spacing.4}"
  pullquote:
    backgroundColor: "{colors.bg-2}"
    textColor: "{colors.fg-1}"
    rounded: "{rounded.md}"
    padding: "{spacing.4}"

  # Chat bubble ("Conv") pattern
  chat-avatar:
    backgroundColor: "{colors.bg-1}"
    rounded: "{rounded.xs}"
    size: 64px
  chat-row:
    backgroundColor: "{colors.bg-soft}"
    textColor: "{colors.fg-1}"
    padding: "{spacing.3}"

  # Links
  link:
    textColor: "{colors.link}"
  link-hover:
    backgroundColor: "{colors.link-hover-bg}"
    textColor: "{colors.link-hover}"
  link-visited:
    textColor: "{colors.link-visited}"
  link-visited-hover:
    backgroundColor: "{colors.link-visited-hover-bg}"
    textColor: "{colors.link-visited-hover}"

  # Admonitions — bg-soft body with safe body text; muted hue on the 4px left rule, bright hue on the icon dot
  admonition-info:
    backgroundColor: "{colors.bg-soft}"
    textColor: "{colors.fg-1}"
    rounded: "{rounded.md}"
    padding: "{spacing.4}"
  admonition-info-rule:
    backgroundColor: "{colors.blue}"
    width: 4px
  admonition-info-dot:
    backgroundColor: "{colors.blue-bright}"
    rounded: "{rounded.xs}"
    size: 8px
  admonition-warning:
    backgroundColor: "{colors.bg-soft}"
    textColor: "{colors.fg-1}"
    rounded: "{rounded.md}"
    padding: "{spacing.4}"
  admonition-warning-rule:
    backgroundColor: "{colors.yellow}"
    width: 4px
  admonition-warning-dot:
    backgroundColor: "{colors.yellow-bright}"
    rounded: "{rounded.xs}"
    size: 8px
  admonition-tip:
    backgroundColor: "{colors.bg-soft}"
    textColor: "{colors.fg-1}"
    rounded: "{rounded.md}"
    padding: "{spacing.4}"
  admonition-tip-rule:
    backgroundColor: "{colors.green}"
    width: 4px
  admonition-tip-dot:
    backgroundColor: "{colors.green-bright}"
    rounded: "{rounded.xs}"
    size: 8px
  admonition-note:
    backgroundColor: "{colors.bg-soft}"
    textColor: "{colors.fg-1}"
    rounded: "{rounded.md}"
    padding: "{spacing.4}"
  admonition-note-rule:
    backgroundColor: "{colors.purple}"
    width: 4px
  admonition-note-dot:
    backgroundColor: "{colors.purple-bright}"
    rounded: "{rounded.xs}"
    size: 8px
  admonition-danger:
    backgroundColor: "{colors.bg-soft}"
    textColor: "{colors.fg-1}"
    rounded: "{rounded.md}"
    padding: "{spacing.4}"
  admonition-danger-rule:
    backgroundColor: "{colors.red}"
    width: 4px
  admonition-danger-dot:
    backgroundColor: "{colors.red-bright}"
    rounded: "{rounded.xs}"
    size: 8px
  admonition-success:
    backgroundColor: "{colors.bg-soft}"
    textColor: "{colors.fg-1}"
    rounded: "{rounded.md}"
    padding: "{spacing.4}"
  admonition-success-rule:
    backgroundColor: "{colors.aqua}"
    width: 4px
  admonition-success-dot:
    backgroundColor: "{colors.aqua-bright}"
    rounded: "{rounded.xs}"
    size: 8px

  # Sponsor / inline accent text — orange reads warm against the cream page
  sponsor-heading:
    textColor: "{colors.orange}"
    typography: "{typography.h5}"

  # Sponsor card — the only decorative gradient (2px top rule, orange → purple)
  sponsor-rule-start:
    backgroundColor: "{colors.orange-bright}"
    width: 50%
    height: 2px
  sponsor-rule-end:
    backgroundColor: "{colors.purple-bright}"
    width: 50%
    height: 2px
---

## Overview

A design system derived from [xeiaso.net](https://xeiaso.net) — the personal blog and portfolio of **Xe Iaso**, a solo blogger, coder, developer advocate, vtuber, and technical educator. The brand voice is conversational, confident, terminally-online, and unapologetically _personal_. Visually, this is **Gruvbox** (by Morhetz) warm-neutral palette, serif headlines (Podkova) over sans body (Schibsted Grotesk), soft parchment-like surfaces.

**Intended uses.** Long-form blog posts, one-person portfolios, zine-style project pages, fictional-conversation explainers, character-driven technical posts. **Bad fits:** enterprise dashboards, consumer apps, anything that wants to feel "clean modern SaaS."

**Content voice.** First-person and conversational — Xe writes _with_ you, not _at_ you. The tone swings between earnest technical walkthrough and dry shitpost, often in the same sentence. Sentence case for headings; product and brand names preserve their own casing (NixOS, Kubernetes, Tailscale, Anubis). Xe's pronouns are it/its (also they/them). Emoji is used sparingly — stickers of a fictional cast (Mara, Cadey, Aoi, Nicole) do the emotional-signalling work instead.

## Colors

The palette is **Gruvbox** — nothing is a true neutral. Every "gray" is yellow-leaning beige and every accent has both a muted and a bright variant. The two variants exist so light and dark modes can pick opposite ends of the ramp and stay visually balanced.

**Surfaces.** `bg-hard` is the page background (the highest-contrast parchment), `bg-0` the body surface, `bg-soft` sits between them for details and admonitions. `bg-1`→`bg-4` are successively darker/warmer — `bg-1` is the code-inline and tag chip, `bg-2` is the card and blockquote body, `bg-3` is the strong divider, `bg-4` is the hairline border color. On the foreground side, `fg-0` is the highest-contrast heading/link color, `fg-1` is body copy, `fg-2`/`fg-3` are subtle text, `fg-4` is caption/disabled.

**Accent pairs.** Each hue — red, green, yellow, blue, purple, aqua, orange — has a muted (darker) and `-bright` (lighter, more saturated) token. The convention in this system is: **muted hue** for surfaces against the cream body (buttons, admonition rule bars) because muted + white text hits WCAG AA; **bright hue** for small non-text accents (admonition-type dot markers, sponsor gradient endpoints) where the hue pops but contrast doesn't matter because no text sits on it. Admonition _body text_ stays on `fg-1` (body color) — bright hues against warm cream fail contrast. `primary` is an alias of muted orange (`#af3a03`) — the signature CTA and the surface `button-primary` uses.

**Links are a signature.** Light-mode links start at magenta `#b80050` and invert on hover to cream on `#9e0045`. Visited links drop to a desaturated brown and get their own near-black hover. This magenta invert is probably the single most recognisable interaction in the system — do not override it.

**Dark mode.** The live site supports a full dark mode via `prefers-color-scheme`. This spec documents light mode as canonical because DESIGN.md has no native dark-mode slot. In dark mode: surfaces invert to warm charcoal (`#1d2021` hard → `#282828` body → `#3c3836` raised), foreground swaps to cream, and the accent ramp flips — **light mode uses muted hues on surfaces and bright hues on bg-soft text; dark mode swaps, using bright hues on surfaces because they pop on charcoal**. The link magenta shifts to pink `#ffa8ce` / hover `#b3004d`. Implementations should mirror every surface, foreground, and accent token under a `@media (prefers-color-scheme: dark)` block.

**Discipline:** no blue-purple gradients, no neon, no cool grays. The only decorative gradient anywhere is a 2px orange→purple top rule on the SponsorCard (see `sponsor-rule-start` + `sponsor-rule-end`); every other surface is flat and opaque.

## Typography

**Podkova** (serif, weight 400–800, used at 600) for every heading, `h1` through `h6` and any display lockup. It's a warm, slab-ish serif that carries the parchment surfaces.

**Schibsted Grotesk** (sans, variable 400–900) for body copy — 400 for prose, 600 for emphasis. Line height 1.55 in prose, `text-wrap: pretty` on headings and paragraphs.

**Iosevka Curly Iaso** (mono) for code. This is a custom-cut Iosevka variant self-hosted at `files.xeiaso.net`; it is _not_ bundled with this system. Fallbacks are the broader Iosevka Iaso family (Aile / Etoile / Curly) and then `ui-monospace`. The `code` typography token points at the custom face, but any generic mono will read correctly.

**Scale.** 14 / 16 / 18 / 20 / 24 / 30 / 36 / 48 px. `h6` is the only heading that drops to a tight 0.04em tracked, upper-case label style, used sparingly. Everything else is cased naturally.

## Layout

Single column, prose width capped at roughly 65–80ch. Chat-bubble sequences go wider (`~80ch`) so stickers don't squash the text. Vertical rhythm comes from a 16px gap between blocks; no vertical-rhythm grid, just disciplined use of `spacing.4`.

Mobile is not flashy — it just drops to full-width with light horizontal padding. There is no "hamburger reveal" or slide-out nav; the site is flat enough that things fit.

**Prose.** Figures are full-bleed with an italic muted caption centred below. Images use `<picture>` with AVIF/WebP/JPG fallbacks and `loading="lazy"`. The Tailwind Typography plugin decorates `<p>`, `<ul>`, `<figure>`, and `<figcaption>` to match the tokens here; if you're not on Tailwind, the `colors_and_type.css` primitives in the companion skill produce the same result from raw HTML.

## Elevation & Depth

Two shadows, both soft and low:

- `shadow-sm` = `0 1px 2px rgba(40,40,40,.08)` — default for cards, buttons, tags.
- `shadow-md` = `0 2px 6px rgba(40,40,40,.12)` — hover state for interactive elements.

There is no blur, no transparency, no glow. Buttons lift exactly 1px (`translateY(-1px)`) on hover and their shadow grows from sm→md; that is the entire "depth" vocabulary. Transitions are 200ms or less — no bouncing, no spring, no parallax, no scroll-jacking.

**Button hover colour.** Buttons darken or saturate one ramp step on hover (e.g. `button-primary`'s `primary` surface shifts toward `orange-bright`, `button-accent`'s `purple` toward `purple-bright`). Hover states aren't modelled as separate component tokens here — they belong in CSS — but the rule is always "one step warmer or brighter on hover, never lighter to the point of losing the hue."

## Shapes

Radii are deliberate and small.

| Token        | Value | Used for                                                         |
| ------------ | ----- | ---------------------------------------------------------------- |
| `rounded.xs` | 2px   | Sticker avatar frames — boxy on purpose, the sticker is the star |
| `rounded.sm` | 4px   | Inline code                                                      |
| `rounded.md` | 6px   | Cards, `<details>`, `<pre>`, admonitions                         |
| `rounded.lg` | 8px   | Tags, blockquotes                                                |
| `rounded.xl` | 12px  | Pill buttons                                                     |

**Borders.** 1px solid, colour `fg-4` (the `border-hairline` component models this as a 1px `bg-4` strip because DESIGN.md has no `borderColor` slot). No coloured borders anywhere except on semantic admonitions, where a 4px left rule uses the muted accent hue — see `admonition-*-rule` components. No "left-border accent" cards — the only non-admonition block with a left bar is `PullQuote`, and that rule is 4px blue, intentionally.

## Components

**Buttons** come in five flavours: `button-primary` (muted orange, WCAG-safe white text), `button-secondary` (inverted — `fg-0` surface, `bg-hard` text), `button-accent` (muted purple), `button-ghost` (transparent with a 1px border and no shadow), and `button-danger` (muted red for destructive actions). All share a 12px pill radius and 8×16 px padding. _The live site uses `-bright` variants for button defaults and drops to muted on hover; this spec flips that mapping because muted + white text is the only pairing that hits WCAG AA at normal body size. Prefer the spec's mapping in new work._

**Cards** use `bg-2` surface, 6px radius, `spacing.4` padding, and a hairline border (`bg-4`). The text ramp — `text-strong`, `text-subtle`, `text-muted`, `text-caption` — lets card content step from headline to caption without drifting off the fg-0/1/2/3/4 scale.

**Tags** are pill-radius (8px) chips at small body size, on `bg-1` — used for post taxonomies and keyword chips.

**Blockquote** is custom: `bg-2` surface, no left border, prefixed with a literal `>` character — email-style quoting. `PullQuote` is the only non-admonition block with a coloured left rule; use `admonition-info-rule` as a reference for the 4px blue bar.

**Chat bubbles (`Conv`)** are the character-dialogue pattern. `chat-row` rows share a `bg-soft` background; the first rounds top corners, the last rounds bottom, middle rows pull up 1px to form a continuous surface. A 64×64 `chat-avatar` (2px radius, boxy) sits on the left of each row, fed by a sticker URL.

**Admonitions.** Six types — info (blue), warning (yellow), tip (green), note (purple), danger (red), success (aqua). Each is a `bg-soft` card with body text on `fg-1`, a 4px left rule in the muted hue (`admonition-X-rule`), and a small bright-hue marker dot beside the title (`admonition-X-dot`). Body text is _not_ coloured; the hue lives on the rule and the dot so contrast never becomes an issue. Use plain titles "Note", "Warning", "Tip", "Info" — never "👀 Heads up!".

**Stickers.** Character portraits are fetched live from `https://stickers.xeiaso.net/sticker/{character}/{mood}` — not stored anywhere in this repo. Characters include `xe`, `mara`, `cadey`, `nicole`, `aoi`; moods include `aha`, `happy`, `confused`, `coffee`, `wat`. These are the emotional channel of the entire system, replacing what other systems would use emoji for.

**Iconography.** Deliberately low-icon. When you do need one, use [Tabler Icons](https://tabler.io/icons) — 24×24 viewBox, `stroke-width="2"`, `stroke-linecap="round"`, `stroke-linejoin="round"`, `fill="none"`, colour inherits `currentColor`. SponsorCard drops to 20×20. Never invent an SVG icon — either use Tabler or leave it out.

**Sponsor card** has the system's only gradient: a 2px top rule that runs orange-bright → purple-bright. Modelled here as two components, `sponsor-rule-start` and `sponsor-rule-end`, each filling half the card width.

## Do's and Don'ts

**Do**

- Use warm cream surfaces (`bg-hard` `#f9f5d7`) in light and warm charcoal (`#1d2021`) in dark.
- Pair Podkova 600 headings with Schibsted Grotesk body, always in that order.
- Keep borders to 1px, shadows to the two-step sm/md pair, and radii to 2/4/6/8/12px.
- Let links invert to magenta on hover — it's the system's signature.
- Use stickers from `stickers.xeiaso.net/sticker/{char}/{mood}` whenever a character speaks.
- Write sentence-case headings and first-person, conversational body copy. Be specific and slightly funny.
- Reach for Tabler stroke icons only when text alone won't carry the meaning.
- Prefer muted accent hues on button surfaces with white text — they're the only pairing that meets WCAG AA.

**Don't**

- Don't introduce blue-purple gradients, neon, or cool grays — the palette is warm-neutral Gruvbox.
- Don't add emoji decoratively; stickers are the emotional channel.
- Don't use "left-border accent" cards — only `PullQuote` and admonitions carry a left rule, and both are intentional.
- Don't invent SVG icons; use Tabler or a placeholder and flag it.
- Don't override `a:hover` — the magenta invert is load-bearing.
- Don't pad with filler copy or hype; the voice is spare and personal.
- Don't use blur, transparency, glow, or any animation beyond the 1px hover lift and 200ms fades.
- Don't swap `-bright` accents onto button surfaces with white text — the contrast dips below 4.5:1.
