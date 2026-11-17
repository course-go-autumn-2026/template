package create

import (
	"io"

	api "github.com/freyliee/template/internal/generated"
	"github.com/freyliee/template/internal/http/jsonbody"
)

func decodeCreateTrip(body io.Reader) (api.CreateTripJSONRequestBody, error) {
	var data api.CreateTripJSONRequestBody

	coordinates := jsonbody.Shape{"latitude": nil, "longitude": nil}
	err := jsonbody.Decode(
		body,
		&data,
		jsonbody.Shape{
			"user_id":     nil,
			"driver_id":   nil,
			"price":       nil,
			"start_point": coordinates,
			"end_point":   coordinates,
		},
	)

	return data, err
}
