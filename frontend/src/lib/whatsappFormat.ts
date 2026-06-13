// Converts a Markdown string into WhatsApp's lightweight formatting syntax.
// WhatsApp supports: *bold*, _italic_, ~strikethrough~, ```monospace blocks```,
// and `inline monospace` (single backticks render literally, so we keep them).
// This is best-effort: anything WhatsApp can't represent is flattened to plain text.

export function markdownToWhatsApp(md: string): string {
  if (!md) return ''
  const lines = md.replace(/\r\n/g, '\n').split('\n')
  const out: string[] = []
  let inFence = false

  for (let raw of lines) {
    const fenceMatch = raw.match(/^\s*```/)
    if (fenceMatch) {
      // Preserve fenced code blocks verbatim using WhatsApp's ``` syntax,
      // but drop the language tag on the opening fence.
      out.push('```')
      inFence = !inFence
      continue
    }
    if (inFence) {
      out.push(raw)
      continue
    }

    let line = raw

    // Headings -> bold line (strip leading #'s).
    const heading = line.match(/^\s{0,3}(#{1,6})\s+(.*)$/)
    if (heading) {
      line = `*${heading[2].trim()}*`
      out.push(line)
      continue
    }

    // Blockquote -> prefix with "> " kept (WhatsApp shows it literally, acceptable).
    line = line.replace(/^\s{0,3}>\s?/, '> ')

    // Unordered list bullets (-, *, +) -> "• ".
    line = line.replace(/^(\s*)[-*+]\s+/, '$1• ')

    // Inline formatting on the remaining text.
    line = applyInline(line)
    out.push(line)
  }

  return out.join('\n').trim()
}

function applyInline(text: string): string {
  // Protect inline code spans so their contents aren't reformatted.
  const codeSpans: string[] = []
  text = text.replace(/`([^`]+)`/g, (_m, code) => {
    codeSpans.push(code)
    return `\u0000C${codeSpans.length - 1}\u0000`
  })

  // Links [label](url) -> "label (url)".
  text = text.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '$1 ($2)')

  // Bold first, into a placeholder so the italic pass can't touch the single
  // asterisks we're about to emit. **x** / __x__ -> bold token.
  const bolds: string[] = []
  const stashBold = (inner: string) => {
    bolds.push(inner)
    return `\u0000B${bolds.length - 1}\u0000`
  }
  text = text.replace(/\*\*([^*]+)\*\*/g, (_m, inner) => stashBold(inner))
  text = text.replace(/__([^_]+)__/g, (_m, inner) => stashBold(inner))

  // Italic: *x* -> _x_  (single-asterisk emphasis). Underscore italic already
  // matches WhatsApp, so leave _x_ as-is.
  text = text.replace(/(^|[^*])\*([^*\n]+)\*(?!\*)/g, '$1_$2_')

  // Strikethrough: ~~x~~ -> ~x~
  text = text.replace(/~~([^~]+)~~/g, '~$1~')

  // Restore bold as WhatsApp *x*.
  text = text.replace(/\u0000B(\d+)\u0000/g, (_m, i) => '*' + bolds[Number(i)] + '*')
  // Restore protected code spans.
  text = text.replace(/\u0000C(\d+)\u0000/g, (_m, i) => '`' + codeSpans[Number(i)] + '`')

  return text
}
