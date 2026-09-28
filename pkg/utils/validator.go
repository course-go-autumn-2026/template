package utils

import (
	"bytes"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/imrtlDrifter/template/internal/domainerrors"
	"github.com/imrtlDrifter/template/internal/models"
)

func ValidateCreateTripRequiredFields(body []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return err
	}

	requiredFields := []string{
		"user_id",
		"driver_id",
		"start_point",
		"end_point",
		"price",
	}

	for _, field := range requiredFields {
		if err := requireJSONField(fields, field, field); err != nil {
			return err
		}
	}

	for _, pointName := range []string{"start_point", "end_point"} {
		var pointFields map[string]json.RawMessage
		if err := json.Unmarshal(fields[pointName], &pointFields); err != nil {
			return err
		}

		for _, coordinate := range []string{"latitude", "longitude"} {
			fieldPath := pointName + "." + coordinate
			if err := requireJSONField(pointFields, coordinate, fieldPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func requireJSONField(fields map[string]json.RawMessage, name string, fieldPath string) error {
	value, exists := fields[name]
	if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return &domainerrors.ValidationError{
			Field:   fieldPath,
			Message: "is required",
		}
	}

	return nil
}

func ValidateCreateTrip(c *models.CreateTripCommand) error {
	switch {
	case c.UserID == uuid.Nil:
		return &domainerrors.ValidationError{
			Field:   "user_id",
			Message: "must be a non-empty UUID",
		}
	case c.DriverID == uuid.Nil:
		return &domainerrors.ValidationError{
			Field:   "driver_id",
			Message: "must be a non-empty UUID",
		}
	case c.StartPoint.Latitude < -90 || c.StartPoint.Latitude > 90:
		return &domainerrors.ValidationError{
			Field:   "start_point.latitude",
			Message: "must be between -90 and 90",
		}
	case c.StartPoint.Longitude < -180 || c.StartPoint.Longitude > 180:
		return &domainerrors.ValidationError{
			Field:   "start_point.longitude",
			Message: "must be between -180 and 180",
		}
	case c.EndPoint.Latitude < -90 || c.EndPoint.Latitude > 90:
		return &domainerrors.ValidationError{
			Field:   "end_point.latitude",
			Message: "must be between -90 and 90",
		}
	case c.EndPoint.Longitude < -180 || c.EndPoint.Longitude > 180:
		return &domainerrors.ValidationError{
			Field:   "end_point.longitude",
			Message: "must be between -180 and 180",
		}
	case c.Price < 0:
		return &domainerrors.ValidationError{
			Field:   "price",
			Message: "must be greater than or equal to 0",
		}
	default:
		return nil
	}
}
