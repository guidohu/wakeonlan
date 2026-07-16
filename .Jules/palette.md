## 2024-05-15 - Missing ARIA Labels on Icon-only Buttons
**Learning:** Found that multiple icon-only buttons (`+` for Add Host, `x` for Close, trash icon for Delete) and switch toggles lacked `aria-label`s, making them difficult for screen reader users to understand.
**Action:** Always ensure that any interactive elements without visible text labels include a descriptive `aria-label`. When dealing with dynamic items (like a list of hosts), include specific identifiers (like the host name) in the `aria-label` to provide necessary context (e.g., `aria-label="Delete ${host.name}"`).
