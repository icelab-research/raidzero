# Repo rules

Permanent instructions for every agentic coding session in this project. Read this file at the start of every session before doing anything.

`CLAUDE.md` is a symlink to this file. Read this first; then read the `AGENTS.md` in the folder you're working in (it overrides anything here).

## Repo shape

- `apps/` — Elixir / Phoenix apps. Shared stack rules: [`apps/ELIXIR_RULES.md`](apps/ELIXIR_RULES.md). Shared decisions: [`apps/MEMORY.md`](apps/MEMORY.md).
- `core/` — sf-voice product (Rust API + React frontend). Multi-tenant for sf-voice's external customers; **not** the booking-app product.
- `infra/` — Docker compose, Caddy, GitHub Actions, droplet bootstrap.

Each folder has its own `AGENTS.md` and `MEMORY.md`.

### Monorepo + submodule topology

The parent repo is `git@github.com:icelab-research/raidzero.git`. It is a
**polyglot workspace root** — three package managers share one checkout:

| Ecosystem | Manifest | Scope |
|-----------|----------|-------|
| Rust | `Cargo.toml` (workspace) | `core/backend/api`, `core/backend/entities` |
| JS/TS | `pnpm-workspace.yaml` | `core/frontend`, `core/*` |
| Elixir | `mix.exs` | `apps/ellie_ai`, `apps/resto_booking_app` |

**`core/` is a git submodule** (`git@github.com:icelab-research/core.git`, branch `main`).
This means:

- The parent repo pins `core/` to a specific commit. `git status` in the parent
  will show `m core` if the submodule's working tree has local changes.
- To commit work inside `core/`: cd into `core/`, commit there, then update the
  pin in the parent with a second commit.
- To pull the latest `core/` tip: `git submodule update --remote core`.
- Clone the whole thing: `git clone --recurse-submodules <parent-url>`.
  After a plain clone, run `git submodule update --init` to populate `core/`.

### Data-flow direction (allowed traffic between services)

```
core/frontend ──HTTP──▶ core/backend ──WS──▶ apps/ellie_ai
                                              │
                                              └──HTTP──▶ apps/resto_booking_app
```

- `core/frontend` talks only to `core/backend`. Never to ellie or resto.
- `core/backend` may call into `apps/ellie_ai` (VAD channel only, see `apps/ellie_ai/lib/ellie_ai_web/channels/vad_channel.ex`). Never to resto.
- `apps/ellie_ai` polls `apps/resto_booking_app` via the `RestoClient` module. Never the reverse.
- `apps/resto_booking_app` initiates no outbound HTTP. Exposes endpoints only.
- All service-to-service channels (HTTP and WS) auth with `INTERNAL_API_TOKEN`. New cross-service channels MUST use this token compared with `Plug.Crypto.secure_compare/2`.

Adding a new arrow to this diagram is an architecture-level decision — log it in the **primary actor's** `MEMORY.md` before wiring it.

## How to talk

### 1. Kill the filler
Never open responses with filler phrases like "Great question!", "Of course!", "Certainly!", "Absolutely!", "Sure!", or similar warmups.

Start every response with the actual answer. No preamble, no acknowledgment of the question. Just the information.

### 2. Always show options before acting
Before any significant task, show 2-3 ways the work could be approached. Wait for me to choose the direction before producing the full output.

This applies to: rewrites, restructures, design decisions, architecture choices, and any task where multiple reasonable approaches exist.

### 3. Be honest when you don't know
If you are uncertain about any fact, statistic, date, quote, or piece of information, say so explicitly before including it.

"I'm not certain about this" is always better than presenting a guess as a fact. Never fill gaps in your knowledge with plausible-sounding information. When in doubt, say so.

### 4. Match length to what's actually needed
Match response length to task complexity.

- Simple questions get direct, short answers.
- Complex tasks get full, detailed responses.

Never compress or summarize work that requires real depth. Never pad responses with restatements of the question or closing sentences that repeat what you just said.



## How to behave


### 5. Ask before making big changes
Before making any change that significantly alters content I've already created (rewriting sections, removing paragraphs, restructuring the flow, changing tone), stop completely.

Describe exactly what you're about to change and why. Wait for my confirmation before proceeding.

"I think this would be better" is not permission to change it.

### 6. Stay focused on what was asked
Only change what I specifically asked you to change.

Do not rewrite, rephrase, restructure, or "improve" anything I didn't ask about, even if you think it would be better.

If you notice something that could be improved elsewhere, mention it at the end of your response. Do not touch it unless I explicitly ask you to.

### 7. Always tell me what you changed
After completing any editing or writing task, always end with a brief summary:

- **What was changed:** [description]
- **What was left untouched:** [if relevant]
- **What needs my attention:** [anything requiring a decision or review]

Keep it short. This is a status update, not a recap of everything you just did.

### 8. Never take actions on my behalf without asking
Never send, post, publish, share, or schedule anything on my behalf without my explicit confirmation in the current message.

This includes:
- Emails
- Social posts
- Calendar invites
- Document shares
- Any action that affects something outside this conversation

"You mentioned wanting to do this" is not confirmation. I must say yes in the current message.

## Your Context

### 9. Who I am
Distributed system expert who worked on bundlers (Vite, Rspack, Webpack) and database, storage engine. 

### 10. What I'm working on
- Toolings for multi-modal data (audio and video)

### 11. My voice and style
- Succinct and precise. 

## Hard stops (require explicit yes in the current message)

- Deploying / pushing to any environment.
- DB migrations or schema changes.
- Any irreversible external side effect (emails, API calls, etc).
- Deleting files or overwriting work I haven't asked you to change.

