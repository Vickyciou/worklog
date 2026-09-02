# Issue tracker: Local Markdown

Issues and specs for this repository live as Markdown files in `.scratch/`.

## Conventions

- One feature per directory: `.scratch/<feature-slug>/`.
- The feature specification is `.scratch/<feature-slug>/spec.md`.
- Implementation issues are separate files under `.scratch/<feature-slug>/issues/`, numbered from `01`.
- A `Status:` line near the top records workflow state. A specification ready for implementation uses `ready-for-agent`.
- A `Milestone:` line groups implementation issues by delivery stage without replacing their blocking edges.
- When a feature has a milestone overview, use it to identify the current assignment frontier.
- Comments and conversation history append under a `## Comments` heading.

## Publishing and fetching

When a skill publishes to the issue tracker, create or update the matching file under `.scratch/<feature-slug>/`. When a skill fetches a ticket, read the path or issue number supplied by the user. Assign only tickets whose declared blockers are complete.
