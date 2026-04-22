/**
 * PDF Text Tokenizer
 * Splits extracted PDF text into draggable tokens with type classification
 * Supports word positions and neighbor detection for tooltip display
 */

// Italian month names for date detection
const ITALIAN_MONTHS = [
  'gennaio', 'febbraio', 'marzo', 'aprile', 'maggio', 'giugno',
  'luglio', 'agosto', 'settembre', 'ottobre', 'novembre', 'dicembre'
]

// Threshold for detecting same line (in Y coordinate units)
const SAME_LINE_THRESHOLD = 5

/**
 * Token types for classification
 */
export const TokenType = {
  CURRENCY: 'currency',     // Euro amounts: 123,45 or 1.234,56
  NUMBER: 'number',         // Plain numbers
  DATE: 'date',             // Date formats: DD/MM/YYYY, etc.
  MONTH: 'month',           // Italian month names
  SYMBOL: 'symbol',         // Currency/unit symbols: €, $
  TEXT: 'text',             // Regular text
  POD: 'pod',               // Electricity POD code
  PDR: 'pdr',               // Gas PDR code
  PUNCTUATION: 'punctuation', // Symbols we might skip
  NOISE: 'noise'            // QR codes, URLs, encoded data to filter out
}

/**
 * Check if text looks like QR code noise or other non-useful data
 * @param {string} text - The text to check
 * @returns {boolean} - True if it's noise that should be filtered
 */
