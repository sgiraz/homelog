# Contributing to HomeLog

Thanks for your interest in contributing! This guide covers the main ways to contribute.

## Table of Contents

- [Reporting bugs](#reporting-bugs)
- [Code contributions](#code-contributions)
- [Translations](#translations)

---

## Reporting bugs

Open an issue on GitHub. Include your HomeLog version, how to reproduce the bug, and what you expected vs. what happened.

---

## Code contributions

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Follow the existing code style (Go: `gofmt`; Vue: `<script setup>` Composition API; Tailwind CSS)
4. Commit using [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `chore:`, etc.
5. Open a Pull Request against `main`

See `CLAUDE.md` for architecture details, handler patterns, and known gotchas.

### Running locally

```bash
# Docker (recommended)
docker compose -f docker-compose.dev.yml up -d
# Frontend: http://localhost:5173 — Backend: http://localhost:8080

# Manual
cd backend && go run ./cmd/api/     # :8080
cd frontend && npm install && npm run dev  # :5173
```

---

## Translations

HomeLog uses [vue-i18n](https://vue-i18n.intlify.dev/) with a file-per-feature JSON layout:

```
frontend/src/i18n/locales/
├── en/          ← canonical source language (English)
│   ├── auth.json
│   ├── common.json
│   ├── expenses.json
│   └── ...
└── it/          ← Italian (and any other language)
    └── ...
```

English is the canonical source. All other languages mirror its key structure exactly.

### Fix or improve an existing translation

Edit the relevant file directly and open a PR:

```
frontend/src/i18n/locales/it/expenses.json   ← for Italian
frontend/src/i18n/locales/fr/expenses.json   ← for French, etc.
```

No code changes needed — new or edited JSON keys are picked up automatically at build time.

### Add a new language

1. Create the language directory and copy the EN files as a starting point:

   ```bash
   cp -r frontend/src/i18n/locales/en frontend/src/i18n/locales/fr
   ```

2. Translate all values in the new files (keys must stay identical to the EN source).

3. Register the locale in `frontend/src/i18n/index.js`:

   ```js
   export const SUPPORTED_LOCALES = ['it', 'en', 'fr']  // add your code
   ```

4. Add the language option to the settings UI in `frontend/src/components/settings/PreferencesTab.vue`.

5. Open a PR. Partial translations are fine — missing keys fall back to English automatically.

### Auto-translate with the script (optional)

If you've added new keys to the EN source and want to auto-generate a first pass for other languages, there's a helper script:

```bash
cd frontend
ANTHROPIC_API_KEY=sk-... npm run i18n:translate
# defaults: EN → IT

# Translate to a different language
ANTHROPIC_API_KEY=sk-... I18N_TARGETS=fr npm run i18n:translate

# Translate to multiple languages at once
ANTHROPIC_API_KEY=sk-... I18N_TARGETS=it,fr,de npm run i18n:translate

# Use a different source language (e.g. if you only know French)
ANTHROPIC_API_KEY=sk-... I18N_SOURCE=fr I18N_TARGETS=en npm run i18n:translate
```

The script fills in any keys present in the source but missing in the target. It never overwrites existing translations. Requires your own [Anthropic API key](https://console.anthropic.com/). Entirely optional — manual JSON editing works just as well.

### Translation rules

- **Never hardcode UI text in components** — always use `t('feature.key')`.
- Keep keys identical across all language files.
- Preserve interpolation placeholders verbatim: `{n}`, `{name}`, `{count}`, etc.
- Preserve vue-i18n escape sequences verbatim: `{'@'}`, `{'|'}`, `{'{'}`, `{'}'}`.
- Keep brand names unchanged: HomeLog, IBAN, POD, PDR.
- For new keys, add them to the EN source first, then to other languages.
