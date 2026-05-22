package steam

import (
	"encoding/json"
	"testing"
)

func TestFlexibleStringUnmarshalString(t *testing.T) {
	var got FlexibleString
	if err := json.Unmarshal([]byte(`"18"`), &got); err != nil {
		t.Fatalf("unmarshal string: %v", err)
	}
	if got != "18" {
		t.Fatalf("got %q, want 18", got)
	}
}

func TestFlexibleStringUnmarshalNumber(t *testing.T) {
	var got FlexibleString
	if err := json.Unmarshal([]byte(`18`), &got); err != nil {
		t.Fatalf("unmarshal number: %v", err)
	}
	if got != "18" {
		t.Fatalf("got %q, want 18", got)
	}
}

func TestRequirementsUnmarshalString(t *testing.T) {
	var got Requirements
	if err := json.Unmarshal([]byte(`"Requires a computer"`), &got); err != nil {
		t.Fatalf("unmarshal requirements string: %v", err)
	}
	if got.Minimum != "Requires a computer" || got.Recommended != "" {
		t.Fatalf("unexpected requirements: %+v", got)
	}
}

func TestRequirementsUnmarshalObject(t *testing.T) {
	var got Requirements
	if err := json.Unmarshal([]byte(`{"minimum":"min","recommended":"rec"}`), &got); err != nil {
		t.Fatalf("unmarshal requirements object: %v", err)
	}
	if got.Minimum != "min" || got.Recommended != "rec" {
		t.Fatalf("unexpected requirements: %+v", got)
	}
}
