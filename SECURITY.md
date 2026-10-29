# Security policy

Cache Vent moves and deletes files on people's machines, so security and data-safety reports get priority.

## Reporting a vulnerability

Email **lovekeshanand6@gmail.com** with "Cache Vent security" in the subject. Please don't open a public issue for anything that could let someone delete or read files they shouldn't, reach the local interface from another site or program, or tamper with downloads or install scripts.

Include the version (`cachevent version`), your OS, and steps to reproduce. You'll get a reply within a few days, and credit in the release notes if you want it.

## In scope

- Anything that removes or moves data outside the documented quarantine rules (see the safety model in the README)
- The local interface: its token, host check and HTTP API
- The install scripts and the integrity of files served from cachevent.kelados.in
- Following links or junctions while scanning

## Supported versions

Only the latest release gets fixes.
