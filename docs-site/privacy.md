---
title: Privacy Policy
description: What HomeLog's public sites store, who can see it, and how to exercise your rights.
---

# Privacy Policy

This policy covers the three public HomeLog sites:

| Site | What it is |
| --- | --- |
| `homelog.dev` | Static landing page |
| `docs.homelog.dev` | This documentation site |
| `demo.homelog.dev` | Public demo instance, on a single shared account |

It does **not** cover a HomeLog instance you install on your own server. There,
you are the data controller — see [Self-hosted installations](#self-hosted-installations).

**Last updated:** 30 August 2026

## Who is responsible

**Controller:** Simone Girardi
**Contact:** [privacy@homelog.dev](mailto:privacy@homelog.dev)

Write to that address for anything concerning your personal data — it reaches a
private mailbox. For everything else (bugs, feature requests, questions about
the software) please use [GitHub issues](https://github.com/sgiraz/homelog/issues)
instead, so the discussion stays public and useful to others.

## The short version

- No analytics, no advertising, no tracking, no profiling.
- No cookies requiring consent — which is why you see no cookie banner.
- The sites store a few technical values in your browser (language, theme,
  and on the demo a login token). These never leave your device.
- Web fonts are served from our own domains, so no font provider sees you.

## What is stored in your browser

None of this is a cookie: it is `localStorage` / `sessionStorage`, which stays
on your device and is never transmitted to us. All of it is either strictly
necessary or a preference you set yourself, so under Article 5(3) of the
ePrivacy Directive (in Italy, Article 122 of the Privacy Code) it is exempt
from consent.

| Site | Key | Purpose | Retained |
| --- | --- | --- | --- |
| `homelog.dev` | `homelog-lang` | The language you picked | Until you clear site data |
| `homelog.dev` | `homelog-theme` | Light/dark preference | Until you clear site data |
| `docs.homelog.dev` | `vitepress-theme-appearance` | Light/dark preference | Until you clear site data |
| `docs.homelog.dev` | `hl-lang-redirect` | Marks that the one-time redirect to the Italian docs already happened | Until the tab is closed |
| `demo.homelog.dev` | `token`, `refreshToken`, `user` | Keeps you signed in to the demo account | Until logout or clearing site data |
| `demo.homelog.dev` | `themeMode`, `colorTheme`, `locale` | Appearance and language preferences | Until you clear site data |

You can delete all of it at any time through your browser's "clear site data"
function. The sites keep working; they just forget your preferences.

## Server logs

Each site is served by a hosting provider that records ordinary web-server
logs — IP address, timestamp, requested URL, user agent — for security and
troubleshooting. This is processed on the basis of legitimate interest
(Article 6(1)(f) GDPR) in keeping the services available and secure. Retention
and further details are governed by each provider:

- `homelog.dev` and `demo.homelog.dev` — [Render](https://render.com/privacy)
- `docs.homelog.dev` — [GitHub Pages](https://docs.github.com/site-policy/privacy-policies/github-general-privacy-statement)

## Third parties your browser contacts

Loading a page means your browser connects to whoever serves its content, and
those parties necessarily see your IP address. On these sites that is:

- **Rendering `homelog.dev`:** the Tailwind CSS CDN (`cdn.tailwindcss.com`),
  which serves the stylesheet engine the page is built with.
- **Rendering `docs.homelog.dev` and `demo.homelog.dev`:** nothing beyond the
  hosting provider.

Web fonts used to be loaded from Google Fonts on the first two sites. They are
now served from our own domains, so Google is no longer contacted.

Links to GitHub, Ko-fi and GitHub Sponsors are ordinary outbound links: nothing
is sent to them unless you click.

## The public demo

`demo.homelog.dev` runs on **one shared account** that every visitor signs into.
This has consequences worth stating plainly:

- Anything you enter is **visible to every other visitor** while it is there.
- The whole dataset is **wiped and re-seeded every hour**, and anyone can
  trigger a reset at any time. Nothing you enter is preserved.
- The demo exists to show the software. **Do not enter real personal data** —
  yours or anyone else's.

The demo account's credentials are published on the sign-in screen, so access to
it is not a security boundary.

## Self-hosted installations

If you install HomeLog on your own server, your data stays there. We receive
nothing: there is no telemetry, no phone-home, no usage reporting. The only
outbound request the application makes on its own is a call to the GitHub
releases API to check whether a newer version exists, made by *your* server, not
by your browser.

In that setup, whoever runs the instance is the data controller for the
household data it holds, and this policy does not apply to it.

## Your rights

Under Articles 15–22 GDPR you may request access to your personal data, its
rectification or erasure, restriction of processing, portability, and you may
object to processing based on legitimate interest. Given how little these sites
process, in practice this concerns the server logs described above.

Write to the contact address at the top of this page. You also have the right to
lodge a complaint with a supervisory authority — in Italy, the
[Garante per la protezione dei dati personali](https://www.garanteprivacy.it).

## Changes

The date at the top of this page reflects the last substantive change. The full
revision history of this document is public in the
[HomeLog repository](https://github.com/sgiraz/homelog/commits/main/docs-site/privacy.md).
