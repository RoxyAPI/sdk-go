<p align="center">
  <a href="https://roxyapi.com">
    <img src="https://raw.githubusercontent.com/RoxyAPI/sdk-go/main/assets/hero.png" alt="RoxyAPI Go SDK, ship in an afternoon. The Spiritual OS layer for agentic AI. One key, flat pricing." width="100%">
  </a>
</p>

# RoxyAPI Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/RoxyAPI/sdk-go.svg)](https://pkg.go.dev/github.com/RoxyAPI/sdk-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/RoxyAPI/sdk-go)](https://goreportcard.com/report/github.com/RoxyAPI/sdk-go)
[![CI](https://github.com/RoxyAPI/sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/RoxyAPI/sdk-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

The official **Go SDK for [RoxyAPI](https://roxyapi.com)**, the typed astrology API for Go: 18+ insight domains and 258+ endpoints behind one API key, with ephemeris output [verified against NASA JPL Horizons](https://roxyapi.com/methodology) across 210 reference points. Western and Vedic astrology, forecast, human design, Chinese astrology, feng shui, Mesoamerican astrology, Vastu, numerology, Kabbalah, tarot, biorhythm, Ayurveda, I Ching, crystals, dreams, angel numbers, and location geocoding, all from one `go get`. **Build anything, fast.**

A fully typed, idiomatic golang client generated from the live OpenAPI spec: one direct runtime dependency, the standard library `net/http` underneath, autocomplete on every endpoint and field, and bundled docs for AI coding agents.

## Why developers use Roxy

- **One key, every domain.** Eighteen plus insight domains under a single subscription, with flat all-inclusive pricing.
- **Typed end to end.** Domain grouped methods, typed request bodies, typed responses, and one catchable error type. Your editor walks the whole API.
- **Lean.** Standard `net/http` and one direct dependency. No vendor cloud, no heavy framework.
- **Agent ready.** Bundled `AGENTS.md` and `docs/llms-full.txt`, plus a remote MCP server per domain.
- **Proof before pay.** The playground at https://roxyapi.com/api-reference returns real production responses.

## Start with one call

Get real product value with a single typed call. No setup beyond your API key.

```bash
go get github.com/RoxyAPI/sdk-go
```

```go
roxy, err := roxyapi.NewRoxy(os.Getenv("ROXY_API_KEY"))
if err != nil {
	panic(err)
}
horoscope, err := roxy.Astrology.GetDailyHoroscope(context.Background(), "aries", nil)
if err != nil {
	panic(err) // a 4xx or 5xx is returned as *roxyapi.RoxyError
}
fmt.Println(horoscope.JSON200.Overview, horoscope.JSON200.Love, horoscope.JSON200.LuckyNumber)
```

Then expand into charts, compatibility, numerology, tarot, and more.

## Quick start

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
	roxy, err := roxyapi.NewRoxy(os.Getenv("ROXY_API_KEY"))
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	// Step 1: geocode the birth city once. Every chart endpoint takes these three values.
	search, err := roxy.Location.SearchCities(ctx, &roxyapi.SearchCitiesParams{Q: "London"})
	if err != nil {
		panic(err)
	}
	if len(search.JSON200.Cities) == 0 {
		panic("no city matched the search") // a 200 can still return zero cities
	}
	city := search.JSON200.Cities[0] // City, Country, Latitude, Longitude, Timezone (IANA), UtcOffset

	// Step 2: a Western natal chart. Timezone is a per-request union; pass the IANA string
	// from the lookup ("Europe/London") and the server resolves it to the DST-correct
	// offset for the date of the chart.
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

	// Step 3: the same birth as a Vedic kundli. Same inputs, sidereal zodiac. Vedic bodies
	// take Timezone as an optional pointer (omit it to default to IST).
	var vtz roxyapi.BirthChartRequest_Timezone
	_ = vtz.FromBirthChartRequestTimezone1(city.Timezone)
	kundli, err := roxy.VedicAstrology.GenerateBirthChart(ctx, nil, roxyapi.BirthChartRequest{
		Date:     roxyapi.Date(1990, time.January, 15),
		Time:     "14:30:00",
		Latitude: city.Latitude, Longitude: city.Longitude, Timezone: &vtz,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(chart.JSON200.Ascendant.Sign, kundli.JSON200.Meta["Moon"].Rashi)
}
```

`NewRoxy` sets the base URL (`https://roxyapi.com/api/v2`) and injects the auth header and SDK identification header on every request. Every method returns `(resp, err)`: check `err` first, then read the typed body off `resp.JSON200` (see Error handling).

## What you can build

Horoscope apps and daily push readings, natal and Vedic birth chart generators, compatibility and synastry matchers, numerology and tarot experiences, human design bodygraphs, transit and forecast timelines, biorhythm trackers, dream and angel number lookups, and AI agents that reason over any of it through MCP.

## Domains

Reach every domain through its accessor on the client returned by `NewRoxy`.

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

## Most-used endpoints

The highest-demand endpoints by domain, in the order you are most likely to ship them. Every example below reads the same birth through a different domain, and every coordinate comes from one location lookup at the top: one API key, one lookup, and eighteen domains that compose into a single product instead of eighteen separate ones. Full catalog in the [API reference](https://roxyapi.com/api-reference); method index in [`docs/llms-full.txt`](https://github.com/RoxyAPI/sdk-go/blob/main/docs/llms-full.txt).

The blocks are written inside a function that returns `error`, so every call is followed by its `if err != nil` check. Each response is read off `JSON200`; the comment under a call names the fields as they sit there, and an optional field is a pointer (nil when absent). Each chart body carries its own `Timezone` union type, built from the IANA string with the generated `From...Timezone1` method as in the Quick start; bodies whose nested person or plot structs have no named type are filled field by field.

### Location first: one lookup feeds every chart

Every chart, horoscope, panchang, dasha, dosha, synastry and compatibility endpoint needs `Latitude`, `Longitude` and `Timezone`. Never ask users to type coordinates. Look the city up once and reuse the result in every domain below.

```go
// One lookup feeds every chart below. Timezone is the IANA name from the city
// record; the server resolves it to the DST-correct offset for the date of each chart.
place, err := roxy.Location.SearchCities(ctx, &roxyapi.SearchCitiesParams{Q: "New York"})
if err != nil {
	return err
}
city := place.JSON200.Cities[0]
latitude, longitude, timezone := city.Latitude, city.Longitude, city.Timezone
birthDate, birthTime := roxyapi.Date(1990, time.January, 15), "14:30:00"

// A second person for the two-chart calls (synastry, Guna Milan, Human Design connection).
london, err := roxy.Location.SearchCities(ctx, &roxyapi.SearchCitiesParams{Q: "London"})
if err != nil {
	return err
}
partner := london.JSON200.Cities[0] // Latitude, Longitude and Timezone for the second chart
partnerDate, partnerTime := roxyapi.Date(1992, time.July, 22), "09:00:00"
```

### 1. Western astrology API (natal chart, daily horoscope, synastry)

Natal chart products, daily horoscope features, dating and compatibility apps, and lunar-cycle wellness apps start here.

```go
// Natal chart. The most requested Western call, run once at onboarding.
// The body carries the latitude, longitude and timezone from the location lookup above.
var natalTz roxyapi.NatalChartRequest_Timezone
_ = natalTz.FromNatalChartRequestTimezone1(timezone)
natal, err := roxy.Astrology.GenerateNatalChart(ctx, nil, roxyapi.NatalChartRequest{
	Date: birthDate, Time: birthTime, Latitude: latitude, Longitude: longitude, Timezone: natalTz,
})
if err != nil {
	return err
}
// natal.JSON200.Planets[n].Name, .Sign, .House, .Interpretation.Summary; natal.JSON200.Ascendant.Sign; natal.JSON200.Aspects

// Daily horoscope. The highest per-user call frequency in the catalog: daily content, streaks, push.
horoscope, err := roxy.Astrology.GetDailyHoroscope(ctx, "aries", nil)
if err != nil {
	return err
}
// horoscope.JSON200.Overview, .Love, .Career, .Column, .Events, .LuckyNumber

// Synastry. Full inter-aspect analysis between two charts, the relationship feature of dating apps.
// Person1 and Person2 are inline structs, so the body is filled field by field from the two lookups above.
var synastryBody roxyapi.CalculateSynastryJSONRequestBody
synastryBody.Person1.Date, synastryBody.Person1.Time = birthDate, birthTime
synastryBody.Person1.Latitude, synastryBody.Person1.Longitude = latitude, longitude
_ = synastryBody.Person1.Timezone.FromCalculateSynastryJSONBodyPerson1Timezone1(timezone)
synastryBody.Person2.Date, synastryBody.Person2.Time = partnerDate, partnerTime
synastryBody.Person2.Latitude, synastryBody.Person2.Longitude = partner.Latitude, partner.Longitude
_ = synastryBody.Person2.Timezone.FromCalculateSynastryJSONBodyPerson2Timezone1(partner.Timezone)
synastry, err := roxy.Astrology.CalculateSynastry(ctx, nil, synastryBody)
if err != nil {
	return err
}
// synastry.JSON200.CompatibilityScore, .InterAspects, .Analysis.Strengths

// Moon phase. A zero-setup GET for wellness, cycle-tracking and meditation apps.
moon, err := roxy.Astrology.GetCurrentMoonPhase(ctx, nil)
if err != nil {
	return err
}
// moon.JSON200.Phase, .Illumination, .Sign, .Meaning.Description
```

### 2. Vedic astrology API (kundli, panchang, dasha, Guna Milan, KP)

Kundli generators, matrimonial matching, muhurta and panchang apps, and KP practitioners. The same birth values, read sidereally.

```go
// Vedic kundli. The same birth read sidereally: the body reuses the location lookup above.
// Vedic bodies take Timezone as an optional pointer; omit it to default to IST.
var kundliTz roxyapi.BirthChartRequest_Timezone
_ = kundliTz.FromBirthChartRequestTimezone1(timezone)
kundli, err := roxy.VedicAstrology.GenerateBirthChart(ctx, nil, roxyapi.BirthChartRequest{
	Date: birthDate, Time: birthTime, Latitude: latitude, Longitude: longitude, Timezone: &kundliTz,
})
if err != nil {
	return err
}
// kundli.JSON200.Meta["Moon"].Rashi, kundli.JSON200.Meta["Moon"].Nakshatra.Name, kundli.JSON200.Houses, .Combustion

// Detailed panchang. Tithi, nakshatra, yoga, karana, rahu kaal and the muhurtas for a date and place.
var panchangTz roxyapi.GetDetailedPanchangJSONBody_Timezone
_ = panchangTz.FromGetDetailedPanchangJSONBodyTimezone1(timezone)
panchang, err := roxy.VedicAstrology.GetDetailedPanchang(ctx, nil, roxyapi.GetDetailedPanchangJSONRequestBody{
	Date: roxyapi.Date(2026, time.October, 1), Latitude: latitude, Longitude: longitude, Timezone: &panchangTz,
})
if err != nil {
	return err
}
// panchang.JSON200.Tithi, .Nakshatra, .RahuKaal, .AbhijitMuhurta

// Vimshottari dasha. The mahadasha, antardasha and pratyantardasha running right now.
var dashaTz roxyapi.GetCurrentDashaJSONBody_Timezone
_ = dashaTz.FromGetCurrentDashaJSONBodyTimezone1(timezone)
dasha, err := roxy.VedicAstrology.GetCurrentDasha(ctx, nil, roxyapi.GetCurrentDashaJSONRequestBody{
	Date: birthDate, Time: birthTime, Latitude: latitude, Longitude: longitude, Timezone: &dashaTz,
})
if err != nil {
	return err
}
// dasha.JSON200.Mahadasha, .Antardasha, .RemainingInMahadasha

// Mangal Dosha. The most asked matrimonial check.
var doshaTz roxyapi.ManglikRequest_Timezone
_ = doshaTz.FromManglikRequestTimezone1(timezone)
dosha, err := roxy.VedicAstrology.CheckManglikDosha(ctx, nil, roxyapi.ManglikRequest{
	Date: birthDate, Time: birthTime, Latitude: latitude, Longitude: longitude, Timezone: &doshaTz,
})
if err != nil {
	return err
}
// dosha.JSON200.Present; Severity and Remedies are pointers, set only when Present is true

// Guna Milan. The 36-point Ashtakoota score behind kundli matching, both people from the lookups above.
var milanBody roxyapi.CompatibilityRequest
milanBody.Person1.Date, milanBody.Person1.Time = birthDate, birthTime
milanBody.Person1.Latitude, milanBody.Person1.Longitude = latitude, longitude
milanBody.Person1.Timezone = new(roxyapi.CompatibilityRequest_Person1_Timezone)
_ = milanBody.Person1.Timezone.FromCompatibilityRequestPerson1Timezone1(timezone)
milanBody.Person2.Date, milanBody.Person2.Time = partnerDate, partnerTime
milanBody.Person2.Latitude, milanBody.Person2.Longitude = partner.Latitude, partner.Longitude
milanBody.Person2.Timezone = new(roxyapi.CompatibilityRequest_Person2_Timezone)
_ = milanBody.Person2.Timezone.FromCompatibilityRequestPerson2Timezone1(partner.Timezone)
milan, err := roxy.VedicAstrology.CalculateGunMilan(ctx, nil, milanBody)
if err != nil {
	return err
}
// milan.JSON200.Total, .Percentage, .IsCompatible, .Breakdown

// KP ruling planets. Horary answers at the moment of the question, for the place looked up above.
var kpTz roxyapi.GetKpRulingPlanetsJSONBody_Timezone
_ = kpTz.FromGetKpRulingPlanetsJSONBodyTimezone1(timezone)
kp, err := roxy.VedicAstrology.GetKpRulingPlanets(ctx, nil, roxyapi.GetKpRulingPlanetsJSONRequestBody{
	Latitude: latitude, Longitude: longitude, Timezone: &kpTz,
})
if err != nil {
	return err
}
// kp.JSON200.DayLord, .MoonSublord, .RulingPlanets
```

### 3. Astrology forecast API (transit forecast, cross-domain timeline)

Forecast feeds, transit alerts and timing tools. One call returns a dated, significance-scored event list; the timeline variant merges Vedic dasha boundaries and biorhythm critical days into the same list, which no single-domain API can do.

```go
// Transit forecast. Transit-to-natal aspects, sign ingresses and retrograde stations over a window.
// BirthData is an inline struct carrying the same birth: date, time, latitude, longitude, timezone.
var transitsBody roxyapi.ForecastTransitsJSONRequestBody
transitsBody.BirthData.Date, transitsBody.BirthData.Time = birthDate, birthTime
transitsBody.BirthData.Latitude, transitsBody.BirthData.Longitude = roxyapi.Ptr(latitude), roxyapi.Ptr(longitude)
_ = transitsBody.BirthData.Timezone.FromForecastTransitsJSONBodyBirthDataTimezone1(timezone)
transitsBody.StartDate, transitsBody.EndDate = roxyapi.Ptr(roxyapi.Date(2026, time.October, 1)), roxyapi.Ptr(roxyapi.Date(2026, time.October, 31))
transits, err := roxy.Forecast.ForecastTransits(ctx, nil, transitsBody)
if err != nil {
	return err
}
// transits.JSON200.Count, transits.JSON200.Events[n].Date, .Type, .Body, .Target, .Aspect, .Significance

// Cross-domain timeline. The same window with Vedic dasha boundaries and biorhythm critical days merged in.
var timelineBody roxyapi.GenerateTimelineJSONRequestBody
timelineBody.BirthData.Date, timelineBody.BirthData.Time = birthDate, birthTime
timelineBody.BirthData.Latitude, timelineBody.BirthData.Longitude = roxyapi.Ptr(latitude), roxyapi.Ptr(longitude)
_ = timelineBody.BirthData.Timezone.FromGenerateTimelineJSONBodyBirthDataTimezone1(timezone)
timelineBody.StartDate, timelineBody.EndDate = roxyapi.Ptr(roxyapi.Date(2026, time.October, 1)), roxyapi.Ptr(roxyapi.Date(2026, time.October, 31))
timeline, err := roxy.Forecast.GenerateTimeline(ctx, nil, timelineBody)
if err != nil {
	return err
}
// timeline.JSON200.Events[n].Domain ("western", "vedic" or "biorhythm"), .Description, .Significance
```

### 4. Human Design API (bodygraph, connection)

Self-discovery apps, coaching bots and compatibility products. The full bodygraph is one call, and the Design side is solved on the exact 88-degree solar arc rather than approximated as calendar days.

```go
// Bodygraph. Type, strategy, authority, profile, definition, centers, channels and all 26 gates in one call.
// Human Design needs only the birth instant, so it takes the date, time and timezone from the lookup above.
var hdTz roxyapi.GenerateBodygraphJSONBody_Timezone
_ = hdTz.FromGenerateBodygraphJSONBodyTimezone1(timezone)
hd, err := roxy.HumanDesign.GenerateBodygraph(ctx, nil, roxyapi.GenerateBodygraphJSONRequestBody{
	Date: birthDate, Time: birthTime, Timezone: hdTz,
})
if err != nil {
	return err
}
// hd.JSON200.Type, .Strategy, .Authority, .Profile, .Definition, .IncarnationCross.Name, .Centers, .Channels, .Gates

// Connection. Two bodygraphs combined, each of the 36 channels classified by how the pair forms it.
var connectionBody roxyapi.CalculateConnectionJSONRequestBody
connectionBody.PersonA.Date, connectionBody.PersonA.Time = birthDate, birthTime
_ = connectionBody.PersonA.Timezone.FromCalculateConnectionJSONBodyPersonATimezone1(timezone)
connectionBody.PersonB.Date, connectionBody.PersonB.Time = partnerDate, partnerTime
_ = connectionBody.PersonB.Timezone.FromCalculateConnectionJSONBodyPersonBTimezone1(partner.Timezone)
connection, err := roxy.HumanDesign.CalculateConnection(ctx, nil, connectionBody)
if err != nil {
	return err
}
// connection.JSON200.TotalChannels, .Summary.Electromagnetic, .CombinedDefinition
```

### 5. Chinese zodiac API (BaZi four pillars, zodiac animal, almanac)

BaZi readings, zodiac content and Tong Shu date pages. The school splits that make two calculators disagree (`DayBoundary`, `YearBoundary`, `HourClock`) are typed request parameters with named defaults.

```go
// BaZi Four Pillars. The anchor call of the domain, from the same birth instant as every chart above.
// Each response echoes the Conventions it was computed under, so a chart can be reproduced, not guessed.
var baziTz roxyapi.GenerateBaziChartJSONBody_Timezone
_ = baziTz.FromGenerateBaziChartJSONBodyTimezone1(timezone)
bazi, err := roxy.ChineseAstrology.GenerateBaziChart(ctx, nil, roxyapi.GenerateBaziChartJSONRequestBody{
	Date: birthDate, Time: birthTime, Timezone: baziTz,
})
if err != nil {
	return err
}
// bazi.JSON200.Pillars[n].Position ("year", "month", "day" or "hour"), .Stem.Element, .Branch.Animal, .TenGod.Name
// bazi.JSON200.DayMaster.Element, .ZodiacAnimal, .FiveElements, .Conventions

// Chinese zodiac animal. Defaults YearBoundary to the Lunar New Year, the folk rule people mean
// when they ask which animal they are. Pass the li-chun value for the classical BaZi boundary.
animal, err := roxy.ChineseAstrology.CalculateZodiacAnimal(ctx, nil, roxyapi.CalculateZodiacAnimalJSONRequestBody{Date: birthDate})
if err != nil {
	return err
}
// animal.JSON200.Animal.Name, .Animal.Element, .Element (the year stem element), .Interpretation

// Almanac day. The Tong Shu view of a date: day officer, mansion, clash animal, favours and avoids.
// The date is a path parameter and typed, so it takes the same Date helper as a body field.
almanac, err := roxy.ChineseAstrology.GetAlmanacDay(ctx, roxyapi.Date(2026, time.October, 1), nil)
if err != nil {
	return err
}
// almanac.JSON200.DayPillar, .DayOfficer, .ClashAnimal, .Favours, .Avoids
```

### 6. Feng shui API (Kua number, flying star chart)

Kua numbers with the Eight Mansions map, Xuan Kong flying star charts for any of the nine periods and 24 mountains, annual and monthly star plates, and the annual afflictions.

```go
// Kua number. One birth date and a gender give the personal directions everything else reads off.
kua, err := roxy.FengShui.CalculateKuaNumber(ctx, nil, roxyapi.CalculateKuaNumberJSONRequestBody{
	Date: birthDate, Gender: roxyapi.CalculateKuaNumberJSONBodyGenderFemale,
})
if err != nil {
	return err
}
// kua.JSON200.Kua, .Group ("east" or "west"), .Trigram.English, kua.JSON200.Sectors[n].Direction, .Nature, .Rank

// Flying star natal chart. Period plus facing gives the nine palaces with base, mountain and water stars.
// Send Facing (a mountain id like Bing or a compass label like S2, both generated constants) or FacingDegrees, not neither.
stars, err := roxy.FengShui.GenerateFlyingStarChart(ctx, nil, roxyapi.GenerateFlyingStarChartJSONRequestBody{
	Period: roxyapi.Ptr(9), Facing: roxyapi.Ptr(roxyapi.GenerateFlyingStarChartJSONBodyFacingS2),
})
if err != nil {
	return err
}
// stars.JSON200.Facing.Label, .Sitting.Label, .Structure.Name, stars.JSON200.Palaces[n].Palace, .Base, .Mountain, .Water, .Reading
```

### 7. Mayan astrology API (Tzolkin day sign, full Maya chart)

Maya day signs, the Haab and Long Count, and the Aztec tonalpohualli, every value a function of the date under a typed `Correlation` convention echoed back in `Conventions`.

```go
// Tzolkin day sign. The most asked Maya question, answered from a date alone.
tzolkin, err := roxy.MesoamericanAstrology.CalculateTzolkin(ctx, nil, roxyapi.CalculateTzolkinJSONRequestBody{Date: birthDate})
if err != nil {
	return err
}
// tzolkin.JSON200.DaySign, .DaySignName, .Number, .Trecena, .Reading

// Full Maya chart. Tzolkin, Haab, Long Count, Calendar Round, Lord of the Night, Year Bearer and the Cruz Maya.
maya, err := roxy.MesoamericanAstrology.GenerateMayanChart(ctx, nil, roxyapi.GenerateMayanChartJSONRequestBody{Date: birthDate})
if err != nil {
	return err
}
// maya.JSON200.Tzolkin, .Haab, .LongCount, .CalendarRound, .YearBearer, .Cross, .Conventions.Correlation
```

### 8. Vastu Shastra API (entrance analysis, room compliance)

Home and plot analysis from typed geometry. Every verdict carries a `Source` object naming the text, chapter and verse it rests on, or a convention label where the texts are silent.

```go
// Entrance analysis. Plot, facing and door in; the pada, its devata, the classical effect and the recommended padas out.
// Plot is an inline struct, so the body is filled field by field; the enum values are generated constants.
var entranceBody roxyapi.CalculateEntrancePadaJSONRequestBody
entranceBody.Plot.Width, entranceBody.Plot.Depth = roxyapi.Ptr[float32](30), roxyapi.Ptr[float32](40)
entranceBody.Plot.Unit = roxyapi.Ptr(roxyapi.CalculateEntrancePadaJSONBodyPlotUnitFeet)
entranceBody.Facing = roxyapi.Ptr(roxyapi.CalculateEntrancePadaJSONBodyFacingNorth)
entranceBody.DoorPosition = roxyapi.Ptr[float32](0.4)
entrance, err := roxy.Vastu.CalculateEntrancePada(ctx, nil, entranceBody)
if err != nil {
	return err
}
// entrance.JSON200.Pada, .Devata, .Effect, .Auspiciousness, .RecommendedPadas, .Source

// Room compliance. A verdict per room with the verse or the convention it rests on, and a scored composite.
// Rooms is a slice of inline structs with no named type, so this body is unmarshalled from its JSON shape.
var roomsBody roxyapi.CalculateRoomComplianceJSONRequestBody
if err := json.Unmarshal([]byte(`{
	"plot": {"width": 30, "depth": 40, "unit": "feet"},
	"facing": "North",
	"rooms": [
		{"type": "kitchen", "direction": "Southeast"},
		{"type": "master-bedroom", "direction": "Southwest"},
		{"type": "puja", "direction": "Northeast"}
	]
}`), &roomsBody); err != nil {
	return err
}
rooms, err := roxy.Vastu.CalculateRoomCompliance(ctx, nil, roomsBody)
if err != nil {
	return err
}
// rooms.JSON200.Score, rooms.JSON200.Rooms[n].Type, .Verdict, .IdealDirections, .Source
```

### 9. Numerology API (life path, full chart, personal year)

Works from the birth date and name alone, no coordinates, which makes it the easiest domain to integrate.

```go
// Life Path. The most searched numerology number, from the birth date alone.
lifePath, err := roxy.Numerology.CalculateLifePath(ctx, nil, roxyapi.CalculateLifePathJSONRequestBody{Year: 1990, Month: 1, Day: 15})
if err != nil {
	return err
}
// lifePath.JSON200.Number, .Type ("single" or "master"), .Meaning

// Full numerology chart. All six core numbers plus karmic lessons, pinnacles and the personal year in one call.
numerology, err := roxy.Numerology.GenerateNumerologyChart(ctx, nil, roxyapi.GenerateNumerologyChartJSONRequestBody{
	FullName: "Jane Smith", Year: 1990, Month: 1, Day: 15,
})
if err != nil {
	return err
}
// numerology.JSON200.CoreNumbers.LifePath, .Expression, .SoulUrge, numerology.JSON200.AdditionalInsights.PersonalYear

// Personal Year. The annual theme, the January feature of every numerology app.
personalYear, err := roxy.Numerology.CalculatePersonalYear(ctx, nil, roxyapi.CalculatePersonalYearJSONRequestBody{Month: 1, Day: 15, Year: roxyapi.Ptr(2026)})
if err != nil {
	return err
}
// personalYear.JSON200.PersonalYear, .Theme, .Advice
```

### 10. Kabbalah API (gematria, birth profile)

Gematria of a Latin name under a declared transliteration convention, the 72 names, the Tree of Life, and a Hebrew birthday computed from the same birth instant as every chart above.

```go
// Gematria. A Latin name transliterated under a declared convention, ten ciphers, each with its tradition and source.
gematria, err := roxy.Kabbalah.CalculateGematria(ctx, nil, roxyapi.CalculateGematriaJSONRequestBody{Text: roxyapi.Ptr("Sarah")})
if err != nil {
	return err
}
// gematria.JSON200.Chosen.Hebrew, gematria.JSON200.Values[n].ID, .Name, .Value, .Tradition; gematria.JSON200.Matches, .Conventions

// Birth profile. The Hebrew date and birthday, the three birth angels and the birth sephirah from the instant above.
var kabbalahTz roxyapi.GenerateBirthProfileJSONBody_Timezone
_ = kabbalahTz.FromGenerateBirthProfileJSONBodyTimezone1(timezone)
kabbalah, err := roxy.Kabbalah.GenerateBirthProfile(ctx, nil, roxyapi.GenerateBirthProfileJSONRequestBody{
	Date: birthDate, Time: roxyapi.Ptr(birthTime), Timezone: kabbalahTz,
})
if err != nil {
	return err
}
// kabbalah.JSON200.HebrewDate, .HebrewBirthday, .Angels, .Sephirah
```

### 11. Tarot API (daily card, three-card, Celtic Cross, yes or no)

The complete 78-card deck with meanings for love, career, health and spirit. Pass a `Seed` per user for deterministic once-per-day draws.

```go
// Daily card. Deterministic per (seed, date), so one user sees one card per day.
card, err := roxy.Tarot.GetDailyCard(ctx, nil, roxyapi.GetDailyCardJSONRequestBody{Seed: roxyapi.Ptr("user-42")})
if err != nil {
	return err
}
// card.JSON200.Card.Name, .Card.Reversed, .Card.ImageURL, .DailyMessage

// Three-card spread. Past, present, future: the most drawn spread on every tarot platform.
three, err := roxy.Tarot.CastThreeCard(ctx, nil, roxyapi.CastThreeCardJSONRequestBody{Question: roxyapi.Ptr("My next quarter"), Seed: roxyapi.Ptr("user-42")})
if err != nil {
	return err
}
// three.JSON200.Positions[n].Name, .Card.Name, .Interpretation; three.JSON200.Summary

// Celtic Cross. The ten-position professional reading.
celtic, err := roxy.Tarot.CastCelticCross(ctx, nil, roxyapi.CastCelticCrossJSONRequestBody{Question: roxyapi.Ptr("What should I focus on?"), Seed: roxyapi.Ptr("user-42")})
if err != nil {
	return err
}
// celtic.JSON200.Positions[n].Name, .Card.Name, .Interpretation; celtic.JSON200.Summary

// Yes or no. One card, one answer, with its strength.
answer, err := roxy.Tarot.CastYesNo(ctx, nil, roxyapi.CastYesNoJSONRequestBody{Question: roxyapi.Ptr("Should I take the offer?")})
if err != nil {
	return err
}
// answer.JSON200.Answer ("Yes", "No" or "Maybe"), .Strength, .Card.Name
```

### 12. Biorhythm API (reading, forecast)

Ten cycle types across primary, secondary and extended cycles, for wellness, productivity, sports and couples apps.

```go
// Biorhythm reading. All ten cycles for a date, from the same birth date as every chart above.
bio, err := roxy.Biorhythm.GetReading(ctx, nil, roxyapi.GetReadingJSONRequestBody{BirthDate: birthDate, TargetDate: roxyapi.Ptr(roxyapi.Date(2026, time.October, 1))})
if err != nil {
	return err
}
// bio.JSON200.Cycles["physical"].Value, .Phase; bio.JSON200.EnergyRating, .OverallPhase, .CriticalAlerts, .Interpretation

// Forecast. Every cycle for every day of a window, with the best and worst days named.
bioForecast, err := roxy.Biorhythm.GetForecast(ctx, nil, roxyapi.GetForecastJSONRequestBody{
	BirthDate: birthDate, StartDate: roxyapi.Ptr(roxyapi.Date(2026, time.October, 1)), EndDate: roxyapi.Ptr(roxyapi.Date(2026, time.October, 31)),
})
if err != nil {
	return err
}
// bioForecast.JSON200.Summary.BestDay, .WorstDay, .AverageEnergy; bioForecast.JSON200.Days[n].Date, .Physical, .Emotional, .Intellectual, .IsCritical
```

### 13. Ayurveda API (dosha constitution, dinacharya)

The dosha profile read from a verified sidereal chart with the verse on each factor, a daily routine anchored on the local sunrise, and the six seasons from real solar ingresses. Every response carries `Meta.Disclaimer`.

```go
// Constitution. The dosha profile read from the sidereal chart of the same birth, each factor with its verse.
var constitutionTz roxyapi.AyurvedaConstitutionRequest_Timezone
_ = constitutionTz.FromAyurvedaConstitutionRequestTimezone1(timezone)
constitution, err := roxy.Ayurveda.CalculateAyurvedicConstitution(ctx, nil, roxyapi.AyurvedaConstitutionRequest{
	Date: birthDate, Time: birthTime, Latitude: latitude, Longitude: longitude, Timezone: &constitutionTz,
})
if err != nil {
	return err
}
// constitution.JSON200.Composite.Dominant, .Type; constitution.JSON200.Factors[n].ID, .Input, .Doshas, .Source; constitution.JSON200.Meta.Disclaimer

// Dinacharya. Brahma muhurta, the dosha periods and the routine for a date at the place looked up above.
var dinacharyaTz roxyapi.AyurvedaDinacharyaRequest_Timezone
_ = dinacharyaTz.FromAyurvedaDinacharyaRequestTimezone1(timezone)
dinacharya, err := roxy.Ayurveda.GetDinacharyaSchedule(ctx, nil, roxyapi.AyurvedaDinacharyaRequest{
	Date: roxyapi.Date(2026, time.October, 1), Latitude: latitude, Longitude: longitude, Timezone: &dinacharyaTz,
})
if err != nil {
	return err
}
// dinacharya.JSON200.BrahmaMuhurta, .DoshaPeriods, .Routine
```

### 14. I Ching API (cast a reading, hexagram catalog)

All 64 hexagrams, 384 changing lines and 8 trigrams, for meditation apps, decision tools and wisdom chatbots.

```go
// Cast a reading. Three coins six times: the primary hexagram, the changing lines and the resulting hexagram.
reading, err := roxy.Iching.CastReading(ctx, &roxyapi.CastReadingParams{Seed: roxyapi.Ptr("user-42")})
if err != nil {
	return err
}
// reading.JSON200.Hexagram.Number, .Hexagram.English, .Lines, .ChangingLinePositions, .ResultingHexagram

// Hexagram catalog. Paginated, 20 per page by default; ask for all 64 once and cache them.
hexagrams, err := roxy.Iching.ListHexagrams(ctx, &roxyapi.ListHexagramsParams{Limit: roxyapi.Ptr(64)})
if err != nil {
	return err
}
// hexagrams.JSON200.Total, hexagrams.JSON200.Hexagrams[n].Number, .English, .Pinyin; call roxy.Iching.GetHexagram(ctx, number, nil) for the judgment and lines
```

### 15. Crystal healing API (by zodiac, by chakra, birthstone)

Crystal retail and metaphysical content: "crystals for [sign]" and "[chakra] chakra stones" pages, plus the birthstone for each month.

```go
// By zodiac. The most searched crystal query pattern.
bySign, err := roxy.Crystals.GetCrystalsByZodiac(ctx, "scorpio", nil)
if err != nil {
	return err
}
// bySign.JSON200.Crystals[n].ID, .Name, .ImageURL, .Colors; call roxy.Crystals.GetCrystal(ctx, id, nil) for full properties

// By chakra. Wellness and yoga content pages.
byChakra, err := roxy.Crystals.GetCrystalsByChakra(ctx, "Heart", nil)
if err != nil {
	return err
}
// byChakra.JSON200.Crystals[n].Name, .Colors

// Birthstone. Evergreen gift and jewelry pages.
birthstone, err := roxy.Crystals.GetBirthstones(ctx, 1, nil)
if err != nil {
	return err
}
```

### 16. Dream interpretation API (symbol dictionary, search)

A 2,000+ symbol dream dictionary for journal apps, AI companions and self-discovery products.

```go
// Symbol detail. Every "what does it mean to dream about X" page lands here.
symbol, err := roxy.Dreams.GetDreamSymbol(ctx, "flying") // no params argument on this endpoint
if err != nil {
	return err
}
// symbol.JSON200.ID, .Name, .Meaning

// Symbol search. Chatbots fetch the dictionary once and keep it locally.
symbols, err := roxy.Dreams.SearchDreamSymbols(ctx, &roxyapi.SearchDreamSymbolsParams{Q: roxyapi.Ptr("water")})
if err != nil {
	return err
}
// symbols.JSON200.Symbols[n].ID, .Name
```

### 17. Angel numbers API (1111, 222, 333 meanings plus universal lookup)

Meanings for every common sequence, and a lookup that answers any positive integer through its digit root.

```go
// By number. Every "meaning of 1111" page is backed by this. The path param is a string.
angel, err := roxy.AngelNumbers.GetAngelNumber(ctx, "1111", nil)
if err != nil {
	return err
}
// angel.JSON200.Title, .CoreMessage, .Meaning.Spiritual, .Meaning.Love, .Affirmation

// Universal lookup. Any positive integer, with the digit root carrying the answer when no curated entry exists.
sequence, err := roxy.AngelNumbers.AnalyzeNumberSequence(ctx, &roxyapi.AnalyzeNumberSequenceParams{Number: "4242"})
if err != nil {
	return err
}
// sequence.JSON200.DigitRoot, .IsRepeating, .KnownMeaning (nil when not curated), .DigitRootMeaning.Title
```

## Built for AI agents

Every endpoint is also a remote MCP tool at `https://roxyapi.com/mcp/{domain}` (Streamable HTTP, no local setup). Use the SDK for typed Go services; use MCP for agents (Claude, Cursor, ChatGPT) that select tools from user intent. This module ships `AGENTS.md` and `docs/llms-full.txt` so coding agents read them straight from the module cache.

## Reliability

- Ephemeris output verified against NASA JPL Horizons. Methodology: https://roxyapi.com/methodology. Open benchmark (210 reference points vs JPL Horizons DE441): https://github.com/RoxyAPI/astrology-api-benchmark.
- Generated from the same OpenAPI spec that powers the live API, so the SDK never drifts from production.
- One catchable `RoxyError` with a stable `Code` for every failure.

## Gotchas

- **Argument arity varies.** Calls are `(ctx, pathParams..., params, body)`, but an endpoint with no query parameters has **no `params` argument**. Passing a stray `nil` there is read as a request editor and panics. The exact list is under Methods with no params argument below; let autocomplete show the signature.
- **The body type is `roxyapi.<MethodName>JSONRequestBody`.** Some are aliases of a named request (`NatalChartRequest`); both names compile.
- **`Date` and `Timezone` are typed.** Build a date with `roxyapi.Date(1990, time.January, 15)`, never a string. `Timezone` is a per-request union: `tz.From<Req>Timezone1("Europe/Berlin")` (IANA) or `From<Req>Timezone0(1)` (decimal offset).
- **Optional fields are pointers.** Use `roxyapi.Ptr(...)` for `Seed`, `Question`, `Lang`, `Limit`, and similar.
- **Enum-like strings are validated server-side.** `sign` and `Lang` are open string types; an invalid value compiles and comes back as a `validation_error` (400).
- **Read responses off `JSON200`** (`resp.JSON200.Cities[0].Latitude`). It is nil unless the call was a 2xx (errors are returned, not in the body).
- **Person-pair, forecast and Vastu bodies use anonymous nested structs** (`Person1`, `PersonA`, `BirthData`, `Plot`). Declare the body with `var` and fill those fields one by one, as the synastry, Guna Milan, forecast, connection and Vastu blocks above do; a slice of inline structs (`Rooms`) is easiest to unmarshal from its JSON shape.

### Methods with no params argument

Read from the generated facade on every release. Every other method takes `params` right after the path parameters.

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

## FAQ

**Q: `SearchCities` returned 200 but `Cities[0]` panics, or my city is not found.**
A: A successful search can still return an empty `Cities` slice, so check `len(search.JSON200.Cities) == 0` before indexing. Add the state or country whenever the name is common (`"Springfield, Illinois"`, `"London, United Kingdom"`); `Total` above 1 means the name is ambiguous, so show `Province` and `Country` and let the user confirm.

**Q: I got `nil pointer dereference` in `applyEditors`. What did I do?**
A: You passed `nil` to an endpoint that has no query-parameters argument. That is every method whose endpoint takes no query parameters, not even `lang`; the list is under Methods with no params argument above. The `nil` is read as a request editor: the call compiles, then panics at runtime. Drop the argument and let autocomplete show the signature.

**Q: `NewRoxy` returned no error but every call is `401 api_key_required`.**
A: Make sure `ROXY_API_KEY` is exported. `NewRoxy` returns an error for an empty key; a non-empty but wrong key only fails on the first request.

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

Configure the client with the generated options: `roxyapi.WithBaseURL`, `roxyapi.WithHTTPClient` (any `*http.Client`, for timeouts or proxies), and `roxyapi.WithRequestEditorFn`. The underlying `ClientWithResponses` stays exported for advanced use.

```go
roxy, err := roxyapi.NewRoxy(key, roxyapi.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}))
```

## Keywords

go sdk, golang api client, astrology api, vedic astrology api, kundli api, horoscope api, numerology api, tarot api, human design api, chinese astrology api, bazi api, feng shui api, mayan astrology api, vastu api, gematria api, ayurveda api, biorhythm api, i ching api, dream interpretation api, angel numbers api, geocoding api, rest api client, ai agent sdk, mcp server.

## License

MIT. See [LICENSE](https://github.com/RoxyAPI/sdk-go/blob/main/LICENSE).
