// Central registry of external project links (repo, demo, docs, funding).
// Keep these in one place so the demo banner, login screen, settings "About"
// card, and any future surface all point at the same canonical URLs.
export const LINKS = {
  github: 'https://github.com/sgiraz/homelog',
  githubIssues: 'https://github.com/sgiraz/homelog/issues',
  githubSponsors: 'https://github.com/sponsors/sgiraz',
  kofi: 'https://ko-fi.com/sgiraz',
  demo: 'https://demo.homelog.dev',
  landing: 'https://homelog.dev',
  docs: 'https://docs.homelog.dev',
}

// The privacy policy is published per language on the docs site. Takes the
// active locale so an Italian UI links to the Italian text.
export function privacyUrl(locale) {
  return locale === 'it' ? `${LINKS.docs}/it/privacy` : `${LINKS.docs}/privacy`
}
