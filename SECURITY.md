# Security Policy

## Reporting a Vulnerability

Do not report vulnerabilities, credentials, host names, or connection logs in public issues. Use GitHub's private vulnerability reporting for this repository. If that channel is unavailable, contact the repository owner through a private GitHub channel.

Include the affected version, reproduction steps, expected impact, and any mitigations you have already applied. Do not include working credentials unless explicitly requested through a secure channel.

## Operational Guidance

- Treat `~/.sshmcp/state.db` as sensitive: it contains registered connection parameters and execution history.
- Keep the state database on a protected local disk. Do not sync it to shared folders, cloud drives, or source control.
- Use SSH keys or short-lived credentials where possible. Never commit passwords, private keys, `.env` files, local configuration, or database files.
- Credentials exposed in repository history before August 20, 2026 must be rotated and considered compromised.