function isNoiseContent(text) {
  const trimmed = text.trim()

  // Skip very short tokens (single chars except digits and currency symbols)
  if (trimmed.length === 1 && !/[\d€$]/.test(trimmed)) {
    return true
  }

  // URLs and web content
  if (/^https?:\/\//i.test(trimmed) || /^www\./i.test(trimmed)) {
    return true
  }

  // Email addresses
  if (/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmed)) {
    return true
  }

  // Base64-like encoded strings (long strings of alphanumeric with no spaces)
  if (/^[A-Za-z0-9+/=]{20,}$/.test(trimmed) && !/\s/.test(trimmed)) {
    return true
  }

  // QR code signatures or encoded data patterns
  if (/^[A-Z0-9]{15,}$/.test(trimmed) && !/^IT\d{3}E/.test(trimmed)) {
    // Long uppercase alphanumeric but not POD codes
    return true
  }

  // Mixed random-looking alphanumeric (QR code data)
  if (/^[A-Za-z0-9]{10,}$/.test(trimmed)) {
    // Check if it looks random (mix of upper, lower, digits without word patterns)
    const hasUpper = /[A-Z]/.test(trimmed)
    const hasLower = /[a-z]/.test(trimmed)
    const hasDigit = /\d/.test(trimmed)
    // If all three mixed and no recognizable pattern, likely QR data
    if (hasUpper && hasLower && hasDigit && !/^[A-Za-z]+\d+$/.test(trimmed)) {
      return true
    }
  }

  // Hex strings (often from QR codes)
  if (/^[0-9A-Fa-f]{16,}$/.test(trimmed)) {
    return true
  }

  // Special characters sequences (often from QR/barcode)
  if (/^[[\]{}|\\<>^`~]+/.test(trimmed)) {
    return true
  }

  return false
}

/**
 * Normalize text to fix common PDF encoding issues
 * @param {string} text - The raw token text
 * @returns {string} - Normalized text
 */
function normalizeText(text) {
  return text
    .replace(/\uFFFD{1,2}¬/g, '€')  // Replacement chars + ¬ from broken 3-byte € sequence
    .replace(/â¬/g, '€')        // UTF-8 euro sign interpreted as Latin-1
    .replace(/â\u0082¬/g, '€')  // Another common UTF-8/Latin-1 misinterpretation
    .replace(/â\u201A¬/g, '€')  // 0x82 mapped to ‚ (U+201A) by Win-1252 + ¬
    .replace(/\u0080/g, '€')    // Windows-1252 euro sign
    .replace(/Ã¨/g, 'è')        // UTF-8 è interpreted as Latin-1
    .replace(/Ã©/g, 'é')        // UTF-8 é interpreted as Latin-1
    .replace(/Ã /g, 'à')        // UTF-8 à interpreted as Latin-1
    .replace(/Ã¹/g, 'ù')        // UTF-8 ù interpreted as Latin-1
    .replace(/Ã²/g, 'ò')        // UTF-8 ò interpreted as Latin-1
}

/**
 * Classify a token's type based on its content
 * @param {string} text - The token text
 * @returns {string} - TokenType value
 */
export function classifyToken(text) {
  const trimmed = normalizeText(text.trim())

  // Skip empty or whitespace-only
  if (!trimmed) return null

  // Check for noise content first (QR codes, URLs, encoded data)
  if (isNoiseContent(trimmed)) {
    return TokenType.NOISE
  }

  // POD code (electricity): IT followed by digits and letters
  if (/^IT\d{3}E\d+$/i.test(trimmed)) {
    return TokenType.POD
  }

  // PDR code (gas): long numeric code, typically 14 digits
  // But not too long (over 20 digits is likely noise)
  if (/^\d{10,20}$/.test(trimmed)) {
    return TokenType.PDR
  }

  // Currency/unit symbols (€, $)
  if (/^[€$]$/.test(trimmed)) {
    return TokenType.SYMBOL
  }

  // Currency: Italian format with decimal (e.g., 123,45 or 1.234,56)
  // Must have comma and 2 decimal digits
  if (/^\d{1,3}(?:[.\s]\d{3})*,\d{2}$/.test(trimmed)) {
    return TokenType.CURRENCY
  }

  // Date formats: DD/MM/YYYY, DD-MM-YYYY, DD.MM.YYYY
  if (/^\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4}$/.test(trimmed)) {
    return TokenType.DATE
  }

  // Italian month names
  if (ITALIAN_MONTHS.includes(trimmed.toLowerCase())) {
    return TokenType.MONTH
  }

  // Plain numbers (integers or decimals without Italian currency format)
  // Limit to reasonable length for bill amounts/readings
  if (/^\d{1,10}[,.]?\d{0,4}$/.test(trimmed)) {
    return TokenType.NUMBER
  }

  // Pure punctuation (currency symbols excluded — they are classified above)
  if (/^[.,;:!?\-\u2013\u2014()[\]{}%]+$/.test(trimmed)) {
    return TokenType.PUNCTUATION
  }

  // Filter out very long text that's likely noise
  if (trimmed.length > 30) {
    return TokenType.NOISE
  }

  // Default to text
  return TokenType.TEXT
}

/**
 * Find context around a token (words before and after on the same line)
 * @param {Object} token - The token object
 * @param {Array} allTokens - All tokens from the document
 * @param {number} contextSize - Number of tokens before/after to include
 * @returns {Object} - { before: Array, after: Array }
 */
export function getTokenContext(token, allTokens, contextSize = 3) {
  // Get all tokens on the same line
  const lineTokens = allTokens.filter(t => t.lineIndex === token.lineIndex)
  const tokenIndex = lineTokens.findIndex(t => t.id === token.id)

  if (tokenIndex === -1) {
    return { before: [], after: [] }
  }

  const before = lineTokens.slice(Math.max(0, tokenIndex - contextSize), tokenIndex)
  const after = lineTokens.slice(tokenIndex + 1, tokenIndex + 1 + contextSize)

  return { before, after }
}

/**
 * Find neighbors for a token (left, right, above, below)
 * @param {Object} token - The token to find neighbors for
 * @param {Array} allTokens - All tokens from the document
 * @param {boolean} hasBBox - Whether we have real bounding box coordinates
 * @returns {Object} - { left, right, above, below } with token or null
 */
export function findTokenNeighbors(token, allTokens, hasBBox = false) {
  if (!token || !allTokens || allTokens.length === 0) {
    return { left: null, right: null, above: null, below: null }
  }

  const neighbors = {
    left: null,
    right: null,
    above: null,
    below: null
  }

  if (hasBBox) {
    // Use real coordinates for more accurate neighbor detection
    const tokenCenterX = token.x + (token.width / 2)
    const tokenCenterY = token.y + (token.height / 2)

    let leftDist = Infinity, rightDist = Infinity
    let aboveDist = Infinity, belowDist = Infinity

    allTokens.forEach(other => {
      if (other.id === token.id) return

      const otherCenterX = other.x + (other.width / 2)
      const otherCenterY = other.y + (other.height / 2)
      const dx = otherCenterX - tokenCenterX
      const dy = otherCenterY - tokenCenterY

      // Same line (within threshold)
      if (Math.abs(dy) < SAME_LINE_THRESHOLD) {
        if (dx < 0 && Math.abs(dx) < leftDist) {
          leftDist = Math.abs(dx)
          neighbors.left = other
        }
        if (dx > 0 && dx < rightDist) {
          rightDist = dx
          neighbors.right = other
        }
      }

      // Different lines - check for above/below
      // Must be roughly aligned horizontally
      if (Math.abs(dx) < Math.max(token.width, other.width) * 1.5) {
        if (dy < 0 && Math.abs(dy) < aboveDist && Math.abs(dy) > SAME_LINE_THRESHOLD) {
          aboveDist = Math.abs(dy)
          neighbors.above = other
        }
        if (dy > 0 && dy < belowDist && dy > SAME_LINE_THRESHOLD) {
          belowDist = dy
          neighbors.below = other
        }
      }
    })
  } else {
    // Fallback: use line index and word position
    const sameLineTokens = allTokens
      .filter(t => t.lineIndex === token.lineIndex)
      .sort((a, b) => a.position - b.position)

    const tokenIndex = sameLineTokens.findIndex(t => t.id === token.id)

    if (tokenIndex > 0) {
      neighbors.left = sameLineTokens[tokenIndex - 1]
    }
    if (tokenIndex < sameLineTokens.length - 1) {
      neighbors.right = sameLineTokens[tokenIndex + 1]
    }

    // Find above/below by looking at adjacent lines
    const lineNumbers = [...new Set(allTokens.map(t => t.lineIndex))].sort((a, b) => a - b)
    const currentLineIdx = lineNumbers.indexOf(token.lineIndex)

    if (currentLineIdx > 0) {
      const aboveLine = lineNumbers[currentLineIdx - 1]
      const aboveTokens = allTokens.filter(t => t.lineIndex === aboveLine)
      // Find the closest token on the line above
      neighbors.above = findClosestInLine(token, aboveTokens)
    }

    if (currentLineIdx < lineNumbers.length - 1) {
      const belowLine = lineNumbers[currentLineIdx + 1]
      const belowTokens = allTokens.filter(t => t.lineIndex === belowLine)
      // Find the closest token on the line below
      neighbors.below = findClosestInLine(token, belowTokens)
    }
  }

  return neighbors
}

/**
 * Find the closest token in a line based on horizontal position
 * @param {Object} token - Reference token
 * @param {Array} lineTokens - Tokens on the target line
 * @returns {Object|null} - Closest token or null
 */
function findClosestInLine(token, lineTokens) {
  if (!lineTokens || lineTokens.length === 0) return null

  let closest = null
  let minDist = Infinity

  lineTokens.forEach(other => {
    const dist = Math.abs(other.position - token.position)
    if (dist < minDist) {
      minDist = dist
      closest = other
    }
  })

  return closest
}


