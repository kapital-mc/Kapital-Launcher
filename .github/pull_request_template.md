<!--
Title and Changelog lines: imperative, sentence case, no trailing period, no
em dashes. They become the release notes. One pull request can carry several
changes; split only when a part is risky on its own, has to be revertable on
its own, or ships at a different time. Label it with an area:, and with a
type: when it has no Changelog section. CONTRIBUTING.md has the type ladder.
-->

## Why

<!-- The situation this changes. Link the issue. -->

## What

<!-- What changed, in a few lines. The diff is the record; this is the summary. -->

## Changelog

<!--
One line per change: - type: text. The type is feature, fix, chore, docs or
skip; chore and docs are counted in the notes, not listed. For example:

- feature: Add a storage section to chapter settings
- fix: Open a storage group from anywhere on its tile

Leave it empty for a single change and the notes use the title and the type:
label instead (the pr-labelled status warns that it did).
-->

## How I checked it

<!-- `.claude/suite-check.py` green, plus anything a gate cannot see: a real launch, a screenshot. -->