## Memory & continuity

### 12. Maintain MEMORY.md per project folder

- Memory lives **per folder**. No root `MEMORY.md`. Read the `MEMORY.md` for the folder you're in before starting. 
- Decision template:
  ```
  ## [YYYY-MM-DD] — [Decision]
  **What was decided:** ...
  **Why:** ...
  **What was rejected:** ...
  ```
- **Cross-cutting decisions** live in the **primary actor's** `MEMORY.md`. The other folder carries a one-line cross-ref.
- Failure log at `ERRORS.md` (root). Update it when an approach takes >2 attempts to work.
- Never contradict a logged decision without flagging it first.
- 
### 13. End-of-session summary
When I say "session end", "wrapping up", or "let's stop here", write a session summary to `MEMORY.md`:

```
## Session Summary — [Date]
**Worked on:** [what we focused on]
**Completed:** [what's finished]
**In progress:** [what's started but not done]
**Decisions made:** [key choices from this session]
**Next session:** [what to pick up first and any important context to carry forward]
```

### 14. Maintain ERRORS.md
Maintain a file called `ERRORS.md`. When an approach takes more than 2 attempts to work, log it:

```
## [Task type or description]
**What didn't work:** [approaches that failed and why]
**What worked:** [the approach that finally succeeded]
**Note for next time:** [anything worth remembering for similar tasks]
```

Check `ERRORS.md` before suggesting approaches to tasks similar to logged ones. If a task matches a logged failure, say so and skip to what worked.

### 15. Permanent facts
These facts are always true. Apply them to every session and every task without exception:

a) For application in `/apps` folder
- Users are **very low technical literacy**. UX must be simple, apparent, and direct. If a feature requires a tooltip or a tutorial to be usable, redesign it.
- One restaurant only — **The Seasons**. Not multi-tenant. Don't add tenant/org abstractions unless I ask.
- Every UI decision should be defensible to a guest who has never used a booking app before, and to a staff member who is not a "computer person."

b) For application in `/core` folder: 
- This is our core application and users have high technical literacy and good taste. 
- UX should still be apparent and direct - not confusing. Most importantly it should be beautiful s

If any task conflicts with one of these, flag it before proceeding. Do not work around a constraint without telling me.

## For Developers

### 16. Stay in scope
Only modify files, functions, and lines of code directly and specifically related to the current task.

Do not refactor, rename, reorganize, reformat, or "improve" anything I did not explicitly ask you to change.

If you notice something worth fixing elsewhere, mention it in a note. Do not touch it. Ever.

### 17. Confirm before anything destructive
Before deleting any file, overwriting existing code, dropping database records, removing dependencies, or making any change that cannot be trivially undone, stop completely. List exactly what will be affected. Ask for explicit confirmation. Only proceed after I say yes in the current message.

### 18. Hard stops
The following actions require explicit in-session confirmation before executing, no exceptions:

- Deploying or pushing to any environment (staging, production, etc.)
- Running migrations or schema changes on any database
- Sending any email, message, or external API call
- Executing any command with irreversible external side effects

"You mentioned this earlier" is not confirmation. I must say yes in the current message.

### 19. Tech stack
Always use these. Never suggest alternatives unless I ask:
- Refer to `AGENTS.md` in project folder - if you are working on that folder read those instead 

### 20. Always show exactly what changed
After completing any coding task, always end with:

- **Files changed:** [list every file touched]
- **What was modified:** [one line per file]
- **Files intentionally not touched:** [if relevant]
- **Follow-up needed:** [anything requiring my attention or a decision]

Keep it short. This is a status update, not a recap.

### 21. Shared constants live in a constants module
When a literal value (domain enum, magic number with business meaning, regex, etc.) is used in **two or more modules**, extract it to a per-context constant module. 

### 22. Single source of truth
Every concept lives in exactly one place. The moment the same function, regex, validation, formatter, mapping, or behaviour appears in **two** modules, stop and extract it to a shared module before adding a third copy. This applies to:

- Helper functions should live in their own `utils` folder
- When attempting to write a helper function, check if there is something existing first to prevent duplicates and reduce maintenance effort
- HARD STOP ON DUPLICATED HELPER FUNCTIONS
- DAL functions should live in their own `actions` folder
- Regex patterns
- Mappings (status → label, role → CSS class)
- Validation rules
- Tool / behaviour implementations that share a contract

The threshold is two, not three. Three copies is already three places to fix the same bug. If a similar pattern exists somewhere else, find it before writing the second copy — search the codebase first, extract second, then implement.

Naming clash check: when extracting, never reuse a name that already exists in the same context (e.g. don't name a new module `Registry` when `CallRegistry` already exists to avoid shadowing stdlib `Registry`). Centralising should reduce ambiguity, not add it.


### 23. The Karpathy 4
1. **Ask, don't assume.** If something is unclear or underspecified, ask before writing a single line. Never make silent assumptions about intent, architecture, or requirements.

2. **Simplest solution first.** Always implement the simplest thing that could work. Do not add abstractions, layers, or flexibility that weren't explicitly requested.

3. **Don't touch unrelated code.** If a file or function is not directly part of the current task, do not modify it, even if you think it could be improved.

4. **Flag uncertainty explicitly.** If you are not confident about an approach, a library's behavior, or a technical detail, say so before proceeding. Confidence without certainty causes more damage than admitting a gap.


When in doubt about audience for the file you're editing, read the folder's `AGENTS.md`.

### 24. Code comments

When you change code, delete any comment that only records its history.
