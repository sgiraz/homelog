#!/usr/bin/env node
/**
 * Validates the colour tokens declared in `src/assets/styles/main.css`.
 *
 * The palette is the one place in the UI where colour carries meaning, so it
 * has to hold up under conditions we cannot see while designing: every theme's
 * card surface (5 themes x light/dark) and the three dichromacies. Both checks
 * are mechanical, so they run here instead of living in a comment nobody can
 * verify.
 *
 * Checks, for the light set against every light surface and the dark set
 * against every dark surface:
 *   1. contrast >= MIN_CONTRAST (WCAG 1.4.11, non-text graphical objects)
 *   2. pairwise CIEDE2000 >= MIN_DELTA_E for normal vision, and
 *      >= MIN_DELTA_E_CVD after simulating protanopia/deuteranopia/tritanopia
 *
 * It also checks the semantic status pairs (positive / danger / warning /
 * info): the base is a fill and must clear MIN_CONTRAST against its own
 * surface, the `-soft` variant is used as text and must clear
 * MIN_TEXT_CONTRAST. That second bar is the one `--c-positive` was failing at
 * 3.4:1 in dark while being used for labels.
 *
 * Usage: node scripts/check-color-tokens.mjs
 */
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const CSS_PATH = join(dirname(fileURLToPath(import.meta.url)), '..', 'src', 'assets', 'styles', 'main.css')

const MIN_CONTRAST = 3
const MIN_TEXT_CONTRAST = 4.5
const STATUS_TOKENS = ['positive', 'danger', 'warning', 'info']
const MIN_DELTA_E = 12
const MIN_DELTA_E_CVD = 9

// ── CSS parsing ────────────────────────────────────────────────────────────
// Selector + body of every top-level rule, in source order.
function parseBlocks(css) {
  const blocks = []
  const re = /([^{}]+)\{([^{}]*)\}/g
  let m
  while ((m = re.exec(css))) {
    // Everything after the previous statement is the selector: at-rules such as
    // `@custom-variant dark (...)` sit on the same run of text and would
    // otherwise leak `.dark` into the next selector.
    const selector = m[1].split(/[;}]/).pop().trim()
    if (selector.startsWith('@')) continue
    blocks.push({ selector, body: m[2] })
  }
  return blocks
}

