`en.json` is the source of truth. `uk.json`, `es.json`, `pt.json` mirror its keys
exactly (required for next-intl fallback to stay silent) but most values are still the
English placeholder text — only the `LocaleSwitcher` language names are real
translations so far. Replace the placeholders with real translations as content is
reviewed; do not remove or rename keys without updating all four files together.
