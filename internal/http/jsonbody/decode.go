package jsonbody

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type Shape map[string]Shape

func Decode(body io.Reader, target any, shape Shape) error {
	decoder := json.NewDecoder(body)

	var raw json.RawMessage

	if err := decoder.Decode(&raw); err != nil {
		return fmt.Errorf("request body must contain valid JSON")
	}

	var extra json.RawMessage

	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("request body must contain exactly one JSON value")
	}

	if err := validate(raw, shape); err != nil {
		return err
	}

	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("request body contains invalid values")
	}

	return nil
}

func validate(raw json.RawMessage, shape Shape) error {
	var fields map[string]json.RawMessage

	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return fmt.Errorf("expected an object")
	}

	for name := range fields {
		if _, ok := shape[name]; !ok {
			return fmt.Errorf("unknown field %s", name)
		}
	}

	for name, nested := range shape {
		value, ok := fields[name]

		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("%s is required and must not be null", name)
		}

		if nested != nil {
			if err := validate(value, nested); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	}

	return nil
}
