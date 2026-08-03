---
icon: octicons/shield-lock-16
---

# Security Release Process

This project maintains a security disclosure and response policy to responsibly handle critical
issues.

!!! important
    Do not file public issues on GitHub for security vulnerabilities.

## Supported Versions

For a list of support versions that this project will potentially create security fixes for, please
refer to the Releases page on this project's GitHub and/or project related documentation on release
cadence and support.

## Reporting a Vulnerability

All suspected security vulnerabilities should be reported privately to limit exposure.

Security is of the highest importance and all security vulnerabilities or suspected security
vulnerabilities should be reported to this project privately, to minimize attacks against current
users before they are fixed. Vulnerabilities will be investigated and patched on the next patch (or
minor) release as soon as possible. This information could be kept entirely internal to the project.

If you know of a publicly disclosed security vulnerability for this project, please contact the
maintainers of this project privately.

To report a vulnerability or a security-related issue, please open a GitHub Security Advisory. This
allows for anyone to report security vulnerabilities directly and privately to the maintainers using
GitHub.

Feedback will be sent within 7 business days, including a detailed plan to investigate the issue and
any potential workarounds to perform in the meantime.

Do not report non-security-impacting bugs through this channel. Use GitHub issues for all
non-security-impacting bugs.

## Proposed Report Content

Provide enough context for maintainers to replicate and confirm the issue.

Provide a descriptive title and in the description of the report include the following information:

- Basic identity information, such as your name and your affiliation or company.
- Detailed steps to reproduce the vulnerability.
- Description of the effects of the vulnerability on this project and the related hardware and
  software configurations, so that it can be reproduced.
- How the vulnerability affects this project's usage and an estimation of the attack surface, if
  there is one.
- List other projects or dependencies that were used in conjunction with this project to produce the
  vulnerability.

## When to Report a Vulnerability

Notify maintainers immediately if you discover or suspect any security issue, including potential
vulnerabilities in dependencies.

- When you think this project has a potential security vulnerability.
- When you suspect a potential vulnerability but you are unsure that it impacts this project.
- When you know of or suspect a potential vulnerability on another project that is used by this
  project.

## Patch, Release, and Disclosure

Upon acknowledgment, maintainers investigate, coordinate with the reporter, and prepare a patch or
workaround. If not deemed a true vulnerability, they provide the rationale.

The maintainers will respond to vulnerability reports as follows:

1. Will investigate the vulnerability and determine its effects and criticality.
2. If the issue is not deemed to be a vulnerability, a follow up will be provided with a detailed
   reason for rejection.
3. Will initiate a conversation with the reporter within 7 business days.
4. If a vulnerability is acknowledged and the timeline for a fix is determined, a plan to
   communicate with the appropriate community, including identifying mitigating steps that affected
   users can take to protect themselves until the fix is rolled out will be provided.
5. Will also create a Security Advisory using the
   [CVSS Calculator](https://www.first.org/cvss/calculator/3.0), if it is not created yet. Issues
   may also be reported to [Mitre](https://cve.mitre.org/) using this
   [scoring calculator](https://nvd.nist.gov/vuln-metrics/cvss/v3-calculator). The draft advisory
   will initially be set to private.
6. Will work on fixing the vulnerability and perform internal testing before preparing to roll out
   the fix.
7. Once the fix is confirmed, a patch for the vulnerability will be issued in the next patch or
   minor release, and backport a patch release into all earlier supported releases, if necessary.

## Public Disclosure Process

After validating and fixing an issue, an advisory will be published to recommend any necessary
mitigations.

## Confidentiality, Integrity, and Availability

We prioritize issues impacting data confidentiality, privilege escalation, or integrity.
Availability concerns are also handled promptly.

We consider vulnerabilities leading to the compromise of data confidentiality, elevation of
privilege, or integrity to be our highest priority concerns. Availability, in particular in areas
relating to DoS and resource exhaustion, is also a serious security concern. The maintainer team
takes all vulnerabilities, potential vulnerabilities, and suspected vulnerabilities seriously and
will investigate them in an urgent and expeditious manner.

Note that we do not currently consider the default settings for this project to be
secure-by-default. It is necessary for operators to explicitly configure settings, role based access
control, and other resource related features in this project to provide a hardened environment. We
will not act on any security disclosure that relates to a lack of safe defaults. Over time, we will
work towards improved safe-by-default configuration, taking into account backwards compatibility.
