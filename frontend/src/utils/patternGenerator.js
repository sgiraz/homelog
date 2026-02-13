/**
 * Pattern Generator
 * Automatically generates regex patterns from dropped tokens with context detection
 * Patterns are compatible with Go RE2 (no lookahead/lookbehind)
 */

import { TokenType, getTokenContext } from './tokenizer'

// Italian month pattern for regex
const ITALIAN_MONTHS_PATTERN = '(?:gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)'

/**
 * Field-specific value patterns
 * These match the actual value to capture
 */
const VALUE_PATTERNS = {
  amount_total: '(\\d{1,3}(?:[.\\s]\\d{3})*,\\d{2})',
  bill_number: '(\\d+)',
  period_start: null, // Dynamic based on detected format
  period_end: null,
  due_date: null,
  issue_date: null,
  service_code: null, // Dynamic: POD or PDR
  customer_code: '([A-Z0-9]+)',
  conversion_coefficient: '(\\d+,\\d+)',  // Italian decimal: 1,01843500
  provider_reading: '(\\d{1,3}(?:[.\\s]\\d{3})*(?:,\\d+)?)',  // Meter reading: 5.349 or 6.180,00
  reading_type: '([A-Za-zÀ-ú]+)',
  estimated_reading: '(\\d{1,3}(?:[.\\s]\\d{3})*(?:,\\d+)?)',  // Same as provider_reading
  estimated_consumption: '(\\d{1,3}(?:[.\\s]\\d{3})*,\\d+)',
  previous_estimated_consumption: '(\\d{1,3}(?:[.\\s]\\d{3})*,\\d+)'
}

/**
 * Date pattern based on detected format
 * @param {Object} token - The token containing the date
 * @returns {string} - Regex pattern for the date format
 */
function getDatePattern(token) {
  const text = token.text

  // Check if it's Italian text format (day month-name year)
  if (token.type === TokenType.MONTH) {
    // Will be combined with day and year from context
    return `(\\d{1,2})\\s+${ITALIAN_MONTHS_PATTERN}\\s+(\\d{4})`
  }

  // Numeric date formats
  if (/^\d{1,2}\/\d{1,2}\/\d{4}$/.test(text)) {
    return '(\\d{1,2}/\\d{1,2}/\\d{4})'
  }
  if (/^\d{1,2}-\d{1,2}-\d{4}$/.test(text)) {
    return '(\\d{1,2}-\\d{1,2}-\\d{4})'
  }
  if (/^\d{1,2}\.\d{1,2}\.\d{4}$/.test(text)) {
    return '(\\d{1,2}\\.\\d{1,2}\\.\\d{4})'
  }
  if (/^\d{1,2}\/\d{1,2}\/\d{2}$/.test(text)) {
    return '(\\d{1,2}/\\d{1,2}/\\d{2})'
  }

  // Generic date pattern
  return '(\\d{1,2}[/.-]\\d{1,2}[/.-]\\d{2,4})'
}

/**
 * Get service code pattern based on type
 * @param {Object} token - The token
 * @returns {string} - Regex pattern
 */
function getServiceCodePattern(token) {
  if (token.type === TokenType.POD) {
    return '(IT\\d{3}E\\d+)'
  }
  if (token.type === TokenType.PDR) {
    return '(\\d{10,})'
  }
  // Generic
  return '([A-Z0-9]{10,})'
}

/**
 * Escape special regex characters in a string
 * @param {string} str - The string to escape
 * @returns {string} - Escaped string safe for regex
 */
