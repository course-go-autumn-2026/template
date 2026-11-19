package trip

import (
	"math"
	"time"
)

const maxFutureSkew = time.Minute

func ValidatePosition(lat, lon float64, recordedAt, now time.Time) error {
	if math.IsNaN(lat) || math.IsInf(lat, 0) || lat < -90 || lat > 90 {
		return NewValidationError("latitude", "must be between -90 and 90")
	}
	if math.IsNaN(lon) || math.IsInf(lon, 0) || lon < -180 || lon > 180 {
		return NewValidationError("longitude", "must be between -180 and 180")
	}
	if recordedAt.IsZero() {
		return NewValidationError("recorded_at", "is required")
	}
	if recordedAt.After(now.Add(maxFutureSkew)) {
		return NewValidationError("recorded_at", "cannot be more than 1 minute in the future")
	}
	return nil
}
