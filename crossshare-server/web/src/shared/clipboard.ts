export async function copyText(text: string): Promise<void> {
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text)
      return
    } catch {
      // fall back to textarea copy
    }
  }
  copyFallback(text)
}

function copyFallback(text: string): void {
  const ta = document.createElement('textarea')
  ta.value = text
  ta.setAttribute('readonly', '')
  ta.style.cssText = 'position:fixed;left:-9999px;top:0;opacity:0;font-size:16px'
  document.body.appendChild(ta)

  const isiOS = /ipad|iphone|ipod/i.test(navigator.userAgent)
  if (isiOS) {
    const range = document.createRange()
    range.selectNodeContents(ta)
    const sel = window.getSelection()
    sel?.removeAllRanges()
    sel?.addRange(range)
    ta.setSelectionRange(0, text.length)
  } else {
    ta.select()
  }

  try {
    const ok = document.execCommand('copy')
    if (!ok) {
      throw new Error('copy command failed')
    }
  } finally {
    document.body.removeChild(ta)
  }
}
