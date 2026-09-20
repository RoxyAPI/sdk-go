# RoxyAPI Go SDK - Agent Guide

Go SDK for RoxyAPI. 18+ domains (Western astrology, Vedic astrology, forecast, human design, Chinese astrology, feng shui, Mesoamerican astrology, Vastu, numerology, Kabbalah, tarot, biorhythm, Ayurveda, I Ching, crystals, dreams, angel numbers, location) plus utility namespaces (usage, languages). One API key, fully typed, generated from the OpenAPI spec.

> `docs/llms-full.txt` in this module is the method index (operation id, HTTP method, path, summary for every endpoint). Response and request field names come from the typed Go structs: use editor autocomplete, https://pkg.go.dev/github.com/RoxyAPI/sdk-go, or the JSON shapes at https://roxyapi.com/api-reference.

## Install and initialize

```bash
go get github.com/RoxyAPI/sdk-go
```

```go
import roxyapi "github.com/RoxyAPI/sdk-go"

roxy, err := roxyapi.NewRoxy(os.Getenv("ROXY_API_KEY"))
```

`NewRoxy(apiKey)` sets the base URL (`https://roxyapi.com/api/v2`) and the auth and SDK headers. Every method returns a typed `*XxxResponse` whose `JSON200` holds the parsed success body, plus an `error` that is a `*roxyapi.RoxyError` on a 4xx or 5xx.

For a custom timeout, proxy, or transport, pass the generated option (the API key and SDK headers are still applied first):

```go
roxy, err := roxyapi.NewRoxy(key, roxyapi.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}))
```

## Quality guidelines for agents

Six rules to follow when writing any call with this SDK. Get these right and the generated types do the rest.

- **Methods are grouped by domain and named for the spec operation id in PascalCase.** `roxy.Astrology.GenerateNatalChart(...)`, `roxy.VedicAstrology.GenerateBirthChart(...)`. Never invent a name from the URL path or a guess; the full list is in `docs/llms-full.txt`, and every signature of a domain is one command away: `go doc github.com/RoxyAPI/sdk-go.AstrologyService`.
- **Argument order is `(ctx, pathParams..., params, body)`, but the arity varies.** `params` is a nilable `*XxxParams` of query parameters (it carries `Lang` on i18n endpoints); pass `nil` for none. POST endpoints add a typed `body` last. **An endpoint with no query parameters has NO `params` argument at all** (the exact list is under Go-specific gotchas). Do not pass a stray `nil` to those: it COMPILES (the last arg is variadic) then PANICS at runtime in applyEditors. Drop the argument. When unsure, let autocomplete show the signature.
- **The request body type is always `roxyapi.<MethodName>JSONRequestBody`** (some are aliases of a named request like `NatalChartRequest`; both names work). Build it as a struct literal. A body whose nested person, birth data or plot struct has no named type (`Person1`, `PersonA`, `BirthData`, `Plot`) is declared with `var` and filled field by field.
- **Read the success body from `JSON200`** (a typed struct, nil unless the call was a 2xx): `resp.JSON200.Cities[0].Latitude`. Response field names come from the generated structs, in Go casing (`LuckyNumber`, `ImageURL`, `IncarnationCross.Name`); `go build` fails on any invented field, so if it does not compile, the field does not exist. `resp.StatusCode()` and `resp.Bytes()` give the raw response.
- **Handle errors with `errors.As` on `*RoxyError`.** Every method returns `(resp, err)`; check `err` before touching `JSON200`. Switch on `Code` (stable), not `Message`. On a 400, range over `Issues`.
- **Do not hand-roll requests.** No raw `net/http` calls against the API. The SDK injects auth, the base URL and typed responses; it does not retry, so wrap calls you want retried. Use `NewRoxy(key)` for the common case, or pass `WithHTTPClient` for a custom transport. `roxyapi.Date(1990, time.January, 15)` builds a date field; `roxyapi.Ptr(v)` sets any optional pointer field or query param (`Seed`, `Question`, `Lang`, `Limit`).

## Critical rule: geocode before any chart endpoint

Every chart, horoscope, panchang, dasha, dosha, navamsa, KP, synastry, compatibility, and natal endpoint needs latitude, longitude, and (for Western) timezone. **Never ask the user for coordinates.** Call `roxy.Location.SearchCities` first, then feed the result straight into the chart call.

```go
search, err := roxy.Location.SearchCities(ctx, &roxyapi.SearchCitiesParams{Q: "New York"})
if err != nil || len(search.JSON200.Cities) == 0 { // a 200 can still return zero cities
	return err
}
city := search.JSON200.Cities[0] // fields: City, Country, Latitude, Longitude, Timezone (IANA), UtcOffset, Population
latitude, longitude, timezone := city.Latitude, city.Longitude, city.Timezone
birthDate, birthTime := roxyapi.Date(1990, time.January, 15), "14:30:00"

// Timezone is a per-request union. Build it with the IANA string ("America/New_York") from
// the lookup: the server resolves it to the DST-correct offset using the date of the chart
// itself, so a January 1990 New York chart picks EST (-5) even when you looked the city up
// in July. The decimal UtcOffset also works and produces an identical chart.
var tz roxyapi.NatalChartRequest_Timezone
_ = tz.FromNatalChartRequestTimezone1(timezone)

chart, err := roxy.Astrology.GenerateNatalChart(ctx, nil, roxyapi.NatalChartRequest{
	Date: birthDate, Time: birthTime, Latitude: latitude, Longitude: longitude, Timezone: tz,
})
```

