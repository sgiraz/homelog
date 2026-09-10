/**
 * Badge variants: state → token pair.
 *
 * Lives outside the component because `defineProps()` cannot reference a local
 * variable in `<script setup>`, and the validator has to see this map rather
 * than repeat the list of names.
 *
 * The fill is the base token at 10% and the label is the `-soft` variant, the
 * one guaranteed readable as text (see main.css, checked by
 * scripts/check-color-tokens.mjs).
 */
export const BADGE_VARIANTS = {
  neutral: 'bg-surface-2 text-ink-soft',
  positive: 'bg-positive/10 text-positive-soft',
  warning: 'bg-warning/10 text-warning-soft',
  danger: 'bg-danger/10 text-danger-soft',
  info: 'bg-info/10 text-info-soft'
}
