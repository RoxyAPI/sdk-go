# RoxyAPI Go SDK - Agent Guide

Go SDK for RoxyAPI. 12+ domains (Western astrology, Vedic astrology, numerology, tarot, human design, forecast, biorhythm, I Ching, crystals, dreams, angel numbers, location) plus utility namespaces (usage, languages). One API key, fully typed, generated from the OpenAPI spec.

> `docs/llms-full.txt` in this module is the method index (operation id, HTTP method, path, summary for every endpoint). Response and request field names come from the typed Go structs: use editor autocomplete, https://pkg.go.dev/github.com/RoxyAPI/sdk-go, or the JSON shapes at https://roxyapi.com/api-reference.

## Install and initialize

```bash
go get github.com/RoxyAPI/sdk-go
```

```go
import roxyapi "github.com/RoxyAPI/sdk-go"

roxy, err := roxyapi.NewRoxy(os.Getenv("ROXYAPI_KEY"))
```

`NewRoxy(apiKey)` sets the base URL (`https://roxyapi.com/api/v2`) and the auth and SDK headers. Every method returns a typed `*XxxResponse` whose `JSON200` holds the parsed success body, plus an `error` that is a `*roxyapi.RoxyError` on a 4xx or 5xx.

For a custom timeout, proxy, or transport, pass the generated option (the API key and SDK headers are still applied first):

```go
roxy, err := roxyapi.NewRoxy(key, roxyapi.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}))
```

## Rules to get right

- **Methods are grouped by domain and named for the spec operation id.** `roxy.Astrology.GenerateNatalChart(...)`, `roxy.VedicAstrology.GenerateBirthChart(...)`. Never invent a name; the full list is in `docs/llms-full.txt`.
- **Argument order is `(ctx, pathParams..., params, body)`, but the arity varies.** `params` is a nilable `*XxxParams` of query parameters (it carries `Lang` on i18n endpoints); pass `nil` for none. POST endpoints add a typed `body` last. **An endpoint with no query parameters has NO `params` argument at all** (for example `roxy.Usage.GetUsageStats(ctx)` and `roxy.Languages.ListLanguages(ctx)`). Do not pass a stray `nil` to those: it COMPILES (the last arg is variadic) then PANICS at runtime in applyEditors. Call them with `ctx` only. When unsure, let autocomplete show the signature.
- **The request body type is always `roxyapi.<MethodName>JSONRequestBody`** (some are aliases of a named request like `NatalChartRequest`; both names work). Build it as a struct literal.
- **Read the success body from `JSON200`** (a typed struct, nil unless the call was a 2xx): `resp.JSON200.Cities[0].Latitude`. `resp.StatusCode()` and `resp.Bytes()` give the raw response.
- **Handle errors with `errors.As` on `*RoxyError`.** Switch on `Code` (stable), not `Message`. On a 400, range over `Issues`.
- **Use the helpers for the two fiddly field kinds.** `roxyapi.Date(1990, time.January, 15)` builds a date field; `roxyapi.Ptr(v)` sets any optional pointer field or query param (`Seed`, `Question`, `Lang`, `Limit`).

## Critical rule: geocode before any chart endpoint

Every chart, horoscope, panchang, dasha, dosha, navamsa, KP, synastry, compatibility, and natal endpoint needs latitude, longitude, and (for Western) timezone. **Never ask the user for coordinates.** Call `roxy.Location.SearchCities` first, then feed the result straight into the chart call.

```go
search, err := roxy.Location.SearchCities(ctx, &roxyapi.SearchCitiesParams{Q: "Berlin Germany"})
if err != nil || len(search.JSON200.Cities) == 0 { // a 200 can still return zero cities
	return err
}
city := search.JSON200.Cities[0] // fields: City, Country, Latitude, Longitude, Timezone (IANA), UtcOffset, Population

// Timezone is a union: build it with the IANA string from the geocode result.
var tz roxyapi.NatalChartRequest_Timezone
_ = tz.FromNatalChartRequestTimezone1(city.Timezone)

chart, err := roxy.Astrology.GenerateNatalChart(ctx, nil, roxyapi.NatalChartRequest{
	Date:     roxyapi.Date(1990, time.January, 15),
	Time:     "14:30:00",
	Latitude: city.Latitude, Longitude: city.Longitude, Timezone: tz,
})
```

