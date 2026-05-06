import { onBeforeUnmount, onMounted, ref } from 'vue'

export type Theme = 'light' | 'dark'

function read(): Theme {
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

// useTheme returns a reactive ref reflecting the current 'dark' class on
// <html>. Updates whenever ThemeToggle (or any other code) toggles the class.
export function useTheme() {
  const theme = ref<Theme>(read())
  let observer: MutationObserver | null = null

  onMounted(() => {
    theme.value = read()
    observer = new MutationObserver(() => {
      const next = read()
      if (next !== theme.value) theme.value = next
    })
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    })
  })

  onBeforeUnmount(() => {
    observer?.disconnect()
    observer = null
  })

  return theme
}
