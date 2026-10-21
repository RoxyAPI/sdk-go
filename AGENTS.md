# RoxyAPI Go SDK - Agent Guide

Go SDK for RoxyAPI. 18+ domains (Western astrology, Vedic astrology, forecast, human design, Chinese astrology, feng shui, Mesoamerican astrology, Vastu, numerology, Kabbalah, tarot, biorhythm, Ayurveda, I Ching, crystals, dreams, angel numbers, location) plus utility namespaces (usage, languages). One API key, fully typed, generated from the OpenAPI spec.

> `docs/llms-full.txt` in this module is the method index (operation id, HTTP method, path, summary for every endpoint). Response and request field names come from the typed Go structs: use editor autocomplete, https://pkg.go.dev/github.com/RoxyAPI/sdk-go, or the JSON shapes at https://roxyapi.com/api-reference.

## Install and initialize

```bash
go get github.com/RoxyAPI/sdk-go
```

```go
import roxyapi "github.com/RoxyAPI/sdk-go"

// Retries 429, 500, 502, 503 and 504 by default. Optional settings, shown with their defaults:
// roxyapi.NewRoxy(key, roxyapi.WithRetry(roxyapi.RetryConfig{MaxRetries: 2, InitialDelay: 500 * time.Millisecond, MaxDelay: 30 * time.Second}))
roxy, err := roxyapi.NewRoxy(os.Getenv("ROXY_API_KEY"))
```

`NewRoxy(apiKey)` sets the base URL (`https://roxyapi.com/api/v2`) and the auth and SDK headers. Every method returns a typed `*XxxResponse` whose `JSON200` holds the parsed success body, plus an `error` that is a `*roxyapi.RoxyError` on a 4xx or 5xx.

For a custom timeout, proxy, or transport, pass the generated option (the API key and SDK headers are still applied first):

```go
roxy, err := roxyapi.NewRoxy(key, roxyapi.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}))
```

One direct runtime dependency: `go get` pulls `github.com/oapi-codegen/runtime` (Apache 2.0); it brings two small transitive modules (`google/uuid`, `apapsch/go-jsonmerge`). The HTTP layer is the standard library `net/http`.

## Critical rule: geocode before any chart endpoint

Every chart, horoscope, panchang, dasha, dosha, navamsa, KP, synastry, compatibility, and natal endpoint needs `Latitude`, `Longitude`, and (for Western) `Timezone`. **Never ask the user for coordinates.** Always call `roxy.Location.SearchCities` first.

```go
search, err := roxy.Location.SearchCities(ctx, &roxyapi.SearchCitiesParams{Q: "New York"})
if err != nil || len(search.JSON200.Cities) == 0 { // a 200 can still return zero cities
	return err
}
city := search.JSON200.Cities[0] // fields: City, Country, Latitude, Longitude, Timezone (IANA), UtcOffset, Population
latitude, longitude, timezone := city.Latitude, city.Longitude, city.Timezone
// `city.Timezone` is the IANA string ("America/New_York"). Pass it directly to any chart endpoint.
// The server resolves it to the DST-correct decimal offset using the date of the chart itself,
// so a January 1990 New York chart gets EST (-5) even when you looked the city up in July.
// `city.UtcOffset` is the offset in force today, not on the birth date, so pass the IANA `timezone` string.
```

`Q` accepts a bare city (`"Paris"`), city plus country (`"Berlin Germany"`), or comma qualified (`"Springfield, Illinois"`). Add the state or country whenever the name is common; a `Total` above 1 means the name is ambiguous, so show `Province` and `Country` and let the user confirm.

## Domains