One lookup feeds every domain. The same five values (`birthDate`, `birthTime`, `latitude`, `longitude`, `timezone`) are the body for `Astrology.GenerateNatalChart`, `VedicAstrology.GenerateBirthChart`, `VedicAstrology.GetCurrentDasha`, `Ayurveda.CalculateAyurvedicConstitution` and the `BirthData` of `Forecast.ForecastTransits`; the instant alone (`Date`, `Time`, `Timezone`) is the body for `HumanDesign.GenerateBodygraph`, `ChineseAstrology.GenerateBaziChart` and `Kabbalah.GenerateBirthProfile`. Never look the city up twice for one person. Only the `Timezone` union type differs per body (`BirthChartRequest_Timezone`, `GenerateBodygraphJSONBody_Timezone`, and so on), and the Vedic and Ayurveda bodies take it as a pointer (`Timezone: &tz`).

`Q` accepts a bare city (`"Paris"`), city plus country (`"Berlin Germany"`), or comma qualified (`"Springfield, Illinois"`). Add the state or country whenever the name is common; a `Total` above 1 means the name is ambiguous, so show `Province` and `Country` and let the user confirm.

## Domains

<!-- BEGIN:DOMAINS -->
| Accessor | What it covers |
|----------|----------------|
| `roxy.Astrology` | Western astrology API for natal birth charts, daily, weekly, monthly, and yearly horoscopes with unique content per s... |
| `roxy.VedicAstrology` | Vedic astrology (Jyotish) and KP API for kundli generation with the sixteen Shodasavarga divisional charts (D1 to D60... |
| `roxy.Forecast` | Astrology forecast API that merges upcoming transit aspects, sign ingresses, retrograde stations, new and full moons,... |
| `roxy.HumanDesign` | Human Design API that generates the full bodygraph from a birth moment: type, strategy, inner authority, profile, def... |
| `roxy.ChineseAstrology` | Chinese zodiac and BaZi astrology API: Four Pillars charts, Chinese zodiac signs and the Chinese lunisolar calendar f... |
| `roxy.FengShui` | Compute classical feng shui from one API: Xuan Kong flying star natal charts for any of the nine periods and 24 mount... |
| `roxy.MesoamericanAstrology` | Calculate Mayan astrology day signs, the Tzolkin sacred round, the Haab year, the full Long Count and the Aztec tonal... |
| `roxy.Vastu` | Vastu Shastra API for directional home and plot analysis: entrance padas with the classical effect of each of the 32... |
| `roxy.Numerology` | Numerology API to calculate life path, expression, soul urge, personality, and maturity numbers, with Pinnacle and Ch... |
| `roxy.Kabbalah` | Kabbalah API for gematria, the 72 names, the Tree of Life and the Hebrew birthday, from one key |
| `roxy.Tarot` | Tarot reading API with the complete 78-card Rider-Waite-Smith deck and card meanings for love, career, health, and sp... |
| `roxy.Biorhythm` | The most complete biorhythm API: 10 cycle types across 3 primary (physical, emotional, intellectual), 4 secondary (in... |
| `roxy.Ayurveda` | Ayurveda API for dosha profiles, the dinacharya daily routine and the ritucharya seasonal regimen, with a verse cited... |
| `roxy.Iching` | I-Ching oracle API with all 64 hexagrams, 384 changing lines, 8 trigrams, and modern interpretations for love, career... |
| `roxy.Crystals` | Crystal healing API covering the most popular and widely-searched healing crystals and gemstones, from Amethyst and R... |
| `roxy.Dreams` | Dream interpretation API with a 2,000+ symbol dream dictionary and psychological meanings covering animals, objects,... |
| `roxy.AngelNumbers` | Angel numbers API with meanings for 111, 222, 333, 444, 555, 666, 777, 888, 999, 1111, and 75+ sequences covering eve... |
| `roxy.Location` | Timezone and location API with city search and geocoding across 235,000+ cities in 240+ countries, returning latitude... |
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

<!-- BEGIN:LANGS -->
**Multi-language responses.** Interpretations are available in 10 languages: `en`, `tr`, `de`, `es`, `hi`, `pt`, `fr`, `ru`, `zh-Hans`, `zh-Hant`. Set `Lang` on the params struct of any supported endpoint with `roxyapi.Ptr(...)`; it defaults to `en`. Supported: `roxy.Astrology`, `roxy.VedicAstrology`, `roxy.Forecast`, `roxy.HumanDesign`, `roxy.ChineseAstrology`, `roxy.FengShui`, `roxy.MesoamericanAstrology`, `roxy.Vastu`, `roxy.Numerology`, `roxy.Kabbalah`, `roxy.Tarot`, `roxy.Biorhythm`, `roxy.Ayurveda`, `roxy.Iching`, `roxy.Crystals`, `roxy.AngelNumbers`, `roxy.Languages`. English-only: `roxy.Dreams`, `roxy.Location`, `roxy.Usage`.
<!-- END:LANGS -->

Each endpoint has its own `Lang` type (`roxyapi.GetDailyHoroscopeParamsLang("es")`). The two Chinese scripts currently ship on Chinese astrology and feng shui; every other domain answers those codes in English per field. Call `roxy.Languages.ListLanguages(ctx)` for the live list with display names.

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
| 401 | `subscription_not_found` | Key references non-existent subscription |
| 401 | `subscription_inactive` | Subscription cancelled, expired, or suspended |
| 401 | `api_key_revoked` | Key was deleted from the account |
| 404 | `not_found` | Resource not found |
| 4xx | `bad_request` and other status-derived codes | A client error the endpoint itself detected, such as a date window whose `endDate` precedes `startDate` |
| 429 | `rate_limit_exceeded` | Monthly quota reached |
| 500 | `internal_error` | Server error |

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
| UTC / London (winter) | `0` | Delhi (IST) | `5.5` |
| Berlin / Paris | `1` winter / `2` summer | Bangkok | `7` |
| New York (EST / EDT) | `-5` / `-4` | Singapore / Beijing | `8` |
| Los Angeles (PST / PDT) | `-8` / `-7` | Tokyo | `9` |

DST matters. If the birth date falls inside a daylight-saving window, use the summer / DST offset, or pass the IANA string from the location lookup and let the server resolve it. India observes no DST, so a fixed `5.5` is always right there; anywhere else, a natal chart must carry the offset in force at the time of birth.

## Astrology domain gotchas

LLMs hallucinate confidently here. The specific traps:

- **Ayanamsa is server side in Vedic.** Vedic endpoints apply sidereal Lahiri ayanamsa server side; KP endpoints take an ayanamsa value. Never subtract ayanamsa in client code.
- **Tithi count is 30, not 2.** 15 Shukla (waxing) plus 15 Krishna (waning).
- **Rahu and Ketu are shadow points, not planets.** They do not appear in a real ephemeris.
- **Nakshatra count is 27.**
- **Retrograde is per planet, not global.** Check the specific planet; never generate "Mercury retrograde globally" copy.
- **Seed based daily endpoints are deterministic per (seed, date).** Same seed plus same date returns the same reading, by design.

## Go-specific gotchas

- **Some methods have no `params` argument** (see Quality guidelines). Passing `nil` to those compiles but PANICS at runtime (the trailing arg is a variadic request editor). Affected: every endpoint with no query parameters, not even `lang`. The list below is read from the generated facade on every release; the signature is the source of truth.

<!-- BEGIN:NOPARAMS -->
- `roxy.VedicAstrology.CalculateAshtakavarga(ctx, body)`
- `roxy.VedicAstrology.CalculateDrishti(ctx, body)`
- `roxy.VedicAstrology.CalculateParallels(ctx, body)`
- `roxy.VedicAstrology.CalculateTransit(ctx, body)`
- `roxy.VedicAstrology.GetChoghadiya(ctx, body)`
- `roxy.VedicAstrology.GetEclipticCrossings(ctx, body)`
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
- **`NewRoxy` returns `*roxyapi.Roxy`** (the type for your own function signatures and struct fields) and returns an error on an empty API key, so a missing `ROXY_API_KEY` fails at construction, not as a confusing later 401.
- **A successful `SearchCities` can return zero cities.** Check `len(search.JSON200.Cities) == 0` before indexing `[0]`.
- **Person-pair, forecast and Vastu bodies use anonymous nested structs** (`CalculateSynastry`, `CalculateCompatibility`, `CalculateGunMilan`, `ForecastTransits`, `GenerateTimeline`, `CalculateConnection`, `CalculateEntrancePada` carry inline `Person1`/`PersonA`/`BirthData`/`Plot` structs). Declare the body with `var b roxyapi.<Method>JSONRequestBody` and assign `b.Person1.Date`, `b.Person1.Latitude` and so on, then build its `Timezone` in place with `b.Person1.Timezone.From<Method>JSONBodyPerson1Timezone1(timezone)` (a pointer field takes `new(...)` first). A slice of inline structs (`CalculateRoomCompliance` `Rooms`) is easiest to `json.Unmarshal` from the JSON shape at https://roxyapi.com/api-reference.
- **List endpoints return a paginated envelope**, `Total`, `Limit`, `Offset` plus a named slice (`Cities`, `Crystals`, `Hexagrams`, `Symbols`), never a bare slice. Pass `Limit: roxyapi.Ptr(64)` to widen a page; `ListHexagrams` defaults to 20 of 64 and `SearchCities` to 10.
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
