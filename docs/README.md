# Engineering Docs

This folder is the living technical documentation for the Nucleus Remote Access Portal.

## Current Docs

- `current-state.md`
  - Runtime architecture
  - Preview deployment notes
  - Temporary product assumptions
  - Device identity resolution
  - Export session telemetry flow
  - Known gaps for the next engineer
- `agent-deployment.md`
  - Public preview via Cloudflare quick tunnel
  - External Nucleus Docker install steps
  - Serial device mounting notes
  - Verification and troubleshooting commands
- `public-deploy-runbook.md`
  - Public GitHub and GHCR publishing plan
  - Cloudflare named tunnel routing
  - Windows host startup steps
  - External `agente-rs` install commands

## Working Agreement

- Update `current-state.md` whenever a code change affects runtime behavior.
- Prefer documenting current truth over aspirational design.
- Mark temporary assumptions clearly so future multi-ORG work can replace them safely.
