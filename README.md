# RoxyAPI Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/RoxyAPI/sdk-go.svg)](https://pkg.go.dev/github.com/RoxyAPI/sdk-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/RoxyAPI/sdk-go)](https://goreportcard.com/report/github.com/RoxyAPI/sdk-go)
[![CI](https://github.com/RoxyAPI/sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/RoxyAPI/sdk-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

The official **Go SDK for [RoxyAPI](https://roxyapi.com)**, the typed astrology API for Go: 12+ insight domains and 160+ endpoints behind one API key, with ephemeris output [verified against NASA JPL Horizons](https://roxyapi.com/methodology) across 210 reference points. Western and Vedic astrology, numerology, tarot, human design, biorhythm, I Ching, crystals, dreams, and angel numbers, all from one `go get`. **Build anything, fast.**

A fully typed, idiomatic golang client generated from the live OpenAPI spec: one direct runtime dependency, the standard library `net/http` underneath, autocomplete on every endpoint and field, and bundled docs for AI coding agents.

## Why developers use Roxy

- **One key, every domain.** Twelve plus insight domains under a single subscription. No per product fees, no per request token weighting. One request is one quota unit.
- **Typed end to end.** Domain grouped methods, typed request bodies, typed responses, and one catchable error type. Your editor walks the whole API.
- **Lean.** Standard `net/http` and one direct dependency. No vendor cloud, no heavy framework.
- **Agent ready.** Bundled `AGENTS.md` and `docs/llms-full.txt`, plus a remote MCP server per domain.
- **Proof before pay.** The playground at https://roxyapi.com/api-reference returns real production responses.

## Start with one call

```bash
go get github.com/RoxyAPI/sdk-go
```

```go
roxy, err := roxyapi.NewRoxy(os.Getenv("ROXYAPI_KEY"))
resp, err := roxy.Astrology.GetDailyHoroscope(context.Background(), "aries", nil)
// resp.JSON200 holds the parsed body; a 4xx or 5xx is returned as *roxyapi.RoxyError.
```

`NewRoxy` sets the base URL (`https://roxyapi.com/api/v2`) and injects the auth and SDK headers on every request.

## Quickstart

Two small helpers do the fiddly work: `roxyapi.Date(y, m, d)` builds a date field, and `roxyapi.Ptr(v)` sets any optional pointer field.

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	roxyapi "github.com/RoxyAPI/sdk-go"
)

func main() {
	roxy, err := roxyapi.NewRoxy(os.Getenv("ROXYAPI_KEY"))
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	// Step 1: geocode the birth city (required for any chart endpoint). Use a full
	// country name, not an abbreviation ("London, United Kingdom", not "London, UK").
	search, err := roxy.Location.SearchCities(ctx, &roxyapi.SearchCitiesParams{Q: "London, United Kingdom"})
	if err != nil {
		panic(err)
	}
	if len(search.JSON200.Cities) == 0 {
		panic("no city matched the search") // a 200 can still return zero cities
	}
	city := search.JSON200.Cities[0] // City, Country, Latitude, Longitude, Timezone (IANA), UtcOffset

	// Step 2: Western natal chart. Timezone is a union; pass the IANA string from the geocode.
	var tz roxyapi.NatalChartRequest_Timezone
	_ = tz.FromNatalChartRequestTimezone1(city.Timezone)

	chart, err := roxy.Astrology.GenerateNatalChart(ctx, nil, roxyapi.NatalChartRequest{
		Date:     roxyapi.Date(1990, time.January, 15),
		Time:     "14:30:00",
		Latitude: city.Latitude, Longitude: city.Longitude, Timezone: tz,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(chart.JSON200)
}
```

## What you can build

Horoscope apps and daily push readings, natal and Vedic birth chart generators, compatibility and synastry matchers, numerology and tarot experiences, human design bodygraphs, transit and forecast timelines, biorhythm trackers, dream and angel number lookups, and AI agents that reason over any of it through MCP.

## Domains

Reach every domain through its accessor on the client returned by `NewRoxy`.

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

## Most-used endpoints

The highest-demand endpoints by domain, in the order you are most likely to ship them. Chart endpoints share a birth-data body: build `Date` with `roxyapi.Date(...)` and the `Timezone` union as in the Quickstart. Full catalog in the [API reference](https://roxyapi.com/api-reference); method index in [`docs/llms-full.txt`](https://github.com/RoxyAPI/sdk-go/blob/main/docs/llms-full.txt).

### 1. Western astrology (natal chart, daily horoscope, moon phase)

```go
// Natal chart. The #1 Western query, called on every onboarding.
chart, err := roxy.Astrology.GenerateNatalChart(ctx, nil, roxyapi.NatalChartRequest{
	Date: roxyapi.Date(1990, time.January, 15), Time: "14:30:00",
	Latitude: 40.7128, Longitude: -74.006, Timezone: tz, // tz built with FromNatalChartRequestTimezone1("America/New_York")
})

// Daily horoscope. Highest per-user call frequency, drives DAUs and push.
horo, err := roxy.Astrology.GetDailyHoroscope(ctx, "aries", nil)

// Current moon phase. Viral for wellness and cycle-tracking apps.
moon, err := roxy.Astrology.GetCurrentMoonPhase(ctx, nil)
```

### 2. Vedic astrology (kundli, dasha, dosha)

Vedic charts make `Timezone` an optional pointer (`Timezone: &tz`); omit it to default to IST.

```go
// Vedic kundli (D1 Rashi chart). Entry point for every Jyotish product.
kundli, err := roxy.VedicAstrology.GenerateBirthChart(ctx, nil, roxyapi.BirthChartRequest{
	Date: roxyapi.Date(1990, time.January, 15), Time: "14:30:00",
	Latitude: 28.6139, Longitude: 77.209, Timezone: &vtz,
})

// Current Vimshottari dasha.
dasha, err := roxy.VedicAstrology.GetCurrentDasha(ctx, nil, roxyapi.GetCurrentDashaJSONRequestBody{
	Date: roxyapi.Date(1990, time.January, 15), Time: "14:30:00",
	Latitude: 28.6139, Longitude: 77.209, Timezone: &dtz,
})

// Mangal (Manglik) Dosha. Note: this endpoint takes no params argument.
dosha, err := roxy.VedicAstrology.CheckManglikDosha(ctx, roxyapi.CheckManglikDoshaJSONRequestBody{
	Date: roxyapi.Date(1990, time.January, 15), Time: "14:30:00", Latitude: 28.6139, Longitude: 77.209,
})
```

### 3. Numerology (life path, full chart, personal year)

No birth time needed, the easiest domain to integrate.

```go
// Life Path. The #1 numerology keyword.
lp, err := roxy.Numerology.CalculateLifePath(ctx, nil, roxyapi.CalculateLifePathJSONRequestBody{Year: 1990, Month: 1, Day: 15})

// Full numerology chart: all core numbers from name plus birth date.
chart, err := roxy.Numerology.GenerateNumerologyChart(ctx, nil,
	roxyapi.GenerateNumerologyChartJSONRequestBody{FullName: "Jane Smith", Year: 1990, Month: 1, Day: 15})

// Personal Year. Drives January traffic spikes (Year optional, defaults to current).
py, err := roxy.Numerology.CalculatePersonalYear(ctx, nil, roxyapi.CalculatePersonalYearJSONRequestBody{Month: 1, Day: 15})
```

### 4. Tarot (daily card, three-card, yes / no)

```go
// Daily card. Seed per user for deterministic once-per-day behavior.
card, err := roxy.Tarot.GetDailyCard(ctx, nil, roxyapi.GetDailyCardJSONRequestBody{Seed: roxyapi.Ptr("user-42")})

// Three-card past-present-future spread.
three, err := roxy.Tarot.CastThreeCard(ctx, nil,
	roxyapi.CastThreeCardJSONRequestBody{Question: roxyapi.Ptr("My next quarter"), Seed: roxyapi.Ptr("user-42")})

// Yes / No. Impulse micro-query.
yn, err := roxy.Tarot.CastYesNo(ctx, nil, roxyapi.CastYesNoJSONRequestBody{Question: roxyapi.Ptr("Should I take the offer?")})
```

### 5. Human Design (bodygraph in one call)

```go
// Full bodygraph: type, strategy, authority, profile, centers, channels, gates.
// Timezone union: for an inline-body endpoint the type name uses JSONBody, not
// JSONRequestBody (see Gotchas). If unsure of the name, let autocomplete fill it.
var btz roxyapi.GenerateBodygraphJSONBody_Timezone
_ = btz.FromGenerateBodygraphJSONBodyTimezone1("America/New_York")
hd, err := roxy.HumanDesign.GenerateBodygraph(ctx, nil, roxyapi.GenerateBodygraphJSONRequestBody{
	Date: roxyapi.Date(1990, time.July, 4), Time: "10:12:00",
	Latitude: roxyapi.Ptr[float32](40.7128), Longitude: roxyapi.Ptr[float32](-74.006), Timezone: btz,
})
```

### 6. Biorhythm (daily check-in)

```go
// Physical, emotional, intellectual, intuitive, plus extended cycles.
bio, err := roxy.Biorhythm.GetDailyBiorhythm(ctx, nil,
	roxyapi.GetDailyBiorhythmJSONRequestBody{Seed: roxyapi.Ptr("user-1"), Date: roxyapi.Ptr(roxyapi.Date(2026, time.April, 23))})
```

### 7. I Ching (cast a reading, hexagram catalog)

```go
// Cast a reading: primary hexagram, changing lines, transformed hexagram.
reading, err := roxy.Iching.CastReading(ctx, nil)

// Catalog of all 64 hexagrams (cache once).
hexes, err := roxy.Iching.ListHexagrams(ctx, nil)
```

### 8. Crystals (by zodiac, birthstone)

```go
byZodiac, err := roxy.Crystals.GetCrystalsByZodiac(ctx, "scorpio", nil)
birthstone, err := roxy.Crystals.GetBirthstones(ctx, 4, nil) // month number
```

### 9. Dreams (symbol lookup, search)

```go
symbol, err := roxy.Dreams.GetDreamSymbol(ctx, "flying") // no params argument
results, err := roxy.Dreams.SearchDreamSymbols(ctx, &roxyapi.SearchDreamSymbolsParams{Q: roxyapi.Ptr("water")})
```

### 10. Angel numbers (meaning, universal lookup)

```go
angel, err := roxy.AngelNumbers.GetAngelNumber(ctx, "1111", nil)
anyNumber, err := roxy.AngelNumbers.AnalyzeNumberSequence(ctx, &roxyapi.AnalyzeNumberSequenceParams{Number: "4242"})
```

> Person-pair and forecast endpoints (`CalculateSynastry`, `CalculateGunMilan`, `GenerateTimeline`) take inline `Person1` / `Person2` / `BirthData` structs that are awkward to build as a Go literal. See the [API reference](https://roxyapi.com/api-reference) for the JSON shape.

## Built for AI agents

Every endpoint is also a remote MCP tool at `https://roxyapi.com/mcp/{domain}` (Streamable HTTP, no local setup). Use the SDK for typed Go services; use MCP for agents (Claude, Cursor, ChatGPT) that select tools from user intent. This module ships `AGENTS.md` and `docs/llms-full.txt` so coding agents read them straight from the module cache.

## Reliability

- Ephemeris output verified against NASA JPL Horizons. Methodology: https://roxyapi.com/methodology. Open benchmark (210 reference points vs JPL Horizons DE441): https://github.com/RoxyAPI/astrology-api-benchmark.
- Generated from the same OpenAPI spec that powers the live API, so the SDK never drifts from production.
- One catchable `RoxyError` with a stable `Code` for every failure.

## Gotchas

- **Argument arity varies.** Calls are `(ctx, pathParams..., params, body)`, but an endpoint with no query parameters has **no `params` argument** (for example `roxy.Usage.GetUsageStats(ctx)`, `roxy.Languages.ListLanguages(ctx)`). Passing a stray `nil` there is read as a request editor and panics. Let autocomplete show the signature.
- **The body type is `roxyapi.<MethodName>JSONRequestBody`.** Some are aliases of a named request (`NatalChartRequest`); both names compile.
- **`Date` and `Timezone` are typed.** Build a date with `roxyapi.Date(1990, time.January, 15)`, never a string. `Timezone` is a per-request union: `tz.From<Req>Timezone1("Europe/Berlin")` (IANA) or `From<Req>Timezone0(1)` (decimal offset).
- **Optional fields are pointers.** Use `roxyapi.Ptr(...)` for `Seed`, `Question`, `Lang`, `Limit`, and similar.
- **Enum-like strings are validated server-side.** `sign` and `Lang` are open string types; an invalid value compiles and comes back as a `validation_error` (400).
- **Read responses off `JSON200`** (`resp.JSON200.Cities[0].Latitude`). It is nil unless the call was a 2xx (errors are returned, not in the body).
- **Person-pair / forecast bodies use anonymous structs** (see note above) and are best built from the JSON shape in the API reference.

## FAQ

**Q: `SearchCities` returned 200 but `Cities[0]` panics, or my city is not found.**
A: A successful search can still return an empty `Cities` slice, so check `len(search.JSON200.Cities) == 0` before indexing. Two-letter country abbreviations are not matched: use the full country name (`"London, United Kingdom"`, not `"London, UK"`) or just the bare city (`"London"`).

**Q: I got `nil pointer dereference` in `applyEditors`. What did I do?**
A: You passed `nil` to an endpoint that has no query-parameters argument (`roxy.Usage.GetUsageStats`, `roxy.Languages.ListLanguages`, `roxy.Crystals.ListCrystalColors`, `roxy.Crystals.ListCrystalPlanets`, `roxy.Dreams.GetSymbolLetterCounts`). That `nil` is read as a request editor: the call compiles, then panics at runtime. Call them with `ctx` only, for example `roxy.Usage.GetUsageStats(ctx)`.

**Q: `NewRoxy` returned no error but every call is `401 api_key_required`.**
A: Make sure `ROXYAPI_KEY` is exported. `NewRoxy` returns an error for an empty key; a non-empty but wrong key only fails on the first request.

**Q: How do I build the `Timezone` when I cannot guess the union type name?**
A: The union type is `<Request>_Timezone` for a named request body (`NatalChartRequest_Timezone`) and `<Operation>JSONBody_Timezone` for an inline body (`GenerateBodygraphJSONBody_Timezone`). If you cannot guess it, write the `Timezone:` field with any value and read the expected type from the compiler error, or let autocomplete fill it. Build it with `.From<Type>Timezone1("IANA")` or `.From<Type>Timezone0(decimal)`.

**Q: What type does `NewRoxy` return, so I can store it or pass it to a function?**
A: `*roxyapi.Roxy`.

## Error handling and advanced use

Any 4xx or 5xx is returned as a `*RoxyError`. Switch on `Code` (stable); `Message` is human readable.

```go
resp, err := roxy.Astrology.GetDailyHoroscope(ctx, "aries", nil)
var rerr *roxyapi.RoxyError
if errors.As(err, &rerr) {
	switch rerr.Code {
	case "rate_limit_exceeded":
		// back off
	case "validation_error":
		for _, iss := range rerr.Issues { // each field that failed
			fmt.Printf("%s: %s\n", iss.Path, iss.Message)
		}
	}
}
```

Configure the client with the generated options: `roxyapi.WithBaseURL`, `roxyapi.WithHTTPClient` (any `*http.Client`, for timeouts or proxies), and `roxyapi.WithRequestEditorFn`. The underlying `ClientWithResponses` stays exported for advanced use.

```go
roxy, err := roxyapi.NewRoxy(key, roxyapi.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}))
```

## Keywords

go sdk, golang api client, astrology api, vedic astrology api, kundli api, horoscope api, numerology api, tarot api, human design api, biorhythm api, i ching api, dream interpretation api, angel numbers api, geocoding api, rest api client, ai agent sdk, mcp server.

## License

MIT. See [LICENSE](https://github.com/RoxyAPI/sdk-go/blob/main/LICENSE).
