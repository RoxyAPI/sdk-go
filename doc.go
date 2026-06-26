// Package roxyapi is the official Go SDK for RoxyAPI, the typed API for spiritual
// wellness insight: astrology, numerology, tarot, human design, forecasting and more,
// all under one key, verified against NASA JPL Horizons.
//
// # Install
//
//	go get github.com/RoxyAPI/sdk-go
//
// # Quickstart
//
//	package main
//
//	import (
//		"context"
//		"fmt"
//		"os"
//
//		roxyapi "github.com/RoxyAPI/sdk-go"
//	)
//
//	func main() {
//		roxy, err := roxyapi.NewRoxy(os.Getenv("ROXYAPI_KEY"))
//		if err != nil {
//			panic(err)
//		}
//		resp, err := roxy.Astrology.ListZodiacSigns(context.Background(), nil)
//		if err != nil {
//			panic(err) // a 4xx or 5xx arrives here as *roxyapi.RoxyError
//		}
//		fmt.Println(resp.JSON200)
//	}
//
// # Shape
//
// Methods are grouped by domain on the [Roxy] value returned by [NewRoxy]
// (roxy.Astrology, roxy.VedicAstrology, roxy.Numerology and so on). Each method
// returns the typed response whose JSON200 field holds the parsed body on success;
// any 4xx or 5xx is returned as a [RoxyError] you can switch on by Code. The
// underlying generated ClientWithResponses stays exported for advanced use.
//
// The full method reference lives in docs/llms-full.txt and at https://roxyapi.com/docs/sdk.
package roxyapi
