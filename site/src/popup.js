// Opens and closes the page's popups with motion in every browser. CSS alone
// can only play a popover's closing half where the browser keeps it in the
// top layer while it fades (the overlay transition); elsewhere it is removed
// at once. So this script takes the closing over: it makes each popover
// manual, plays the closing animation (data-closing in src/popup.css) on the
// Close button, Escape or a click outside, and only then hides it.
//
// The opening is held too. The first frame of an open popup, with the page
// blurred behind it, can take longer than the whole fade in Safari, so the
// fade was over before anything was drawn. data-opening keeps the popup clear
// for two frames, then lets the opening transition run. Without the script
// the popovers open and close on CSS alone, as before.
//
// Like detect.js it runs as part of one classic script, so nothing is
// declared at the top level.
document.addEventListener(
  'DOMContentLoaded',
  () => {
    const popups = [...document.querySelectorAll('[popover]')]
    if (popups.length === 0 || typeof HTMLElement.prototype.hidePopover !== 'function') return

    const isOpen = (popup) => popup.matches(':popover-open')

    const close = (popup) => {
      if (!isOpen(popup) || popup.hasAttribute('data-closing')) return
      popup.removeAttribute('data-opening')
      popup.setAttribute('data-closing', '')
      const done = () => {
        popup.hidePopover()
        popup.removeAttribute('data-closing')
      }
      // Reduced motion turns the animation off, and then there is no end to wait for.
      if (getComputedStyle(popup).animationName === 'none') return done()
      popup.addEventListener('animationend', done, { once: true })
    }

    for (const popup of popups) {
      popup.popover = 'manual'
      popup.addEventListener('beforetoggle', (event) => {
        if (event.newState === 'open') popup.setAttribute('data-opening', '')
      })
      popup.addEventListener('toggle', (event) => {
        if (event.newState !== 'open') return
        requestAnimationFrame(() => requestAnimationFrame(() => popup.removeAttribute('data-opening')))
      })
      for (const button of popup.querySelectorAll('[popovertargetaction="hide"]')) {
        button.addEventListener('click', (event) => {
          event.preventDefault()
          close(popup)
        })
      }
    }

    const open = () => popups.find(isOpen)

    document.addEventListener('keydown', (event) => {
      const popup = open()
      if (event.key !== 'Escape' || !popup) return
      event.preventDefault()
      close(popup)
    })

    // A click outside the open popup closes it and does nothing else, as the
    // browser's light dismiss would.
    document.addEventListener(
      'click',
      (event) => {
        const popup = open()
        if (!popup) return
        // A click on the backdrop lands on the popup itself, outside its box.
        const box = popup.getBoundingClientRect()
        const outside =
          event.clientX < box.left ||
          event.clientX > box.right ||
          event.clientY < box.top ||
          event.clientY > box.bottom
        if (popup.contains(event.target) && !(event.target === popup && outside)) return
        event.preventDefault()
        event.stopPropagation()
        close(popup)
      },
      true,
    )
  },
  { once: true },
)
