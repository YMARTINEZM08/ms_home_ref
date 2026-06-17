# Feature Flags

Flags arrive from the GLOBAL Contentstack entry (`global.feature_flags`) and stay
**runtime-configurable** (no redeploy to toggle). HOME-relevant flags:

| Flag | Effect on HOME |
|---|---|
| `personalization` | master switch: greeting, shortcuts, recommendations, `me` |
| `salesforce` | birthday campaign greeting + recommendations |
| `shopping_assistant` | shopping-assistant shortcut |
| `my_purchases` | buy-again shortcut |
| `groupby` | product enrichment / banner products |

## Environment gate
`PERSONALIZATION_ENABLED` (env) is ANDed with CMS `personalization`:
effective = env AND cms. Lets ops disable personalization per environment without a
CMS change. Implemented in `HomeService.mergeFeatureFlags`.
