# Golden-contract harness

Proves ms_home HOME responses match `digital_bff` structurally, across the request
matrix (surface × auth × preview × flags).

## Capture fixtures
For each scenario, save two files in `fixtures/`:

```
<case>.golden.json   # response from digital_bff
<case>.actual.json   # response from ms_home for the same request
```

Example capture:
```sh
# digital_bff (source of truth)
curl -s "$DIGITAL_BFF/content/page/es-mx/" > fixtures/web-home-anon.golden.json

# ms_home (same request/headers)
curl -s "$MS_HOME/content/page/es-mx/" > fixtures/web-home-anon.actual.json
```

Suggested matrix: `web-home-anon`, `web-home-logged`, `pocket-home-anon`,
`pocket-home-logged`, `web-home-preview`, `web-home-flagsoff`.

## Run
```sh
go test ./test/contract -run HomeParity
```
The diff is structural (object key order ignored, array order significant). With no
fixtures present the test skips. On mismatch it reports the first differing path.

> Note: known-pending features (external-provider strategies, personalization)
> will diff until ported — capture fixtures per phase as parity is reached.
