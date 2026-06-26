package roxyapi

import (
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Date builds the date value the request bodies expect (the Date, BirthDate,
// StartDate and similar fields), so you do not have to construct it by hand or
// import the runtime types package.
//
//	body := roxyapi.NatalChartRequest{Date: roxyapi.Date(1990, time.January, 15), ...}
func Date(year int, month time.Month, day int) openapi_types.Date {
	return openapi_types.Date{Time: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// Ptr returns a pointer to v. Optional request-body fields and query parameters
// (Seed, Question, Lang, Limit and so on) are pointers; Ptr sets them inline.
//
//	body := roxyapi.GetDailyCardJSONRequestBody{Seed: roxyapi.Ptr("user-42")}
//	lang := roxyapi.GetDailyHoroscopeParams{Lang: roxyapi.Ptr(roxyapi.GetDailyHoroscopeParamsLang("es"))}
func Ptr[T any](v T) *T { return &v }
