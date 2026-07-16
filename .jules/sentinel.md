## 2024-07-16 - Stored XSS via DOM textNode escaping bypass
**Vulnerability:** A stored XSS vulnerability existed in the frontend `escapeHTML` function.
**Learning:** The function used `document.createTextNode()` to escape HTML, which correctly escapes `<`, `>`, and `&`, but it fails to escape quotes (`"` and `'`). This allowed an attacker to break out of HTML attributes like `<a href="...">`.
**Prevention:** Always use regex replacements or robust libraries to escape HTML entities, ensuring that quotes (`"` and `'`) are properly replaced, especially when the escaped string is placed inside HTML attributes.
