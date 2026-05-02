#!/usr/bin/env node
/**
 * Auto-translate missing i18n keys using the Claude API.
 *
 * Usage:
 *   ANTHROPIC_API_KEY=sk-... npm run i18n:translate
 *
 * Defaults: EN → IT. Override via env vars:
 *   I18N_SOURCE=en I18N_TARGETS=fr,de npm run i18n:translate
 *
 * Behavior:
 *   - Reads frontend/src/i18n/locales/{SOURCE}/*.json
 *   - For each target language, finds keys present in source but missing in
 *     target, sends them to Claude in one API call per file, merges the
 *     result back. Existing translations are NEVER overwritten.
 *   - Safe to re-run after adding new source keys.
 */

import { readFileSync, writeFileSync, readdirSync, existsSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import Anthropic from '@anthropic-ai/sdk'

const __dirname = dirname(fileURLToPath(import.meta.url))
const LOCALES_DIR = join(__dirname, '..', 'src', 'i18n', 'locales')
const SOURCE_LANG = process.env.I18N_SOURCE || 'en'
const TARGET_LANGS = (process.env.I18N_TARGETS || 'it').split(',').map(l => l.trim()).filter(Boolean)

if (!process.env.ANTHROPIC_API_KEY) {
  console.error('ERROR: ANTHROPIC_API_KEY env var is required')
  process.exit(1)
}

const client = new Anthropic()

const SYSTEM_PROMPT = `You translate UI strings for HomeLog, a self-hosted household expense and utilities management web app for families. The app handles household budgets, bill tracking (electricity, gas, water, internet), meter readings, expense splitting between family members, and shared projects.

You receive source strings as a flat JSON object (keys are dot-paths like "settings.account.title"). You output a JSON object with the EXACT SAME KEYS, values translated to the requested target language.

CRITICAL RULES:
1. Output ONLY a valid JSON object. No markdown fences. No commentary. No leading/trailing text.
2. Preserve interpolation placeholders verbatim: {n}, {0}, {name}, {count}, etc. Don't translate variable names.
3. Preserve vue-i18n escape syntax verbatim: {'@'}, {'|'}, {'$'}, {'{'}, {'}'} — keep exactly as-is, including the surrounding curly braces and single quotes.
4. Keep brand and proper names unchanged: HomeLog, IBAN, POD, PDR, IT, EUR, USD, GBP.
5. Match tone and punctuation: trailing "..." stays, ":" stays, "!" stays, "?" stays, leading "←" stays.
6. Use natural target-language phrasing for UI: short, clear, action-oriented. Don't translate word-by-word.
7. Match button-label conventions of the target language (e.g. for Italian UI, prefer "Accedi" over "Login", "Salva" over "Save").
8. Keep technical terms standard: "bill" → "bolletta", "reading" → "lettura", "consumption" → "consumo", "utility/service" → "utenza/servizio", "expense" → "spesa", "balance" → "saldo/bilancio", "due date" → "scadenza", "installment-based" → "rateizzata", "auto-debit" → "domiciliata".
9. If the value is already in the target language (e.g. proper noun, English-only term), return it unchanged.

EXAMPLES:

Input (translate to Italian):
{"actions.save": "Save", "states.loading": "Loading...", "expenses.empty": "No expenses recorded", "bills.dueIn": "Due in {n} days"}

Output:
{"actions.save": "Salva", "states.loading": "Caricamento...", "expenses.empty": "Nessuna spesa registrata", "bills.dueIn": "Scade tra {n} giorni"}`

const LANG_NAME = { en: 'English', it: 'Italian', fr: 'French', de: 'German', es: 'Spanish', pt: 'Portuguese' }

function flatten(obj, prefix = '') {
  const out = {}
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      Object.assign(out, flatten(v, key))
    } else {
      out[key] = v
    }
  }
  return out
}

function unflatten(flat) {
  const out = {}
  for (const [path, val] of Object.entries(flat)) {
    const parts = path.split('.')
    let node = out
    for (let i = 0; i < parts.length - 1; i++) {
      if (!node[parts[i]] || typeof node[parts[i]] !== 'object') {
        node[parts[i]] = {}
      }
      node = node[parts[i]]
    }
    node[parts[parts.length - 1]] = val
  }
  return out
}

async function translateBatch(strings, targetLang) {
  const langName = LANG_NAME[targetLang] || targetLang
  const userMsg = `Translate to ${langName}:\n\n${JSON.stringify(strings, null, 2)}`

  const response = await client.messages.create({
    model: 'claude-sonnet-4-6',
    max_tokens: 8192,
    system: [
      {
        type: 'text',
        text: SYSTEM_PROMPT,
        cache_control: { type: 'ephemeral' },
      },
    ],
    messages: [{ role: 'user', content: userMsg }],
  })

  const text = response.content.map(b => b.text || '').join('').trim()
  const cleaned = text
    .replace(/^```(?:json)?\s*\n?/i, '')
    .replace(/\n?```\s*$/i, '')
    .trim()

  try {
    return JSON.parse(cleaned)
  } catch (err) {
    console.error('Failed to parse model output as JSON:')
    console.error(cleaned)
    throw err
  }
}

async function processFile(filename, targetLang) {
  const sourcePath = join(LOCALES_DIR, SOURCE_LANG, filename)
  const targetPath = join(LOCALES_DIR, targetLang, filename)

  const source = JSON.parse(readFileSync(sourcePath, 'utf8'))
  const target = existsSync(targetPath)
    ? JSON.parse(readFileSync(targetPath, 'utf8'))
    : {}

  const sourceFlat = flatten(source)
  const targetFlat = flatten(target)

  const missing = {}
  for (const k of Object.keys(sourceFlat)) {
    if (!(k in targetFlat)) missing[k] = sourceFlat[k]
  }

  if (Object.keys(missing).length === 0) {
    console.log(`  [${targetLang}/${filename}] up to date (${Object.keys(sourceFlat).length} keys)`)
    return
  }

  console.log(`  [${targetLang}/${filename}] translating ${Object.keys(missing).length} keys...`)
  const translated = await translateBatch(missing, targetLang)

  const returned = Object.keys(translated)
  const requested = Object.keys(missing)
  const missingFromResponse = requested.filter(k => !returned.includes(k))
  if (missingFromResponse.length > 0) {
    console.warn(`  WARN: model omitted ${missingFromResponse.length} keys: ${missingFromResponse.join(', ')}`)
  }

  const merged = { ...translated, ...targetFlat }
  const out = unflatten(merged)

  writeFileSync(targetPath, JSON.stringify(out, null, 2) + '\n', 'utf8')
  console.log(`  [${targetLang}/${filename}] wrote ${Object.keys(translated).length} new keys`)
}

async function main() {
  const sourceDir = join(LOCALES_DIR, SOURCE_LANG)
  const files = readdirSync(sourceDir).filter(f => f.endsWith('.json'))
  if (files.length === 0) {
    console.log(`No source JSON files in ${sourceDir}`)
    return
  }

  console.log(`Source: ${SOURCE_LANG} (${files.length} files)`)
  for (const lang of TARGET_LANGS) {
    console.log(`\n→ ${lang}`)
    for (const f of files) {
      await processFile(f, lang)
    }
  }
  console.log('\nDone. Review with `git diff` before committing.')
}

main().catch(err => {
  console.error(err)
  process.exit(1)
})
