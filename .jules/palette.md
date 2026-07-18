## 2024-07-18 - Restore Focus States & Context

**Learning:** `outline: none` without a fallback `:focus-visible` styling breaks keyboard navigation accessibility completely. Also, dynamic list controls (like a checkbox inside a card) need unique `aria-label`s, otherwise screen readers just announce "Checkbox, unchecked" with no indication of *which* item the checkbox controls.
**Action:** Always verify that interactive elements have a visible focus state when navigated via keyboard. For dynamic lists/grids, ensure each action element's accessible name includes the context of its parent item (e.g. `aria-label="Toggle monitoring for Server 1"`).
