package transport

import (
	"encoding/json"
	"testing"
)

type sampleEvent struct {
	Foo string `json:"foo"`
}

func TestGetNameOfStruct(t *testing.T) {
	// Pointer to struct -> the element's type name.
	if got := getNameOfStruct(&sampleEvent{}); got != "sampleEvent" {
		t.Errorf("getNameOfStruct(ptr) = %q, want sampleEvent", got)
	}
	// Value struct -> its type name.
	if got := getNameOfStruct(sampleEvent{}); got != "sampleEvent" {
		t.Errorf("getNameOfStruct(value) = %q, want sampleEvent", got)
	}
}

func TestEventToJSON(t *testing.T) {
	data, err := eventToJSON(&sampleEvent{Foo: "bar"})
	if err != nil {
		t.Fatalf("eventToJSON error: %v", err)
	}

	var decoded struct {
		Name string `json:"name"`
		Data struct {
			Foo string `json:"foo"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Name != "sampleEvent" {
		t.Errorf("name = %q, want sampleEvent", decoded.Name)
	}
	if decoded.Data.Foo != "bar" {
		t.Errorf("data.foo = %q, want bar", decoded.Data.Foo)
	}
}
