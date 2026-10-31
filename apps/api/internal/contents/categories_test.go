package contents

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestCategoryNameValidation(t *testing.T) {
	in := CategoryInput{Name: "  บทความ  "}
	if err := in.Validate(); err != nil || in.Name != "บทความ" {
		t.Fatalf("trimmed name: %+v %v", in, err)
	}
	for _, name := range []string{"", "   ", strings.Repeat("ก", 81), "บรรทัด\nสอง"} {
		in = CategoryInput{Name: name}
		if !errors.Is(in.Validate(), ErrInvalid) {
			t.Fatalf("name %q must be rejected", name)
		}
	}
	in = CategoryInput{Name: strings.Repeat("ก", 80)}
	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestContentCategoryIDJSON(t *testing.T) {
	var omitted Input
	if err := json.Unmarshal([]byte(`{"title":"a","body":"b"}`), &omitted); err != nil || omitted.Category.Set {
		t.Fatalf("omitted categoryId must leave the category alone: %+v %v", omitted.Category, err)
	}
	if err := omitted.Validate(); err != nil {
		t.Fatal(err)
	}

	var cleared Input
	if err := json.Unmarshal([]byte(`{"title":"a","body":"b","categoryId":null}`), &cleared); err != nil {
		t.Fatal(err)
	}
	id, set := cleared.Category.Specified()
	if !set || id != "" || !cleared.Category.Null {
		t.Fatalf("null must clear the category, got %+v", cleared.Category)
	}
	if err := cleared.Validate(); err != nil {
		t.Fatal(err)
	}

	var assigned Input
	raw := `{"title":"a","body":"b","categoryId":"33333333-3333-3333-3333-333333333333"}`
	if err := json.Unmarshal([]byte(raw), &assigned); err != nil {
		t.Fatal(err)
	}
	id, set = assigned.Category.Specified()
	if !set || id != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("uuid category: %+v", assigned.Category)
	}
	if err := assigned.Validate(); err != nil {
		t.Fatal(err)
	}

	var bad Input
	if err := json.Unmarshal([]byte(`{"title":"a","body":"b","categoryId":"nope"}`), &bad); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(bad.Validate(), ErrInvalid) {
		t.Fatal("a non-uuid categoryId must be invalid")
	}
	if err := json.Unmarshal([]byte(`{"title":"a","body":"b","categoryId":""}`), &bad); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(bad.Validate(), ErrInvalid) {
		t.Fatal("an empty categoryId must be invalid; null clears it")
	}
	if json.Unmarshal([]byte(`{"title":"a","body":"b","categoryId":1}`), &bad) == nil {
		t.Fatal("a numeric categoryId must not parse")
	}
}
