# Skill Governance Baseline

This note defines the minimum governance boundary for grouped skills in `gen-code` without claiming new runtime behavior.

- governed groups: `common`, `codex`, `cc`
- minimum inventory fields: `skill id`, `group`, `source`, `verification status`, `localization checked`
- allowed baseline states: `implemented`, `verified`, `partial`, `blocked`
- `skill discovered` does not mean `skill accepted`
- Phase B acceptance closes runtime, desktop, approval, and built-in tool coverage first
- grouped skill verification and Chinese localization audit remain separate follow-up work
- MCP metadata and skill governance are tracked separately to avoid mixing inventory status with runtime acceptance status
