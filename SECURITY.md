# Security Policy

## Reporting a vulnerability

OpenFaithMap is pre-code / scaffolding-stage software (see [README.md](README.md#status)) — there
is no production deployment yet, but please still report suspected vulnerabilities privately
rather than opening a public issue, especially once real congregation/admin data is involved
(from M2 onward).

- Preferred: open a [GitHub Security Advisory](https://github.com/olehmushka/open-faith-map/security/advisories/new)
  on this repository.
- Alternative: email olegamysk@gmail.com with a description of the issue, steps to reproduce, and
  its potential impact.

Please include enough detail to reproduce the issue and, if known, which component is affected
(`openfaithmap-api`, `web/`, or the go-oikumenea integration layer).

## Scope

This repository is a facade over [go-oikumenea](https://github.com/olehmushka/go-oikumenea)
(D-CoreDependency — see [docs/architecture/decisions.md](docs/architecture/decisions.md)). A
vulnerability in go-oikumenea itself should be reported to that project directly; a vulnerability
in how OpenFaithMap integrates with it (token handling, authorization delegation, the
D-Exclusions taxon check) is in scope here.

## Response

As a young open-source project, there's no formal SLA yet. Reports are triaged as they arrive and
a fix or mitigation is prioritized based on severity.
