# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in fpm, please report it responsibly.

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please email: security@fpm-cli.dev (or open a private security advisory
on GitHub).

We will acknowledge receipt within 48 hours and provide a timeline for a fix.

## Scope

Security issues we care about:

- Arbitrary code execution during package installation
- Path traversal in wheel extraction
- Command injection via package names or versions
- Cache poisoning attacks
- Authentication credential leakage
- Supply chain attacks via dependency confusion

## Supported Versions

Only the latest release receives security updates.
