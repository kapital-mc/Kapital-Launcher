# Kapital Launcher

Branded desktop front-end for the Kapitel Kapital Minecraft packs

<!-- Project-owned. Seeded once from sandrogekeler/AllesSandro; rewrite freely.
The one agent file: Codex reads it directly, CLAUDE.md and GEMINI.md import it.
Keep it under ~200 lines, since it loads into every session. Put rules that
only matter for some files in .claude/rules/ with `paths:` frontmatter. -->

Rules shared with other projects are in `.claude/rules/`. An agent that does
not load that directory itself reads `working.md`, `writing.md` and
`simplicity.md` there before starting.

## Stack
- Go (module at the repo root)
- TypeScript, pnpm (package in `frontend/`)
- TODO: frameworks and versions that matter

## Project structure

TODO: what the layout does not show: where each kind of change goes and what
is generated.

## Architecture rules

TODO: the few rules a change must not break (layering, IPC boundaries, where
state lives). Short and checkable beats long and aspirational.

## Build & dev commands

```sh
go build ./...
go vet ./...
go test ./...
cd frontend && pnpm install
pnpm typecheck && pnpm lint && pnpm test
python3 .claude/suite-check.py   # every health check in .claude/suite.json
```

Run the last one before calling a task done. A check it reports as skipped did
not pass.

## Task tracking
Work is tracked in GitHub Issues. Every issue carries a `type:`, an `area:` and
exactly one priority (`p0`-`p3`) label. The issue forms ask for it and the
`issue-priority` workflow applies it; an issue opened through the API (by an
agent) sets the label in the same call, `p2` when unclear. Close issues from
the pull request with `Fixes #N`.

## Pull requests

Follow `CONTRIBUTING.md`: one pull request per piece of work, a `## Changelog`
line per change, an `area:` label.

## Files that come from the template

Some files are owned by sandrogekeler/AllesSandro and must not be edited here:
`.claude/rules/template-files.md` lists them.

## Docs

Describe what is on disk, in the present tense. Intent goes in
`agent_docs/ROADMAP.md` as an unchecked item. Unsure whether something landed?
Check before writing it down.

## Do not

- Commit secrets, `.env` files or credentials.
- Edit generated files or lockfiles by hand; regenerate them.
- Add a dependency without reading `agent_docs/DEPENDENCIES.md`.