<!-- BEGIN:DOMAINS -->
| Accessor | What it covers |
|----------|----------------|
| [`roxy.Astrology`](https://roxyapi.com/api-reference/western-astrology) | Western astrology API for natal birth charts, daily, weekly, monthly, and yearly horoscopes with unique content per s... |
| [`roxy.VedicAstrology`](https://roxyapi.com/api-reference/vedic-astrology) | Vedic astrology (Jyotish) and KP API for kundli generation with the sixteen Shodasavarga divisional charts (D1 to D60... |
| [`roxy.Forecast`](https://roxyapi.com/api-reference/forecast) | Astrology forecast API that merges upcoming transit aspects, sign ingresses, retrograde stations, new and full moons,... |
| [`roxy.HumanDesign`](https://roxyapi.com/api-reference/human-design) | Human Design API that generates the full bodygraph from a birth moment: type, strategy, inner authority, profile, def... |
| [`roxy.ChineseAstrology`](https://roxyapi.com/api-reference/chinese-astrology) | Chinese zodiac and BaZi astrology API: Four Pillars charts, Chinese zodiac signs and the Chinese lunisolar calendar f... |
| [`roxy.FengShui`](https://roxyapi.com/api-reference/feng-shui) | Compute classical feng shui from one API: Xuan Kong flying star natal charts for any of the nine periods and 24 mount... |
| [`roxy.MesoamericanAstrology`](https://roxyapi.com/api-reference/mesoamerican-astrology) | Calculate Mayan astrology day signs, the Tzolkin sacred round, the Haab year, the full Long Count and the Aztec tonal... |
| [`roxy.Vastu`](https://roxyapi.com/api-reference/vastu) | Vastu Shastra API for directional home and plot analysis: entrance padas with the classical effect of each of the 32... |
| [`roxy.Numerology`](https://roxyapi.com/api-reference/numerology) | Numerology API for life path numbers, complete numerology charts and compatibility, in both Pythagorean and Chaldean... |
| [`roxy.Kabbalah`](https://roxyapi.com/api-reference/kabbalah) | Kabbalah API for gematria, the 72 names, the Tree of Life and the Hebrew birthday |
| [`roxy.Tarot`](https://roxyapi.com/api-reference/tarot) | Tarot reading API with the complete 78-card Rider-Waite-Smith deck and card meanings for love, career, health, and sp... |
| [`roxy.Biorhythm`](https://roxyapi.com/api-reference/biorhythm) | Biorhythm API for entertainment and reflection, with 10 model cycle types across 3 primary (physical, emotional, inte... |
| [`roxy.Ayurveda`](https://roxyapi.com/api-reference/ayurveda) | Ayurveda API for dosha profiles, the dinacharya daily routine and the ritucharya seasonal regimen, with a verse cited... |
| [`roxy.Iching`](https://roxyapi.com/api-reference/i-ching) | I-Ching oracle API with all 64 hexagrams, 384 changing lines, 8 trigrams, and modern interpretations for love, career... |
| [`roxy.Crystals`](https://roxyapi.com/api-reference/crystals-and-healing-stones) | Crystal healing API covering the most popular and widely-searched healing crystals and gemstones, from Amethyst and R... |
| [`roxy.Dreams`](https://roxyapi.com/api-reference/dreams) | Dream interpretation API with a 2,200+ symbol dream dictionary and psychological meanings covering animals, objects,... |
| [`roxy.AngelNumbers`](https://roxyapi.com/api-reference/angel-numbers) | Angel numbers API with meanings for 111, 222, 333, 444, 555, 666, 777, 888, 999, 1111, and 75+ sequences covering eve... |
| [`roxy.Location`](https://roxyapi.com/api-reference/location-and-timezone) | Timezone and location API with city search and geocoding across 235,000+ cities in 240+ countries, returning latitude... |
| [`roxy.Usage`](https://roxyapi.com/api-reference/usage) | Monitor your API usage, check rate limits, and track request consumption |
| [`roxy.Languages`](https://roxyapi.com/api-reference/languages) | List the response languages accepted by the `lang` query parameter on every i18n-aware endpoint |
<!-- END:DOMAINS -->

## Quality guidelines for agents

The rules to follow when writing any call with this SDK. Get these right and the generated types do the rest.

- **Methods are grouped by domain and named for the spec operation id in PascalCase.** `roxy.Astrology.GenerateNatalChart(...)`, `roxy.VedicAstrology.GenerateBirthChart(...)`. Never invent a name from the URL path or a guess; the full list is in `docs/llms-full.txt`, and every signature of a domain is one command away: `go doc github.com/RoxyAPI/sdk-go.AstrologyService`.
- **Argument order is `(ctx, pathParams..., params, body)`, but the arity varies.** `params` is a nilable `*XxxParams` of query parameters (it carries `Lang` on i18n endpoints); pass `nil` for none. POST endpoints add a typed `body` last. **An endpoint with no query parameters has NO `params` argument at all** (the exact list is under Gotchas). Do not pass a stray `nil` to those: it COMPILES (the last arg is variadic) then PANICS at runtime in applyEditors. Drop the argument. When unsure, let autocomplete show the signature.
- **The request body type is always `roxyapi.<MethodName>JSONRequestBody`** (some are aliases of a named request like `NatalChartRequest`; both names work). Build it as a struct literal. A body whose nested person, birth data or plot struct has no named type (`Person1`, `PersonA`, `BirthData`, `Plot`) is declared with `var` and filled field by field.
- **Read the success body from `JSON200`** (a typed struct, nil unless the call was a 2xx): `resp.JSON200.Cities[0].Latitude`. Response field names come from the generated structs, in Go casing (`LuckyNumber`, `ImageURL`, `IncarnationCross.Name`); `go build` fails on any invented field, so if it does not compile, the field does not exist. `resp.StatusCode()` and `resp.Bytes()` give the raw response.
- **Handle errors with `errors.As` on `*RoxyError`.** Every method returns `(resp, err)`; check `err` before touching `JSON200`. Switch on `Code` (stable), not `Message`. On a 400, range over `Issues`.
- **Look up any operation or field beyond this guide.** Query the combined OpenAPI spec at `https://roxyapi.com/api/v2/openapi.json` with the jq recipe in `https://roxyapi.com/AGENTS.md`, or search the keyless Docs MCP server at `https://roxyapi.com/mcp/docs` (one tool, `search_docs`).
- **Do not hand-roll requests.** No raw `net/http` calls against the API. The SDK injects auth, the base URL and typed responses, and retries twice by default (see Retries). Use `NewRoxy(key)` for the common case, or pass `WithHTTPClient` for a custom transport. `roxyapi.Date(1990, time.January, 15)` builds a date field; `roxyapi.Ptr(v)` sets any optional pointer field or query param (`Seed`, `Question`, `Lang`, `Limit`).

## Critical patterns

### Two-step pattern for coordinate-dependent endpoints

With `latitude`, `longitude` and `timezone` from the lookup above:

```go
birthDate, birthTime := roxyapi.Date(1990, time.January, 15), "14:30:00"

// Timezone is a per-request union: build it from the IANA string of the lookup.
var tz roxyapi.NatalChartRequest_Timezone
if err := tz.FromNatalChartRequestTimezone1(timezone); err != nil {
	return err
}
chart, err := roxy.Astrology.GenerateNatalChart(ctx, nil, roxyapi.NatalChartRequest{
	Date: birthDate, Time: birthTime, Latitude: latitude, Longitude: longitude, Timezone: tz,
})
```

One lookup feeds every domain. The same five values (`birthDate`, `birthTime`, `latitude`, `longitude`, `timezone`) are the body for `Astrology.GenerateNatalChart`, `VedicAstrology.GenerateBirthChart`, `VedicAstrology.GetCurrentDasha`, `Ayurveda.CalculateAyurvedicConstitution` and the `BirthData` of `Forecast.ForecastTransits`; the instant alone (`Date`, `Time`, `Timezone`) is the body for `HumanDesign.GenerateBodygraph`, `ChineseAstrology.GenerateBaziChart` and `Kabbalah.GenerateBirthProfile`. Never look the city up twice for one person. Only the `Timezone` union type differs per body (`BirthChartRequest_Timezone`, `GenerateBodygraphJSONBody_Timezone`, and so on), and the Vedic and Ayurveda bodies take it as a pointer (`Timezone: &tz`).

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
// Or .FromNatalChartRequestTimezone0(1) for a decimal offset.
if err := tz.FromNatalChartRequestTimezone1("Europe/Berlin"); err != nil {
	return err
}
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

<!-- BEGIN:LANGS -->
**Multi-language responses.** Interpretations are available in 10 languages: `en`, `tr`, `de`, `es`, `hi`, `pt`, `fr`, `ru`, `zh-Hans`, `zh-Hant`. Set `Lang` on the params struct of any supported endpoint with `roxyapi.Ptr(...)`; it defaults to `en`. Supported: `roxy.Astrology`, `roxy.VedicAstrology`, `roxy.Forecast`, `roxy.HumanDesign`, `roxy.ChineseAstrology`, `roxy.FengShui`, `roxy.MesoamericanAstrology`, `roxy.Vastu`, `roxy.Numerology`, `roxy.Kabbalah`, `roxy.Tarot`, `roxy.Biorhythm`, `roxy.Ayurveda`, `roxy.Iching`, `roxy.Crystals`, `roxy.AngelNumbers`, `roxy.Languages`. English-only: `roxy.Dreams`, `roxy.Location`, `roxy.Usage`.
<!-- END:LANGS -->

Each endpoint has its own `Lang` type (`roxyapi.GetDailyHoroscopeParamsLang("es")`). The two Chinese scripts currently ship on Chinese astrology and feng shui; every other domain answers those codes in English per field. Call `roxy.Languages.ListLanguages(ctx)` for the live list with display names.

## Common tasks

In the catalog order (Western astrology, Vedic astrology, forecast, Human Design, Chinese astrology, feng shui, Mesoamerican astrology, Vastu, numerology, Kabbalah, tarot, biorhythm, Ayurveda, I Ching, crystals, dreams, angel numbers, location, usage, languages). `birthDate`, `birthTime`, `latitude`, `longitude` and `timezone` are the five values from the two-step pattern above, and `tz` is the `Timezone` union built from `timezone` for that body. Body literals are abbreviated to their field names: write each as `Field: value`, in the type `roxyapi.<Method>JSONRequestBody` (or its named alias).

| Task | Code |
|------|------|
| Find city coordinates (do this first) | `roxy.Location.SearchCities(ctx, &roxyapi.SearchCitiesParams{Q: "Berlin"})` |
| Daily horoscope | `roxy.Astrology.GetDailyHoroscope(ctx, "aries", nil)` |
| Natal chart (Western) | `roxy.Astrology.GenerateNatalChart(ctx, nil, roxyapi.NatalChartRequest{Date, Time, Latitude, Longitude, Timezone})` |
| Synastry | `var b roxyapi.CalculateSynastryJSONRequestBody`, fill `b.Person1` and `b.Person2`, then `roxy.Astrology.CalculateSynastry(ctx, nil, b)` |
| Compatibility score | `var b roxyapi.CalculateCompatibilityJSONRequestBody`, fill `b.Person1` and `b.Person2`, then `roxy.Astrology.CalculateCompatibility(ctx, nil, b)` |
| Current moon phase | `roxy.Astrology.GetCurrentMoonPhase(ctx, nil)` |
| Transits | `var b roxyapi.TransitsRequest`, fill `b.NatalChart` (a pointer, `new(...)` first), then `roxy.Astrology.CalculateTransits(ctx, nil, b)` |
| Kundli (Vedic birth chart) | `roxy.VedicAstrology.GenerateBirthChart(ctx, nil, roxyapi.BirthChartRequest{Date, Time, Latitude, Longitude, Timezone: &tz})` |
| Panchang (detailed) | `roxy.VedicAstrology.GetDetailedPanchang(ctx, nil, roxyapi.GetDetailedPanchangJSONRequestBody{Date, Latitude, Longitude, Timezone: &tz})` |
| Choghadiya | `roxy.VedicAstrology.GetChoghadiya(ctx, roxyapi.GetChoghadiyaJSONRequestBody{Date, Latitude, Longitude, Timezone: &tz})` (no `params` argument) |
| Current dasha | `roxy.VedicAstrology.GetCurrentDasha(ctx, nil, roxyapi.GetCurrentDashaJSONRequestBody{Date, Time, Latitude, Longitude, Timezone: &tz})` |
| Mangal Dosha | `roxy.VedicAstrology.CheckManglikDosha(ctx, nil, roxyapi.ManglikRequest{Date, Time, Latitude, Longitude, Timezone: &tz})` |
| Guna Milan (matching) | `var b roxyapi.CompatibilityRequest`, fill `b.Person1` and `b.Person2`, then `roxy.VedicAstrology.CalculateGunMilan(ctx, nil, b)` |
| Navamsa (D9) | `roxy.VedicAstrology.GenerateNavamsa(ctx, nil, roxyapi.NavamsaRequest{Date, Time, Latitude, Longitude, Timezone: &tz})` |
| KP chart | `roxy.VedicAstrology.GenerateKpChart(ctx, nil, roxyapi.KPChartRequest{Date, Time, Latitude, Longitude, Timezone: &tz})` |
| KP ruling planets | `roxy.VedicAstrology.GetKpRulingPlanets(ctx, nil, roxyapi.GetKpRulingPlanetsJSONRequestBody{Latitude, Longitude, Timezone: &tz})` |
| Nakshatra detail | `roxy.VedicAstrology.GetNakshatra(ctx, "ashwini", nil)` |
| Transit forecast | `var b roxyapi.ForecastTransitsJSONRequestBody`, fill `b.BirthData`, `b.StartDate`, `b.EndDate`, then `roxy.Forecast.ForecastTransits(ctx, nil, b)` |
| Cross-domain timeline | `var b roxyapi.GenerateTimelineJSONRequestBody`, fill `b.BirthData`, `b.StartDate`, `b.EndDate`, then `roxy.Forecast.GenerateTimeline(ctx, nil, b)` |
| Human Design bodygraph | `roxy.HumanDesign.GenerateBodygraph(ctx, nil, roxyapi.GenerateBodygraphJSONRequestBody{Date, Time, Timezone})` |
| Human Design connection | `var b roxyapi.CalculateConnectionJSONRequestBody`, fill `b.PersonA` and `b.PersonB`, then `roxy.HumanDesign.CalculateConnection(ctx, nil, b)` |
| BaZi Four Pillars | `roxy.ChineseAstrology.GenerateBaziChart(ctx, nil, roxyapi.GenerateBaziChartJSONRequestBody{Date, Time, Timezone})` |
| Chinese zodiac animal | `roxy.ChineseAstrology.CalculateZodiacAnimal(ctx, nil, roxyapi.CalculateZodiacAnimalJSONRequestBody{Date})` |
| Almanac day (Tong Shu) | `roxy.ChineseAstrology.GetAlmanacDay(ctx, roxyapi.Date(2026, time.October, 1), nil)` |
| Kua number | `roxy.FengShui.CalculateKuaNumber(ctx, nil, roxyapi.CalculateKuaNumberJSONRequestBody{Date, Gender})` |
| Flying star natal chart | `roxy.FengShui.GenerateFlyingStarChart(ctx, nil, roxyapi.GenerateFlyingStarChartJSONRequestBody{Period, Facing})` |
| Tzolkin day sign | `roxy.MesoamericanAstrology.CalculateTzolkin(ctx, nil, roxyapi.CalculateTzolkinJSONRequestBody{Date})` |
| Full Maya chart | `roxy.MesoamericanAstrology.GenerateMayanChart(ctx, nil, roxyapi.GenerateMayanChartJSONRequestBody{Date})` |
| Vastu entrance | `var b roxyapi.CalculateEntrancePadaJSONRequestBody`, fill `b.Plot`, `b.Facing`, `b.DoorPosition`, then `roxy.Vastu.CalculateEntrancePada(ctx, nil, b)` |
| Vastu room compliance | `var b roxyapi.CalculateRoomComplianceJSONRequestBody`, `json.Unmarshal` the plot, facing and rooms shape into it, then `roxy.Vastu.CalculateRoomCompliance(ctx, nil, b)` |
| Life path number | `roxy.Numerology.CalculateLifePath(ctx, nil, roxyapi.CalculateLifePathJSONRequestBody{Year, Month, Day})` |
| Full numerology chart | `roxy.Numerology.GenerateNumerologyChart(ctx, nil, roxyapi.GenerateNumerologyChartJSONRequestBody{FullName, Year, Month, Day})` |
| Personal year | `roxy.Numerology.CalculatePersonalYear(ctx, nil, roxyapi.CalculatePersonalYearJSONRequestBody{Month, Day})` |
| Gematria | `roxy.Kabbalah.CalculateGematria(ctx, nil, roxyapi.CalculateGematriaJSONRequestBody{Text: roxyapi.Ptr(text)})` |
| Kabbalah birth profile | `roxy.Kabbalah.GenerateBirthProfile(ctx, nil, roxyapi.GenerateBirthProfileJSONRequestBody{Date, Time: roxyapi.Ptr(birthTime), Timezone})` |
| Daily tarot card | `roxy.Tarot.GetDailyCard(ctx, nil, roxyapi.GetDailyCardJSONRequestBody{Seed: roxyapi.Ptr(seed)})` |
| Three-card spread | `roxy.Tarot.CastThreeCard(ctx, nil, roxyapi.CastThreeCardJSONRequestBody{Question: roxyapi.Ptr(question)})` |
| Celtic Cross | `roxy.Tarot.CastCelticCross(ctx, nil, roxyapi.CastCelticCrossJSONRequestBody{Question: roxyapi.Ptr(question)})` |
| Yes / no tarot | `roxy.Tarot.CastYesNo(ctx, nil, roxyapi.CastYesNoJSONRequestBody{Question: roxyapi.Ptr(question)})` |
| Biorhythm reading | `roxy.Biorhythm.GetReading(ctx, nil, roxyapi.GetReadingJSONRequestBody{BirthDate})` |
| Daily biorhythm (seeded) | `roxy.Biorhythm.GetDailyBiorhythm(ctx, nil, roxyapi.GetDailyBiorhythmJSONRequestBody{Seed: roxyapi.Ptr(seed)})` |
| Biorhythm forecast | `roxy.Biorhythm.GetForecast(ctx, nil, roxyapi.GetForecastJSONRequestBody{BirthDate})` |
| Biorhythm compatibility | `var b roxyapi.CalculateBioCompatibilityJSONRequestBody`, set `b.Person1.BirthDate` and `b.Person2.BirthDate`, then `roxy.Biorhythm.CalculateBioCompatibility(ctx, nil, b)` |
| Ayurvedic constitution | `roxy.Ayurveda.CalculateAyurvedicConstitution(ctx, nil, roxyapi.AyurvedaConstitutionRequest{Date, Time, Latitude, Longitude, Timezone: &tz})` |
| Dinacharya | `roxy.Ayurveda.GetDinacharyaSchedule(ctx, nil, roxyapi.AyurvedaDinacharyaRequest{Date, Latitude, Longitude, Timezone: &tz})` |
| Daily hexagram | `roxy.Iching.GetDailyHexagram(ctx, nil, roxyapi.GetDailyHexagramJSONRequestBody{Seed: roxyapi.Ptr(seed)})` |
| Cast I Ching reading | `roxy.Iching.CastReading(ctx, nil)` |
| Hexagram detail | `roxy.Iching.GetHexagram(ctx, 1, nil)` |
| Crystal by zodiac | `roxy.Crystals.GetCrystalsByZodiac(ctx, "leo", nil)` |
| Crystal by chakra | `roxy.Crystals.GetCrystalsByChakra(ctx, "Heart", nil)` |
| Dream symbol lookup | `roxy.Dreams.GetDreamSymbol(ctx, "flying")` (no `params` argument) |
| Angel number meaning | `roxy.AngelNumbers.GetAngelNumber(ctx, "1111", nil)` |
| Universal number lookup | `roxy.AngelNumbers.AnalyzeNumberSequence(ctx, &roxyapi.AnalyzeNumberSequenceParams{Number: "1234"})` |
| Check API usage | `roxy.Usage.GetUsageStats(ctx)` (no `params` argument) |
| List supported languages | `roxy.Languages.ListLanguages(ctx)` (no `params` argument) |

## Field formats that trip agents

These are the fields AI agents most often get wrong. Copy the format column exactly.

| Field | Format | Good | Bad |
|-------|--------|------|-----|
| `Date` (and `BirthDate`, `StartDate`, ...) | `roxyapi.Date(year, month, day)` | `roxyapi.Date(1990, time.January, 15)` | `"1990-01-15"`, `time.Now()` |
| `Time` | 24 hour string with seconds | `"14:30:00"`, `"09:00:00"` | `"2:30 PM"`, `"14:30"` |
| `Timezone` | Union: `tz.From<Req>Timezone0(decimal)` or `Timezone1("IANA")` | `tz.FromNatalChartRequestTimezone1("America/New_York")` | a bare `5.5` or `"+0530"` |
| `Latitude` / `Longitude` | `float32` (some bodies take a pointer) | `40.7128`, `roxyapi.Ptr[float32](40.7128)` | `"40.7128"`, DMS strings |
| `sign` (horoscope path param) | Zodiac name, lowercase is canonical, any case is accepted | `"aries"`, `"Scorpio"` | `"1"`, `"xx"` |
| `Lang`, `Seed`, `Question` (optional) | `roxyapi.Ptr(value)` | `roxyapi.Ptr("user-42")` | a bare string for a pointer field |

`sign`, `Lang`, and similar enum-like values are open named-string types: an invalid value (`"xx"`) compiles and is rejected by the server as a `validation_error` (400), not by the compiler.

### Timezone cheat sheet (most-asked locations)

| Region | Decimal | Region | Decimal |
|--------|---------|--------|---------|
| UTC / London (winter) | `0` | Dubai | `4` |
| London (summer, BST) | `1` | Karachi | `5` |
| Berlin / Paris | `1` (winter) / `2` (summer) | Delhi (IST) | `5.5` |
| Istanbul | `3` | Kathmandu (NPT) | `5.75` |
| Moscow | `3` | Dhaka | `6` |
| Tehran | `3.5` (winter) / `4.5` (summer) | Bangkok | `7` |
| Adelaide | `9.5` (winter) / `10.5` (summer) | Singapore / Beijing | `8` |
| New York (EST / EDT) | `-5` / `-4` | Tokyo | `9` |
| Chicago (CST / CDT) | `-6` / `-5` | Sydney | `10` (winter) / `11` (summer) |
| Denver (MST / MDT) | `-7` / `-6` | Auckland | `12` (winter) / `13` (summer) |
| Los Angeles (PST / PDT) | `-8` / `-7` | Honolulu | `-10` |

DST matters. If the birth date falls inside a daylight-saving window, use the summer / DST offset, or pass the IANA string from the location lookup and let the server resolve it. India observes no DST, so a fixed `5.5` is always right there; anywhere else, a natal chart must carry the offset in force at the time of birth.

## Astrology domain gotchas for LLMs

LLMs hallucinate confidently in this category. These are the specific traps you will hit when writing client code:

- **Ayanamsa is server-side in Vedic.** LLMs default to tropical / Western math. Vedic endpoints apply sidereal Lahiri ayanamsa server-side. KP endpoints accept `Ayanamsa` of `kp-newcomb` (default), `kp-old`, `lahiri`, or `custom`. Do not try to "correct" server output by subtracting ayanamsa in client code.
- **Tithi count is 30, not 2.** 15 Shukla (waxing) plus 15 Krishna (waning). Older LLM training data conflates Purnima and Amavasya as single tithis. Our panchang response carries a `paksha` field (`"Shukla"` or `"Krishna"`) plus a tithi number, so there are 30 distinct tithis in a lunar month.
- **Rahu and Ketu are shadow points, not planets.** They do not appear in a real ephemeris. Endpoints accept `NodeType` of `"mean"` (smooth mean node, traditional Vedic default) or `"true"` (osculating node with perturbation corrections) to select which calculation to use.
- **Nakshatra count is 27.** Abhijit is sometimes treated as a 28th in some schools, but this API uses the standard 27. `roxy.VedicAstrology.ListNakshatras(ctx, nil)` returns a `JSON200` slice of length 27.
- **Retrograde is per-planet, not global.** Natal chart planets and Vedic `meta` include `IsRetrograde` (a bool) per planet. KP planet lists use `retrograde`. Never generate "Mercury retrograde globally" UI copy, check the specific planet in the response.
- **Tarot reversals are a product choice.** `AllowReversals: roxyapi.Ptr(false)` on a tarot draw means no reversed cards in that draw, period. It is not cosmically meaningful, it is a config flag.
- **Angel number lookup works for any positive integer.** Digit-root fallback covers non-canonical numbers. Do not generate validation logic that rejects anything other than `111` / `222` / `333`.
- **Seed-based daily endpoints are DETERMINISTIC per `(seed, date)` pair.** Same seed plus same date returns the same reading. This is by design for push-notification consistency. Do not describe it as "cached" or retry on stale responses.
- **Timezone affects Western calculations more than Vedic.** Western natal charts must respect DST at the time of birth. Vedic endpoints default to IST (`5.5`) which is DST-free. Pass the IANA `Timezone` string from the location lookup (`"America/New_York"`, `"Asia/Kolkata"`) directly to chart calls and the server resolves the DST-correct offset for the chart `Date`. Never pass the `UtcOffset` of the city record (the offset in force today) or the clock of the user right now.

## Error handling

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

`RoxyError` fields: `StatusCode int`, `Code string` (stable), `Message string`, `Issues []RoxyErrorIssue` (each `{Path, Message, Code, Expected}`), `Allow []string` (405), `DocURL string` (the `doc_url` link for this code), `Docs string` (domain link, present on a path-miss 404), `RetryAfter time.Duration` (zero when the response sent no `Retry-After`).

<!-- BEGIN:ERRORS -->
Every error returns `{ error, code, doc_url }`. Switch on `code`; `error` is readable text and may change. Each meaning below says whether and when to retry, and `Retry-After` carries the wait when the response sends it. Every code is documented at https://roxyapi.com/docs/errors.

| Status | Code | Meaning |
|--------|------|---------|
| 400 | `validation_error` | A field is missing or invalid. The body lists every failing field at once in issues, each with its path and the expected shape. |
| 400 | `bad_request` | The request was rejected as a whole, for example a body that is not valid JSON or a date range that ends before it starts. |
| 400 | `date_out_of_range` | A date lies outside the span the planetary positions are served for. The error names the supported dates. |
| 401 | `api_key_required` | No API key was sent. |
| 401 | `invalid_api_key` | The key sent is malformed or cannot be verified. |
| 401 | `api_key_revoked` | The key was valid once and has since been revoked from the account page. |
| 401 | `subscription_not_found` | The key verifies, but the subscription behind it no longer exists. |
| 401 | `subscription_inactive` | The subscription behind the key is cancelled, expired or suspended, or a trial is past its end date. |
| 401 | `publishable_key_not_allowed_on_mcp` | A publishable key was used on a Remote MCP server, which takes a secret key. |
| 401 | `invalid_client_ip` | A keyless request could not be attributed to a caller. |
| 401 | `unauthorized` | Authentication failed with no more specific reason to give. |
| 403 | `forbidden` | The key is valid, but this request is not permitted. |
| 403 | `origin_required` | A publishable key with an origin allowlist was used with no Origin header, as from a server or a native app. |
| 403 | `origin_not_allowed` | The calling site is not on the origin allowlist of this publishable key. |
| 404 | `not_found` | Nothing is served at this path, or the identifier in it does not exist. A path miss adds suggestions, the closest real endpoints. |
| 405 | `method_not_allowed` | The path exists for a different HTTP method. The body lists the accepted methods in allow, as does the Allow header. |
| 409 | `conflict` | The request cannot be applied in the current state. |
| 410 | `gone` | The path was removed permanently. |
| 413 | `payload_too_large` | The request body is larger than the API accepts. A body far over the cap is refused before it reaches the API and returns a bare 413 with no JSON body. |
| 415 | `unsupported_media_type` | A request with a body was sent without Content-Type: application/json. |
| 422 | `unprocessable_entity` | Every field is valid, but the combination cannot be computed. |
| 429 | `rate_limit_exceeded` | The monthly request allowance is used up. Retry-After gives the seconds to the reset. |
| 429 | `rate_limit_per_minute` | The per-minute limit of the plan was reached. Retry-After gives the seconds to the next minute, and the monthly allowance is not charged. |
| 429 | `rate_limited` | A rate limit was reached, reported without the plan detail. |
| 429 | `free_tier_exhausted` | A keyless request has used up its trial allowance. |
| 500 | `internal_error` | The request failed inside our service. Retry with exponential backoff. |
| 503 | `compute_saturated` | Heavy calculations are briefly at capacity. Send the same request again after Retry-After seconds. |
| Any | `error` | A fallback for a status with no more specific code. The status and the error field carry the signal. |
<!-- END:ERRORS -->

`*RoxyError` exposes `StatusCode`, `Code` (compare with the `roxyapi.ErrCode...` string constants, for example `roxyapi.ErrCodeRateLimitPerMinute`), `Message`, `DocURL` (from `doc_url`) and `RetryAfter` (`time.Duration`, zero when absent).

### Retries

`NewRoxy` retries twice by default (429, 500, 502, 503, 504 and failed connections), honouring `Retry-After`; `roxyapi.WithMaxRetries(0)` turns it off, `roxyapi.WithRetry(roxyapi.RetryConfig{MaxRetries, InitialDelay, MaxDelay})` tunes it, in any option order. When the wait exceeds `MaxDelay` the error is returned with `RetryAfter` set.

## MCP equivalents

Every method has a matching MCP tool. The remote MCP server per domain is at `https://roxyapi.com/mcp/{domain}` (Streamable HTTP, no stdio / no self-hosting). Tool names follow `{method}_{path_snake_case}`, for example:

- `POST /astrology/natal-chart` -> `post_astrology_natal_chart` on `/mcp/astrology`
- `GET /astrology/horoscope/{sign}/daily` -> `get_astrology_horoscope_sign_daily` on `/mcp/astrology`
- `POST /vedic-astrology/birth-chart` -> `post_vedic_astrology_birth_chart` on `/mcp/vedic-astrology`
- `POST /tarot/spreads/celtic-cross` -> `post_tarot_spreads_celtic_cross` on `/mcp/tarot`

Use the SDK for typed Go services. Use MCP for AI agents (Claude Desktop, Cursor MCP, OpenAI agents) where the agent selects tools based on user intent.

## Gotchas

- **Geocode first.** Any chart, panchang, synastry, compatibility, or natal endpoint needs coordinates. Call `roxy.Location.SearchCities` before the chart method.
- **Do not expose API keys client-side.** Call Roxy from server code only.
- **Switch on `rerr.Code`, not `rerr.Message`.** The message may change; the code is stable.
- **Some methods have no `params` argument** (see Quality guidelines). Passing `nil` to those compiles but PANICS at runtime (the trailing arg is a variadic request editor). Affected: every endpoint with no query parameters, not even `lang`. The list below is read from the generated facade on every release; the signature is the source of truth.

<!-- BEGIN:NOPARAMS -->
- `roxy.VedicAstrology.CalculateAshtakavarga(ctx, body)`
- `roxy.VedicAstrology.CalculateDrishti(ctx, body)`
- `roxy.VedicAstrology.CalculateParallels(ctx, body)`
- `roxy.VedicAstrology.CalculateTransit(ctx, body)`
- `roxy.VedicAstrology.GetChoghadiya(ctx, body)`
- `roxy.VedicAstrology.GetHeliacalVisibility(ctx, body)`
- `roxy.VedicAstrology.GetHora(ctx, body)`
- `roxy.VedicAstrology.GetKpDailyFinance(ctx, body)`
- `roxy.VedicAstrology.GetKpPlanetsInterval(ctx, body)`
- `roxy.VedicAstrology.GetKpPlanets(ctx, body)`
- `roxy.VedicAstrology.GetKpRasiChanges(ctx, body)`
- `roxy.VedicAstrology.GetKpSublordChanges(ctx, body)`
- `roxy.VedicAstrology.GetUpagrahaPositions(ctx, body)`
- `roxy.Dreams.GetDailyDreamSymbol(ctx, body)`
- `roxy.Dreams.GetDreamSymbol(ctx, id)`
- `roxy.Dreams.GetSymbolLetterCounts(ctx)`
- `roxy.Usage.GetUsageStats(ctx)`
- `roxy.Languages.ListLanguages(ctx)`
<!-- END:NOPARAMS -->

- **`Timezone` union type names vary:** `<Request>_Timezone` for a named body, `<Operation>JSONBody_Timezone` for an inline body (most POST endpoints). Cannot guess it? Write the field with any value and read the expected type from the compiler error, or use autocomplete.
- **Dates are the typed `roxyapi.Date`, so a malformed date cannot be built.** A `validation_error` from Go comes from a `Time` string that is not `HH:MM:SS`, a coordinate out of range, or an enum-like string such as `Lang` that the server does not know.
- **A city record has `City`, `Province`, `Country`, `Iso2`, `Latitude`, `Longitude`, `Timezone`, `UtcOffset` and `Population`.** The name is `City`, never `Name`.
- **`UtcOffset` on a city record is the offset in force today.** Pass the IANA `Timezone` string to chart calls so the server resolves the offset for the birth date.
- **A natal chart `Planets` is a list of objects** (a slice of anonymous structs). Find a body by `Name`: `for _, p := range chart.JSON200.Planets { if p.Name == "Moon" { fmt.Println(p.Sign, p.House); break } }`.
- **In a Vedic chart `meta.Moon.rashi` is a string and `meta.Moon.nakshatra` is an object** (`Name`, `Pada`, `Lord`). Display `kundli.JSON200.Meta["Moon"].Nakshatra.Name`.
- **A `validation_error` carries `Issues`, each with `Path` (the dot-separated field, such as `time` or `person1.date`), `Message` and `Code`.** Loop over them to show every failing field:
  ```go
  for _, iss := range rerr.Issues {
  	fmt.Println(iss.Path, iss.Message, iss.Code)
  }
  ```
- **`sign` is matched case-insensitively.** Lowercase is canonical.
- **Total requests on a persistent failure = 1 + retries.** The default `MaxRetries` of 2 makes 3 requests; `WithMaxRetries(0)` makes exactly one.
- **`NewRoxy` returns `*roxyapi.Roxy`** (the type for your own function signatures and struct fields) and returns an error on an empty API key, so a missing `ROXY_API_KEY` fails at construction, not as a confusing later 401.
- **A successful `SearchCities` can return zero cities.** Check `len(search.JSON200.Cities) == 0` before indexing `[0]`.
- **Person-pair, forecast and Vastu bodies use anonymous nested structs** (`CalculateSynastry`, `CalculateCompatibility`, `CalculateGunMilan`, `ForecastTransits`, `GenerateTimeline`, `CalculateConnection`, `CalculateEntrancePada` carry inline `Person1`/`PersonA`/`BirthData`/`Plot` structs). Declare the body with `var b roxyapi.<Method>JSONRequestBody` and assign `b.Person1.Date`, `b.Person1.Latitude` and so on, then build its `Timezone` in place with `b.Person1.Timezone.From<Method>JSONBodyPerson1Timezone1(timezone)` (a pointer field takes `new(...)` first). A slice of inline structs (`CalculateRoomCompliance` `Rooms`) is easiest to `json.Unmarshal` from the JSON shape at https://roxyapi.com/api-reference.
- **List endpoints return a paginated envelope**, `Total`, `Limit`, `Offset` plus a named slice (`Cities`, `Crystals`, `Hexagrams`, `Symbols`), never a bare slice. Pass `Limit: roxyapi.Ptr(64)` to widen a page; `ListHexagrams` defaults to 20 of 64 and `SearchCities` to 10.

## Links

- Full method reference: `docs/llms-full.txt` (bundled in this module)
- Documentation: https://roxyapi.com/docs
- Interactive API docs: https://roxyapi.com/api-reference
- Pricing and API keys: https://roxyapi.com/pricing
- MCP for AI agents: https://roxyapi.com/docs/mcp
- Templates: https://roxyapi.com/templates
- Go reference (typed structs and fields): https://pkg.go.dev/github.com/RoxyAPI/sdk-go
- Source and issues: https://github.com/RoxyAPI/sdk-go
- Other SDKs: TypeScript https://www.npmjs.com/package/@roxyapi/sdk, Python https://pypi.org/project/roxy-sdk/, PHP https://packagist.org/packages/roxyapi/sdk, .NET https://www.nuget.org/packages/RoxyApi.Sdk, Rust https://crates.io/crates/roxyapi
