package steam

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type FlexibleString string

func (s *FlexibleString) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*s = FlexibleString(text)
		return nil
	}
	var number float64
	if err := json.Unmarshal(data, &number); err == nil {
		if number == float64(int64(number)) {
			*s = FlexibleString(strconv.FormatInt(int64(number), 10))
		} else {
			*s = FlexibleString(fmt.Sprintf("%g", number))
		}
		return nil
	}
	return fmt.Errorf("expected string or number")
}

func (s FlexibleString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

func (r *Requirements) UnmarshalJSON(data []byte) error {
	var req struct {
		Minimum     string `json:"minimum"`
		Recommended string `json:"recommended"`
	}
	if err := json.Unmarshal(data, &req); err == nil {
		*r = Requirements(req)
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		r.Minimum = text
		r.Recommended = ""
		return nil
	}
	var empty []any
	if err := json.Unmarshal(data, &empty); err == nil {
		*r = Requirements{}
		return nil
	}
	return fmt.Errorf("expected requirements object, string, or array")
}