`Q` accepts a bare city (`"Paris"`), city plus country (`"Berlin Germany"`), or comma qualified (`"Springfield, Illinois"`). Use the qualified form to disambiguate, with a full country name, not an abbreviation (`"London, United Kingdom"`, not `"London, UK"`).

## Domains

<!-- BEGIN:DOMAINS -->
| Accessor | What it covers |
|----------|----------------|
| `roxy.Astrology` | Western astrology API for natal birth charts, daily, weekly, and monthly horoscopes with unique content per sign, syn... |
| `roxy.VedicAstrology` | Vedic astrology (Jyotish) and KP API for kundli generation with 15 divisional charts (D1-D60), Ashtakoot Gun Milan ku... |
| `roxy.Forecast` | Merge upcoming transit aspects, sign ingresses, retrograde stations, new and full moons, biorhythm critical days, and... |
| `roxy.HumanDesign` | Generate the full Human Design bodygraph from a birth moment: type, strategy, inner authority, profile, definition, i... |
| `roxy.Numerology` | Numerology API to calculate life path, expression, soul urge, personality, and maturity numbers, with Pinnacle and Ch... |
| `roxy.Tarot` | Tarot reading API with the complete 78-card Rider-Waite-Smith deck and card meanings for love, career, health, and sp... |
| `roxy.Biorhythm` | The most complete biorhythm API: 10 cycle types across 3 primary (physical, emotional, intellectual), 4 secondary (in... |
| `roxy.Iching` | I-Ching oracle API with all 64 hexagrams, 384 changing lines, 8 trigrams, and modern interpretations for love, career... |
| `roxy.Crystals` | Crystal healing API covering the most popular and widely-searched healing crystals and gemstones, from Amethyst and R... |
| `roxy.Dreams` | Dream interpretation API with a 2,000+ symbol dream dictionary and psychological meanings covering animals, objects,... |
| `roxy.AngelNumbers` | Angel numbers API with meanings for 111, 222, 333, 444, 555, 666, 777, 888, 999, 1111, and 75+ sequences covering eve... |
| `roxy.Location` | City search and geocoding API with 23,000+ cities across 240+ countries, returning latitude, longitude, IANA timezone... |
| `roxy.Usage` | Monitor your API usage, check rate limits, and track request consumption |
| `roxy.Languages` | List the response languages accepted by the `lang` query parameter on every i18n-aware endpoint |
<!-- END:DOMAINS -->

## Building requests

### GET endpoints: path params positional, query params via the pointer struct

```go
// GET /astrology/horoscope/{sign}/daily
resp, err := roxy.Astrology.GetDailyHoroscope(ctx, "aries", nil)

// With a query parameter (lang, date, ...):
resp, err = roxy.Astrology.GetDailyHoroscope(ctx, "aries",
	&roxyapi.GetDailyHoroscopeParams{Lang: roxyapi.Ptr(roxyapi.GetDailyHoroscopeParamsLang("es"))})
```

The `sign` path parameter is a named string type, so a bare `"aries"` literal works.

### POST endpoints: typed body struct

Most high value endpoints (charts, spreads, calculations) are POST. Pass `nil` for params when you need no query parameters.

```go
import (
	"context"
	"time"

	roxyapi "github.com/RoxyAPI/sdk-go"
)

// The Timezone field is a per-request union. Build it with the generated From..0
// (decimal offset) or From..1 (IANA) method. The union TYPE NAME is not always
// guessable: a $ref body uses <Request>_Timezone (NatalChartRequest_Timezone); an
// inline body (most POST endpoints) uses <Operation>JSONBody_Timezone, e.g.
// GenerateBodygraphJSONBody_Timezone. If unsure, write the field with any value and
// read the expected type from the compiler error, or use autocomplete.
var tz roxyapi.NatalChartRequest_Timezone
_ = tz.FromNatalChartRequestTimezone1("Europe/Berlin") // or .FromNatalChartRequestTimezone0(1)

body := roxyapi.NatalChartRequest{
	Date:      roxyapi.Date(1990, time.January, 15),
	Time:      "14:30:00",
	Latitude:  52.52,
	Longitude: 13.405,
	Timezone:  tz,
}
resp, err := roxy.Astrology.GenerateNatalChart(ctx, nil, body)
```

Western charts require `Timezone`; Vedic charts make it an optional pointer (`Timezone: &tz`, or omit it to default to IST 5.5).

### Multi-language via the params pointer

Eight languages: `en`, `tr`, `de`, `es`, `fr`, `hi`, `pt`, `ru`. Defaults to `en`. Each endpoint has its own `Lang` type; set it with `roxyapi.Ptr(...)`. Call `roxy.Languages.ListLanguages(ctx)` for the live list. Supported domains: astrology, vedicAstrology, numerology, tarot, biorhythm, iching, crystals, angelNumbers. English only: dreams, location, usage, languages.

### Error handling

```go
resp, err := roxy.Astrology.GetDailyHoroscope(ctx, "aries", nil)
var rerr *roxyapi.RoxyError
if errors.As(err, &rerr) {
	fmt.Printf("%d %s: %s\n", rerr.StatusCode, rerr.Code, rerr.Message)
	for _, iss := range rerr.Issues { // populated on a 400
		fmt.Printf("  %s: %s\n", iss.Path, iss.Message)
	}
	return
}
```

`RoxyError` fields: `StatusCode int`, `Code string` (stable), `Message string`, `Issues []RoxyErrorIssue` (each `{Path, Message, Code, Expected}`), `Allow []string` (405), `Docs string`.

| Status | Code | When |
|--------|------|------|
| 400 | `validation_error` | Missing or invalid parameters (see `Issues`) |
| 401 | `api_key_required` | No API key provided |
| 401 | `invalid_api_key` | Key format invalid or tampered |
| 401 | `subscription_inactive` | Subscription cancelled, expired, or suspended |
| 404 | `not_found` | Resource not found |
| 429 | `rate_limit_exceeded` | Monthly quota reached |
| 500 | `internal_error` | Server error |

## Field formats that trip agents

| Field | Format | Good | Bad |
|-------|--------|------|-----|
| `Date` (and `BirthDate`, `StartDate`, ...) | `roxyapi.Date(year, month, day)` | `roxyapi.Date(1990, time.January, 15)` | `"1990-01-15"`, `time.Now()` |
| `Time` | 24 hour string with seconds | `"14:30:00"`, `"09:00:00"` | `"2:30 PM"`, `"14:30"` |
| `Timezone` | Union: `tz.From<Req>Timezone0(decimal)` or `Timezone1("IANA")` | `tz.FromNatalChartRequestTimezone1("America/New_York")` | a bare `5.5` or `"+0530"` |
| `Latitude` / `Longitude` | `float32` (some bodies take a pointer) | `40.7128`, `roxyapi.Ptr[float32](40.7128)` | `"40.7128"`, DMS strings |
| `sign` (horoscope path param) | Lowercase zodiac name | `"aries"`, `"scorpio"` | `"Aries"`, `"1"` |
| `Lang`, `Seed`, `Question` (optional) | `roxyapi.Ptr(value)` | `roxyapi.Ptr("user-42")` | a bare string for a pointer field |

`sign`, `Lang`, and similar enum-like values are open named-string types: an invalid value (`"Aries"`, `"xx"`) compiles and is rejected by the server as a `validation_error` (400), not by the compiler.

### Timezone cheat sheet (decimal offsets)

| Region | Decimal | Region | Decimal |
|--------|---------|--------|---------|
| UTC / London (winter) | `0` | Delhi / Kolkata (IST) | `5.5` |
| Berlin / Paris | `1` winter / `2` summer | Bangkok | `7` |
| New York (EST / EDT) | `-5` / `-4` | Singapore / Beijing | `8` |
| Los Angeles (PST / PDT) | `-8` / `-7` | Tokyo | `9` |

DST matters for Western charts: use the summer offset for a daylight saving birth date, or pass the IANA name and let the server resolve it. Vedic endpoints default to IST (`5.5`), which is DST free.

## Astrology domain gotchas

LLMs hallucinate confidently here. The specific traps:

- **Ayanamsa is server side in Vedic.** Vedic endpoints apply sidereal Lahiri ayanamsa server side; KP endpoints take an ayanamsa value. Never subtract ayanamsa in client code.
- **Tithi count is 30, not 2.** 15 Shukla (waxing) plus 15 Krishna (waning).
- **Rahu and Ketu are shadow points, not planets.** They do not appear in a real ephemeris.
- **Nakshatra count is 27.**
- **Retrograde is per planet, not global.** Check the specific planet; never generate "Mercury retrograde globally" copy.
- **Seed based daily endpoints are deterministic per (seed, date).** Same seed plus same date returns the same reading, by design.

## Go-specific gotchas

- **Some methods have no `params` argument** (see Rules). Passing `nil` to those compiles but PANICS at runtime (the trailing arg is a variadic request editor). Affected: `Usage.GetUsageStats`, `Languages.ListLanguages`, `Crystals.ListCrystalColors`, `Crystals.ListCrystalPlanets`, `Dreams.GetSymbolLetterCounts`.
- **`Timezone` union type names vary:** `<Request>_Timezone` for a named body, `<Operation>JSONBody_Timezone` for an inline body (most POST endpoints). Cannot guess it? Write the field with any value and read the expected type from the compiler error, or use autocomplete.
- **`NewRoxy` returns `*roxyapi.Roxy`** (the type for your own function signatures and struct fields) and returns an error on an empty API key, so a missing `ROXYAPI_KEY` fails at construction, not as a confusing later 401.
- **A successful `SearchCities` can return zero cities.** Check `len(search.JSON200.Cities) == 0` before indexing `[0]`.
- **Person-pair and forecast bodies use anonymous nested structs** (`CalculateSynastry`, `CalculateGunMilan`, `GenerateTimeline` carry inline `Person1`/`Person2`/`BirthData` structs). They are awkward to build as a Go literal; for those, see https://roxyapi.com/api-reference for the JSON shape.
- **`SearchCities` paginates** with `Limit` and `Offset` (`roxyapi.Ptr(20)`); the default page is 10.
- **One direct runtime dependency.** `go get` pulls `github.com/oapi-codegen/runtime` (Apache 2.0); it brings two small transitive modules (`google/uuid`, `apapsch/go-jsonmerge`). The HTTP layer is the standard library `net/http`.

## MCP equivalents

Every method has a matching remote MCP tool at `https://roxyapi.com/mcp/{domain}` (Streamable HTTP, no stdio, no self hosting). Tool names follow `{method}_{path_snake_case}`:

- `POST /astrology/natal-chart` -> `post_astrology_natal_chart` on `/mcp/astrology`
- `GET /astrology/horoscope/{sign}/daily` -> `get_astrology_horoscope_sign_daily` on `/mcp/astrology`

Use the SDK for typed Go services. Use MCP for AI agents (Claude, Cursor, ChatGPT) that select tools from user intent.

## Links

- Full method index: `docs/llms-full.txt` (bundled in this module)
- Go reference (typed structs and fields): https://pkg.go.dev/github.com/RoxyAPI/sdk-go
- Interactive API docs and JSON shapes: https://roxyapi.com/api-reference
- Pricing and API keys: https://roxyapi.com/pricing
- MCP for AI agents: https://roxyapi.com/docs/mcp
- TypeScript SDK: https://www.npmjs.com/package/@roxyapi/sdk | Python SDK: https://pypi.org/project/roxy-sdk/
