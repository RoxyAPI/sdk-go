package roxyapi_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	roxyapi "github.com/RoxyAPI/sdk-go"
)

// The simplest call: a daily horoscope by sign. Pass nil for the optional params.
func ExampleNewRoxy() {
	roxy, err := roxyapi.NewRoxy(os.Getenv("ROXYAPI_KEY"))
	if err != nil {
		panic(err)
	}
	resp, err := roxy.Astrology.GetDailyHoroscope(context.Background(), "aries", nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.JSON200)
}

// Geocode the birth city, then feed its coordinates and IANA timezone into a chart.
func Example_geocodeThenChart() {
	roxy, _ := roxyapi.NewRoxy(os.Getenv("ROXYAPI_KEY"))
	ctx := context.Background()

	search, err := roxy.Location.SearchCities(ctx, &roxyapi.SearchCitiesParams{Q: "London, UK"})
	if err != nil {
		panic(err)
	}
	city := search.JSON200.Cities[0]

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

// A 4xx or 5xx is returned as *RoxyError. Switch on the stable Code.
func Example_errorHandling() {
	roxy, _ := roxyapi.NewRoxy("sk_live_invalid")
	_, err := roxy.Astrology.GetDailyHoroscope(context.Background(), "aries", nil)

	var rerr *roxyapi.RoxyError
	if errors.As(err, &rerr) {
		fmt.Printf("%d %s\n", rerr.StatusCode, rerr.Code)
		for _, iss := range rerr.Issues { // populated on a 400
			fmt.Printf("%s: %s\n", iss.Path, iss.Message)
		}
	}
}

// Configure a timeout (or proxy) with the generated WithHTTPClient option.
func ExampleNewRoxy_customClient() {
	roxy, err := roxyapi.NewRoxy(
		os.Getenv("ROXYAPI_KEY"),
		roxyapi.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
	)
	if err != nil {
		panic(err)
	}
	_ = roxy
}