function escapeRegex(str) {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * Build prefix pattern from context tokens
 * @param {Array} beforeTokens - Tokens appearing before the value
 * @returns {string} - Regex pattern for the prefix
 */
function buildPrefixPattern(beforeTokens) {
  if (!beforeTokens || beforeTokens.length === 0) return ''

  // Take last 1-3 tokens for context, preferring text tokens
  const contextTokens = beforeTokens.slice(-3)

  // Build flexible pattern allowing whitespace between words
  const parts = contextTokens.map(t => {
    // For common variations, make them flexible
    let text = escapeRegex(t.text)

    // Make "N°" flexible to match "N°", "Nº", "N."
    if (/^[Nn][°º.]?$/.test(t.text)) {
      text = '[Nn][°º.]?'
    }

    return text
  })

  return parts.join('\\s*')
}

/**
 * Build suffix pattern from context tokens
 * @param {Array} afterTokens - Tokens appearing after the value
 * @returns {string} - Regex pattern for the suffix
 */
function buildSuffixPattern(afterTokens) {
  if (!afterTokens || afterTokens.length === 0) return ''

  // Usually just take the first token after
  const firstAfter = afterTokens[0]
  if (!firstAfter) return ''

  // Common suffix patterns
  const text = firstAfter.text

  // Currency symbols
  if (/^[€E$]$/.test(text)) {
    return '\\s*[€E$]?'
  }

  // Unit of measure
  if (/^(?:kWh|Smc|mc|m3|kW)$/i.test(text)) {
    return '\\s*' + escapeRegex(text)
  }

  return ''
}

/**
 * Get the human-readable prefix text (for display)
 * @param {Array} beforeTokens - Tokens appearing before the value
 * @returns {string} - Human-readable prefix
 */
function getPrefixText(beforeTokens) {
  if (!beforeTokens || beforeTokens.length === 0) return ''
  return beforeTokens.slice(-3).map(t => t.text).join(' ')
}

/**
 * Get context to the left of a word (same line)
 * @param {Object} token - The token
 * @param {Array} allTokens - All tokens
 * @returns {string} - Words to the left joined by space
 */
function getContextLeft(token, allTokens) {
  const sameLineTokens = allTokens.filter(t =>
    t.page === token.page &&
    Math.abs(t.y - token.y) < 10 && // Same line (within 10 units)
    t.x < token.x // To the left
  )
  // Sort by X position and get last 3 words
  sameLineTokens.sort((a, b) => a.x - b.x)
  const lastThree = sameLineTokens.slice(-3)
  return lastThree.map(t => t.text).join(' ')
}

/**
 * Get context above a word
 * @param {Object} token - The token
 * @param {Array} allTokens - All tokens
 * @returns {string} - Words above joined by space
 */
function getContextAbove(token, allTokens) {
  const aboveTokens = allTokens.filter(t =>
    t.page === token.page &&
    t.y < token.y && // Above
    t.y > token.y - 120 && // Within 120 units (~4cm)
    t.x < token.x + (token.width || 50) && // Horizontally overlapping
    t.x + (t.width || 50) > token.x
  )
  // Sort by Y position (closest first) and get first word
  aboveTokens.sort((a, b) => b.y - a.y)
  return aboveTokens.slice(0, 2).map(t => t.text).join(' ')
}

/**
 * Italian bill keywords for anchor confidence scoring
 */
const ANCHOR_KEYWORDS = [
  'importo', 'totale', 'pagare', 'scadenza', 'bolletta', 'consumo',
  'periodo', 'coefficiente', 'conversione', 'emissione', 'fattura',
  'numero', 'codice', 'cliente', 'lettura', 'data', 'spesa',
  'fornitura', 'contratto', 'utenza', 'fascia', 'tariffa'
]

/**
 * Fields that should use global search (unique identifiers in document)
 */
const GLOBAL_SEARCH_FIELDS = ['service_code', 'customer_code']

/**
 * Find the best anchor label for a dropped token by looking at surrounding text.
 * Searches left (same line) and above the token for descriptive label text.
 *
 * @param {Object} token - The dropped token
 * @param {Array} allTokens - All tokens from the document
 * @returns {Object} - { anchorText, anchorDirection, confidence }
 */
function detectAnchor(token, allTokens) {
  const leftAnchor = findLeftAnchor(token, allTokens)
  const aboveAnchor = findAboveAnchor(token, allTokens)

  // Pick the best anchor
  if (leftAnchor.confidence >= aboveAnchor.confidence && leftAnchor.anchorText) {
    return leftAnchor
  }
  if (aboveAnchor.anchorText) {
    return aboveAnchor
  }
  if (leftAnchor.anchorText) {
    return leftAnchor
  }

  return { anchorText: '', anchorDirection: 'right_or_below', confidence: 0 }
}

/**
 * Find anchor text to the left of the token (same line)
 * @param {Object} token - The target token
 * @param {Array} allTokens - All tokens
 * @returns {Object} - { anchorText, anchorDirection, confidence }
 */
function findLeftAnchor(token, allTokens) {
  // Find text tokens on the same line, to the left
  const sameLineLeft = allTokens.filter(t =>
    t.page === token.page &&
    Math.abs(t.y - token.y) < 10 &&
    t.x < token.x
  )

  // Sort by X descending (closest first)
  sameLineLeft.sort((a, b) => b.x - a.x)

  // Take up to 5 contiguous words (no gap > 40 units between them)
  const anchorWords = []
  for (let i = 0; i < Math.min(sameLineLeft.length, 5); i++) {
    const w = sameLineLeft[i]
    // Skip if it looks like a value (number, currency)
    if (/^\d+[,.]?\d*$/.test(w.text) || /^[€$]$/.test(w.text)) {
      break
    }
    // Check gap from previous word
    if (anchorWords.length > 0) {
      const prev = anchorWords[anchorWords.length - 1]
      if (prev.x - (w.x + (w.width || 0)) > 40) {
        break
      }
    }
    anchorWords.push(w)
  }

  if (anchorWords.length === 0) {
    return { anchorText: '', anchorDirection: 'right', confidence: 0 }
  }

  // Reverse to get left-to-right order
  anchorWords.reverse()
  const anchorText = anchorWords.map(w => w.text).join(' ')

  // Calculate confidence based on Italian bill keywords
  const lowerText = anchorText.toLowerCase()
  const keywordMatches = ANCHOR_KEYWORDS.filter(kw => lowerText.includes(kw))
  const confidence = Math.min(0.3 + keywordMatches.length * 0.25, 0.95)

  return { anchorText, anchorDirection: 'right', confidence }
}

/**
 * Find anchor text above the token
 * @param {Object} token - The target token
 * @param {Array} allTokens - All tokens
 * @returns {Object} - { anchorText, anchorDirection, confidence }
 */
function findAboveAnchor(token, allTokens) {
  // Find text tokens above, horizontally aligned, within 50 units vertical
  const aboveTokens = allTokens.filter(t =>
    t.page === token.page &&
    t.y < token.y &&
    t.y > token.y - 50 &&
    // Horizontal overlap check
    t.x < token.x + (token.width || 50) + 30 &&
    t.x + (t.width || 50) > token.x - 30
  )

  if (aboveTokens.length === 0) {
    return { anchorText: '', anchorDirection: 'below', confidence: 0 }
  }

  // Sort by Y descending (closest row first)
  aboveTokens.sort((a, b) => b.y - a.y)

  // Group by line (Y within 5 units) and take the closest line
  const closestY = aboveTokens[0].y
  const closestLine = aboveTokens.filter(t => Math.abs(t.y - closestY) < 5)

  // Sort line by X position
  closestLine.sort((a, b) => a.x - b.x)

  // Filter out pure numbers
  const textWords = closestLine.filter(t => !/^\d+[,.]?\d*$/.test(t.text) && !/^[€$]$/.test(t.text))

  if (textWords.length === 0) {
    return { anchorText: '', anchorDirection: 'below', confidence: 0 }
  }

  const anchorText = textWords.map(w => w.text).join(' ')

  const lowerText = anchorText.toLowerCase()
  const keywordMatches = ANCHOR_KEYWORDS.filter(kw => lowerText.includes(kw))
  const confidence = Math.min(0.2 + keywordMatches.length * 0.25, 0.9)

  return { anchorText, anchorDirection: 'below', confidence }
}

/**
 * Generate pattern for a specific field based on dropped token
 * @param {Object} token - The dropped token (includes position data)
 * @param {string} fieldKey - The target field key
 * @param {Array} allTokens - All tokens from the document
 * @param {string} rawText - Original raw text (for validation)
 * @returns {Object} - { pattern, prefix, suffix, valuePattern, position data, context, anchor data }
 */
export function generatePatternForField(token, fieldKey, allTokens, rawText) {
  const { before, after } = getTokenContext(token, allTokens, 3)

  // Get value pattern based on field type
  let valuePattern
  switch (fieldKey) {
    case 'amount_total':
    case 'consumption_total':
      valuePattern = VALUE_PATTERNS[fieldKey]
      break

    case 'period_start':
    case 'period_end':
    case 'due_date':
    case 'issue_date':
      valuePattern = getDatePattern(token)
      break

    case 'bill_number':
      valuePattern = VALUE_PATTERNS.bill_number
      break

    case 'service_code':
      valuePattern = getServiceCodePattern(token)
      break

    case 'customer_code':
      valuePattern = VALUE_PATTERNS.customer_code
      break

    case 'conversion_coefficient':
      valuePattern = VALUE_PATTERNS.conversion_coefficient
      break

    case 'provider_reading':
      valuePattern = VALUE_PATTERNS.provider_reading
      break

    case 'reading_type':
      valuePattern = VALUE_PATTERNS.reading_type
      break

    case 'estimated_date':
      valuePattern = getDatePattern(token)
      break

    case 'estimated_reading':
      valuePattern = VALUE_PATTERNS.estimated_reading
      break

    case 'estimated_consumption':
    case 'previous_estimated_consumption':
      valuePattern = VALUE_PATTERNS.estimated_consumption
      break

    default:
      // Generic: capture the exact value format
      valuePattern = `(${escapeRegex(token.text)})`
  }

  // Build context-aware pattern
  const prefixPattern = buildPrefixPattern(before)
  const suffixPattern = buildSuffixPattern(after)

  // Combine into full pattern
  let fullPattern = ''
  if (prefixPattern) {
    fullPattern += prefixPattern + '\\s*'
  }
  fullPattern += valuePattern
  if (suffixPattern) {
    fullPattern += suffixPattern
  }

  // Validate the pattern works on the raw text
  try {
    const regex = new RegExp(fullPattern, 'i')
    const match = rawText.match(regex)
    if (!match) {
      // Fallback to simpler pattern without full context
      fullPattern = valuePattern
    }
  } catch (e) {
    console.error('Invalid regex generated:', fullPattern, e)
    fullPattern = valuePattern
  }

  // Get spatial context for position-based extraction
  const contextLeft = getContextLeft(token, allTokens)
  const contextAbove = getContextAbove(token, allTokens)

  // Detect anchor and global search overrides
  const isGlobalSearch = GLOBAL_SEARCH_FIELDS.includes(fieldKey)
  let anchorText = ''
  let anchorDirection = 'right_or_below'

  if (!isGlobalSearch) {
    const anchor = detectAnchor(token, allTokens)
    anchorText = anchor.anchorText
    anchorDirection = anchor.anchorDirection
  }

  return {
    pattern: fullPattern,
    prefix: getPrefixText(before),
    suffix: after.length > 0 ? after[0].text : '',
    valuePattern: valuePattern,
    matchedValue: token.text,
    // Position data for position-based extraction
    page: token.page ?? token.pageIndex ?? 0,
    x: token.x ?? 0,
    y: token.y ?? 0,
    width: token.width ?? 0,
    height: token.height ?? 0,
    // Context for validation
    contextLeft,
    contextAbove,
    // Anchor-based extraction
    anchorText,
    anchorDirection,
    globalSearch: isGlobalSearch
  }
}

/**
 * Get neighbor words around a token, organized by direction
 * @param {Object} token - The target token
 * @param {Array} allTokens - All tokens from the document
 * @returns {Object} - { left: [...], above: [...], right: [...] }
 */
export function getNeighborWordsForToken(token, allTokens) {
  const result = { left: [], above: [], right: [] }

  if (!token || !allTokens || allTokens.length === 0) return result

  // Left: same line, to the left, sorted by distance (closest first but output left-to-right)
  const leftTokens = allTokens.filter(t =>
    t.page === token.page &&
    Math.abs(t.y - token.y) < 10 &&
    t.x < token.x &&
    t.id !== token.id
  )
  leftTokens.sort((a, b) => a.x - b.x) // left-to-right order
  result.left = leftTokens.slice(-8) // last 8 (closest to the token)

  // Right: same line, to the right
  const rightTokens = allTokens.filter(t =>
    t.page === token.page &&
    Math.abs(t.y - token.y) < 10 &&
    t.x > token.x &&
    t.id !== token.id
  )
  rightTokens.sort((a, b) => a.x - b.x)
  result.right = rightTokens.slice(0, 3)

  // Above: vertically above, horizontally overlapping
  // Use 120 units (~4cm) to reach labels separated by 2-3 lines from the value
  const aboveTokens = allTokens.filter(t =>
    t.page === token.page &&
    t.y < token.y &&
    t.y > token.y - 120 &&
    t.x < token.x + (token.width || 50) + 30 &&
    t.x + (t.width || 50) > token.x - 30 &&
    t.id !== token.id
  )
  // Group by line (Y within 5 units), collect up to 3 closest lines
  // so the user can see labels that are a few rows above the value
  if (aboveTokens.length > 0) {
    aboveTokens.sort((a, b) => b.y - a.y) // closest row first
    const aboveWords = []
    let currentLineY = null
    let lineCount = 0
    for (const t of aboveTokens) {
      if (currentLineY === null || Math.abs(t.y - currentLineY) > 5) {
        lineCount++
        if (lineCount > 3) break
        currentLineY = t.y
      }
      aboveWords.push(t)
    }
    // Sort all collected words by Y desc then X asc for consistent display
    aboveWords.sort((a, b) => a.y !== b.y ? b.y - a.y : a.x - b.x)
    result.above = aboveWords.slice(0, 8)
  }

  return result
}

/**
 * Regenerate a pattern using user-selected context words.
 * Called when the user adds/removes context words in the context editor.
 *
 * @param {Object} token - The mapped token
 * @param {string} fieldKey - The field key (e.g., 'amount_total')
 * @param {Array} allTokens - All tokens from the document
 * @param {string} rawText - Raw PDF text for validation
 * @param {Object} selectedContext - { left: [wordObj,...], above: [wordObj,...], right: [wordObj,...] }
 * @returns {Object} - Pattern info object (same shape as generatePatternForField result)
 */
export function regeneratePatternWithContext(token, fieldKey, allTokens, rawText, selectedContext) {
  // 1. Get the value pattern based on field type (same logic as generatePatternForField)
  let valuePattern
  switch (fieldKey) {
    case 'amount_total':
    case 'consumption_total':
      valuePattern = VALUE_PATTERNS[fieldKey]
      break
    case 'period_start':
    case 'period_end':
    case 'due_date':
    case 'issue_date':
    case 'estimated_date':
      valuePattern = getDatePattern(token)
      break
    case 'bill_number':
      valuePattern = VALUE_PATTERNS.bill_number
      break
    case 'service_code':
      valuePattern = getServiceCodePattern(token)
      break
    case 'customer_code':
      valuePattern = VALUE_PATTERNS.customer_code
      break
    case 'conversion_coefficient':
      valuePattern = VALUE_PATTERNS.conversion_coefficient
      break
    case 'provider_reading':
      valuePattern = VALUE_PATTERNS.provider_reading
      break
    case 'reading_type':
      valuePattern = VALUE_PATTERNS.reading_type
      break
    case 'estimated_reading':
      valuePattern = VALUE_PATTERNS.estimated_reading
      break
    case 'estimated_consumption':
    case 'previous_estimated_consumption':
      valuePattern = VALUE_PATTERNS.estimated_consumption
      break
    default:
      valuePattern = `(${escapeRegex(token.text)})`
  }

  // 2. Build prefix/suffix from selected context words
  const prefixTokens = selectedContext.left || []
  const suffixTokens = selectedContext.right || []

  const prefixParts = prefixTokens.map(t => escapeRegex(t.text))
  const suffixParts = suffixTokens.map(t => escapeRegex(t.text))

  // 3. Assemble the full pattern
  let fullPattern = ''
  if (prefixParts.length > 0) {
    fullPattern += prefixParts.join('\\s+') + '\\s+'
  }
  fullPattern += valuePattern
  if (suffixParts.length > 0) {
    fullPattern += '\\s+' + suffixParts.join('\\s+')
  }

  // 4. Validate on raw text
  try {
    const regex = new RegExp(fullPattern, 'i')
    if (!rawText.match(regex)) {
      // Try more permissive spacing
      let permissive = ''
      if (prefixParts.length > 0) {
        permissive += prefixParts.join('[\\s:]*') + '[\\s:]*'
      }
      permissive += valuePattern
      if (suffixParts.length > 0) {
        permissive += '[\\s:]*' + suffixParts.join('[\\s:]*')
      }

      const permissiveRegex = new RegExp(permissive, 'i')
      if (rawText.match(permissiveRegex)) {
        fullPattern = permissive
      } else if (prefixParts.length > 0) {
        // Fallback: just use last prefix word + value
        fullPattern = prefixParts[prefixParts.length - 1] + '\\s+' + valuePattern
        const fallbackRegex = new RegExp(fullPattern, 'i')
        if (!rawText.match(fallbackRegex)) {
          fullPattern = valuePattern // bare value pattern as last resort
        }
      }
    }
  } catch (e) {
    fullPattern = valuePattern
  }

  // 5. Build anchor text and direction from selected context
  const anchorText = prefixTokens.map(t => t.text).join(' ')
  const aboveTokens = selectedContext.above || []
  let anchorDirection = 'right_or_below'
  if (prefixTokens.length > 0) {
    anchorDirection = 'right'
  } else if (aboveTokens.length > 0) {
    anchorDirection = 'below'
  }

  const isGlobalSearch = GLOBAL_SEARCH_FIELDS.includes(fieldKey)

  return {
    pattern: fullPattern,
    prefix: prefixTokens.map(t => t.text).join(' '),
    suffix: suffixTokens.map(t => t.text).join(' '),
    valuePattern,
    matchedValue: token.text,
    page: token.page ?? token.pageIndex ?? 0,
    x: token.x ?? 0,
    y: token.y ?? 0,
    width: token.width ?? 0,
    height: token.height ?? 0,
    contextLeft: getContextLeft(token, allTokens),
    contextAbove: getContextAbove(token, allTokens),
    anchorText: isGlobalSearch ? '' : anchorText,
    anchorDirection: isGlobalSearch ? 'right_or_below' : anchorDirection,
    globalSearch: isGlobalSearch
  }
}

/**
 * Test a pattern against raw text
 * @param {string} pattern - Regex pattern
 * @param {string} rawText - Text to search
 * @returns {Object} - { success: boolean, value: string|null, error: string|null }
 */
export function testPattern(pattern, rawText) {
  if (!pattern || !rawText) {
    return { success: false, value: null, error: 'Pattern or text is empty' }
  }

  try {
    const regex = new RegExp(pattern, 'i')
    const match = rawText.match(regex)

    if (match && match.length > 1) {
      return { success: true, value: match[1], error: null }
    } else if (match) {
      return { success: true, value: match[0], error: null }
    }

    return { success: false, value: null, error: 'Nessuna corrispondenza trovata' }
  } catch (e) {
    return { success: false, value: null, error: `Regex non valida: ${e.message}` }
  }
}


