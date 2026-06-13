import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js/lib/core'
import DOMPurify from 'dompurify'

// Register a focused set of languages instead of the full highlight.js bundle
// (which would add ~700kB). Add more here as needed.
import javascript from 'highlight.js/lib/languages/javascript'
import typescript from 'highlight.js/lib/languages/typescript'
import python from 'highlight.js/lib/languages/python'
import go from 'highlight.js/lib/languages/go'
import json from 'highlight.js/lib/languages/json'
import bash from 'highlight.js/lib/languages/bash'
import xml from 'highlight.js/lib/languages/xml'
import css from 'highlight.js/lib/languages/css'
import sql from 'highlight.js/lib/languages/sql'
import rust from 'highlight.js/lib/languages/rust'
import java from 'highlight.js/lib/languages/java'
import markdown from 'highlight.js/lib/languages/markdown'
import yaml from 'highlight.js/lib/languages/yaml'

hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('typescript', typescript)
hljs.registerLanguage('python', python)
hljs.registerLanguage('go', go)
hljs.registerLanguage('json', json)
hljs.registerLanguage('bash', bash)
hljs.registerLanguage('shell', bash)
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('html', xml)
hljs.registerLanguage('css', css)
hljs.registerLanguage('sql', sql)
hljs.registerLanguage('rust', rust)
hljs.registerLanguage('java', java)
hljs.registerLanguage('markdown', markdown)
hljs.registerLanguage('yaml', yaml)

// Markdown renderer for the AI playground. Output from models is UNTRUSTED, so
// everything is sanitized with DOMPurify before it ever reaches the DOM.
const md = new MarkdownIt({
  html: false, // never trust raw HTML from a model
  linkify: true,
  breaks: true,
  highlight(code, lang): string {
    const language = lang && hljs.getLanguage(lang) ? lang : ''
    let highlighted: string
    try {
      highlighted = language
        ? hljs.highlight(code, { language, ignoreIllegals: true }).value
        : hljs.highlightAuto(code).value
    } catch {
      highlighted = md.utils.escapeHtml(code)
    }
    const label = language || 'text'
    // The raw code is stashed base64-encoded in a data attribute so the copy
    // button can recover the exact original text regardless of highlighting.
    const raw = b64Encode(code)
    return (
      `<div class="code-block" data-lang="${label}">` +
      `<div class="code-block__bar"><span class="code-block__lang">${label}</span>` +
      `<button class="code-block__copy" type="button" data-code="${raw}">Copy</button></div>` +
      `<pre class="hljs"><code>${highlighted}</code></pre>` +
      `</div>`
    )
  },
})

// Open links in the system browser (Wails) instead of navigating the app away.
const defaultLinkOpen =
  md.renderer.rules.link_open ||
  ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options))
md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  token.attrSet('target', '_blank')
  token.attrSet('rel', 'noopener noreferrer')
  return defaultLinkOpen(tokens, idx, options, env, self)
}

function b64Encode(s: string): string {
  try {
    return btoa(unescape(encodeURIComponent(s)))
  } catch {
    return ''
  }
}

export function b64Decode(s: string): string {
  try {
    return decodeURIComponent(escape(atob(s)))
  } catch {
    return ''
  }
}

export function renderMarkdown(text: string): string {
  const html = md.render(text || '')
  return DOMPurify.sanitize(html, {
    ADD_ATTR: ['target', 'rel', 'data-code', 'data-lang'],
    ADD_TAGS: ['button'],
  })
}