function declarations(body) {
  const out = {}
  for (const decl of body.split(';')) {
    const i = decl.indexOf(':')
    if (i === -1) continue
    out[decl.slice(0, i).trim()] = decl.slice(i + 1).trim().replace(/\/\*[\s\S]*?\*\//g, '').trim()
  }
  return out
}

// Comments are stripped first: a comment mentioning `.dark` right before a
// light-theme block would otherwise be read as part of its selector.
const css = readFileSync(CSS_PATH, 'utf8').replace(/\/\*[\s\S]*?\*\//g, '')
const blocks = parseBlocks(css)

/** All card surfaces a chart can be drawn on, split by mode. */
function surfaces(mode) {
  const found = []
  for (const { selector, body } of blocks) {
    const isDark = selector.includes('.dark')
    if (isDark !== (mode === 'dark')) continue
    const value = declarations(body)['--c-surface']
    if (value) found.push({ selector, rgb: value.split(/\s+/).map(Number) })
  }
  return found
}

/** Series colours of one mode, as { name, rgb } in slot order. */
function series(mode) {
  const merged = {}
  for (const { selector, body } of blocks) {
    const isDark = selector.includes('.dark')
    if (isDark !== (mode === 'dark')) continue
    for (const [prop, value] of Object.entries(declarations(body))) {
      if (prop.startsWith('--c-series-')) merged[prop] = value
    }
  }
  return Object.entries(merged).map(([name, hex]) => ({ name, rgb: hexToRgb(hex), hex }))
}

function hexToRgb(hex) {
  const h = hex.replace('#', '').trim()
  return [0, 2, 4].map((i) => parseInt(h.slice(i, i + 2), 16))
}

// ── Colour maths ───────────────────────────────────────────────────────────
const toLinear = (c) => (c / 255 <= 0.04045 ? c / 255 / 12.92 : ((c / 255 + 0.055) / 1.055) ** 2.4)
const fromLinear = (c) => 255 * (c <= 0.0031308 ? 12.92 * c : 1.055 * c ** (1 / 2.4) - 0.055)

function relativeLuminance([r, g, b]) {
  return 0.2126 * toLinear(r) + 0.7152 * toLinear(g) + 0.0722 * toLinear(b)
}

function contrast(a, b) {
  const [hi, lo] = [relativeLuminance(a), relativeLuminance(b)].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}

// Machado et al. (2009), severity 1.0, applied in linear RGB.
const CVD_MATRICES = {
  protanopia: [0.152286, 1.052583, -0.204868, 0.114503, 0.786281, 0.099216, -0.003882, -0.048116, 1.051998],
  deuteranopia: [0.367322, 0.860646, -0.227968, 0.280085, 0.672501, 0.047413, -0.011820, 0.042940, 0.968881],
  tritanopia: [1.255528, -0.076749, -0.178779, -0.078411, 0.930809, 0.147602, 0.004733, 0.691367, 0.303900]
}

function simulate(rgb, kind) {
  if (kind === 'normal') return rgb
  const m = CVD_MATRICES[kind]
  const [r, g, b] = rgb.map(toLinear)
  return [
    m[0] * r + m[1] * g + m[2] * b,
    m[3] * r + m[4] * g + m[5] * b,
    m[6] * r + m[7] * g + m[8] * b
  ].map((c) => Math.max(0, Math.min(255, fromLinear(Math.max(0, Math.min(1, c))))))
}

function rgbToLab(rgb) {
  const [r, g, b] = rgb.map(toLinear)
  // sRGB -> XYZ (D65), then XYZ -> Lab.
  const x = (0.4124 * r + 0.3576 * g + 0.1805 * b) / 0.95047
  const y = 0.2126 * r + 0.7152 * g + 0.0722 * b
  const z = (0.0193 * r + 0.1192 * g + 0.9505 * b) / 1.08883
  const f = (t) => (t > 216 / 24389 ? Math.cbrt(t) : (841 / 108) * t + 4 / 29)
  const [fx, fy, fz] = [f(x), f(y), f(z)]
  return [116 * fy - 16, 500 * (fx - fy), 200 * (fy - fz)]
}

/** CIEDE2000 colour difference. */
function deltaE2000(lab1, lab2) {
  const [L1, a1, b1] = lab1
  const [L2, a2, b2] = lab2
  const rad = Math.PI / 180
  const C1 = Math.hypot(a1, b1)
  const C2 = Math.hypot(a2, b2)
  const Cbar = (C1 + C2) / 2
  const G = 0.5 * (1 - Math.sqrt(Cbar ** 7 / (Cbar ** 7 + 25 ** 7)))
  const ap1 = (1 + G) * a1
  const ap2 = (1 + G) * a2
  const Cp1 = Math.hypot(ap1, b1)
  const Cp2 = Math.hypot(ap2, b2)
  const hp = (b, ap) => {
    if (b === 0 && ap === 0) return 0
    const h = Math.atan2(b, ap) / rad
    return h >= 0 ? h : h + 360
  }
  const hp1 = hp(b1, ap1)
  const hp2 = hp(b2, ap2)
  const dLp = L2 - L1
  const dCp = Cp2 - Cp1
  let dhp = 0
  if (Cp1 * Cp2 !== 0) {
    dhp = hp2 - hp1
    if (dhp > 180) dhp -= 360
    else if (dhp < -180) dhp += 360
  }
  const dHp = 2 * Math.sqrt(Cp1 * Cp2) * Math.sin((dhp / 2) * rad)
  const Lbar = (L1 + L2) / 2
  const Cpbar = (Cp1 + Cp2) / 2
  let hpbar = hp1 + hp2
  if (Cp1 * Cp2 !== 0) {
    if (Math.abs(hp1 - hp2) > 180) hpbar += hp1 + hp2 < 360 ? 360 : -360
    hpbar /= 2
  }
  const T =
    1 -
    0.17 * Math.cos((hpbar - 30) * rad) +
    0.24 * Math.cos(2 * hpbar * rad) +
    0.32 * Math.cos((3 * hpbar + 6) * rad) -
    0.2 * Math.cos((4 * hpbar - 63) * rad)
  const dTheta = 30 * Math.exp(-(((hpbar - 275) / 25) ** 2))
  const Rc = 2 * Math.sqrt(Cpbar ** 7 / (Cpbar ** 7 + 25 ** 7))
  const Sl = 1 + (0.015 * (Lbar - 50) ** 2) / Math.sqrt(20 + (Lbar - 50) ** 2)
  const Sc = 1 + 0.045 * Cpbar
  const Sh = 1 + 0.015 * Cpbar * T
  const Rt = -Math.sin(2 * dTheta * rad) * Rc
  return Math.sqrt(
    (dLp / Sl) ** 2 + (dCp / Sc) ** 2 + (dHp / Sh) ** 2 + Rt * (dCp / Sc) * (dHp / Sh)
  )
}

// ── Checks ─────────────────────────────────────────────────────────────────
const failures = []

for (const mode of ['light', 'dark']) {
  const colours = series(mode)
  if (!colours.length) failures.push(`${mode}: no --c-series-* tokens found`)

  for (const { selector, rgb } of surfaces(mode)) {
    for (const colour of colours) {
      const ratio = contrast(colour.rgb, rgb)
      if (ratio < MIN_CONTRAST) {
        failures.push(
          `${mode} ${colour.name} (${colour.hex}) on ${selector} surface: ${ratio.toFixed(2)}:1 < ${MIN_CONTRAST}:1`
        )
      }
    }
  }

  for (const vision of ['normal', 'protanopia', 'deuteranopia', 'tritanopia']) {
    const min = vision === 'normal' ? MIN_DELTA_E : MIN_DELTA_E_CVD
    const labs = colours.map((c) => rgbToLab(simulate(c.rgb, vision)))
    for (let i = 0; i < colours.length; i++) {
      for (let j = i + 1; j < colours.length; j++) {
        const dE = deltaE2000(labs[i], labs[j])
        if (dE < min) {
          failures.push(
            `${mode} ${vision}: ${colours[i].name} vs ${colours[j].name} too close (ΔE2000 ${dE.toFixed(1)} < ${min})`
          )
        }
      }
    }
  }
}

// ── Semantic status pairs, checked against the surface of their own block ───
// A block inherits what it does not redefine, so the globals are resolved from
// :root / .dark before comparing.
function inherited(mode) {
  const base = {}
  for (const { selector, body } of blocks) {
    if (selector !== (mode === 'dark' ? '.dark' : ':root')) continue
    Object.assign(base, declarations(body))
  }
  return base
}

for (const { selector, body } of blocks) {
  const decls = declarations(body)
  const surfaceValue = decls['--c-surface']
  if (!surfaceValue) continue
  const mode = selector.includes('.dark') ? 'dark' : 'light'
  const resolved = { ...inherited(mode), ...decls }
  const surface = surfaceValue.split(/\s+/).map(Number)

  for (const token of STATUS_TOKENS) {
    for (const [suffix, min] of [['', MIN_CONTRAST], ['-soft', MIN_TEXT_CONTRAST]]) {
      const value = resolved[`--c-${token}${suffix}`]
      if (!value) {
        failures.push(`${selector}: --c-${token}${suffix} is not defined`)
        continue
      }
      const ratio = contrast(value.split(/\s+/).map(Number), surface)
      if (ratio < min) {
        failures.push(
          `${selector} --c-${token}${suffix} on its surface: ${ratio.toFixed(2)}:1 < ${min}:1`
        )
      }
    }
  }
}

if (failures.length) {
  console.error('Chart palette check FAILED:\n' + failures.map((f) => `  - ${f}`).join('\n'))
  process.exit(1)
}
console.log('Colour tokens OK: series separation and status contrast hold for every theme surface.')
