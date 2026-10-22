/** The arrow keys that move between the tabs: up and down, or all four. */
export type Orientation = "vertical" | "both"

/**
 * Turns the tab list and the panels of a root into tabs: the roles, one visible panel, a
 * roving tab index, Home, End, and the arrow keys. Without the script, the root shows every
 * panel.
 */
export function enhanceTabs(root: HTMLElement, panelSelector: string, orientation: Orientation): void {
  const tablist = root.querySelector<HTMLElement>("[role=tablist]")
  const tabs = [...root.querySelectorAll<HTMLButtonElement>("[role=tab]")]
  const panels = [...root.querySelectorAll<HTMLElement>(panelSelector)]
  if (!tablist || tabs.length === 0) return

  const select = (index: number, focus: boolean) => {
    tabs.forEach((tab, i) => {
      const selected = i === index
      tab.setAttribute("aria-selected", String(selected))
      tab.tabIndex = selected ? 0 : -1
      panels[i].hidden = !selected
    })
    if (focus) tabs[index].focus()
  }

  root.classList.add("tabbed")
  tablist.hidden = false
  panels.forEach((panel, i) => {
    panel.setAttribute("role", "tabpanel")
    panel.setAttribute("aria-labelledby", tabs[i].id)
    panel.tabIndex = 0
  })
  tabs.forEach((tab, i) => {
    tab.addEventListener("click", () => select(i, false))
    tab.addEventListener("keydown", (event) => {
      const moves: Record<string, number> = { ArrowDown: i + 1, ArrowUp: i - 1, Home: 0, End: tabs.length - 1 }
      if (orientation === "both") Object.assign(moves, { ArrowRight: i + 1, ArrowLeft: i - 1 })
      if (!(event.key in moves)) return
      event.preventDefault()
      select((moves[event.key] + tabs.length) % tabs.length, true)
    })
  })
  select(0, false)
}
