package models

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPlanItemAttrsAllowsPolymorphicCustomArrangementSequences(t *testing.T) {
	data := []byte(`{
		"title": "Song",
		"custom_arrangement_sequence": ["Verse 1", "Chorus 1"],
		"custom_arrangement_sequence_full": {"Verse 1": "Verse 1", "Chorus 1": "Chorus 1"},
		"custom_arrangement_sequence_short": "V1, C1"
	}`)

	var attrs PlanItemAttrs
	if err := json.Unmarshal(data, &attrs); err != nil {
		t.Fatalf("PlanItemAttrs did not tolerate polymorphic custom sequence fields: %v", err)
	}

	if compactJSON(attrs.CustomArrangementSequence) != `["Verse 1","Chorus 1"]` {
		t.Fatalf("expected array raw message, got %s", attrs.CustomArrangementSequence)
	}
	if compactJSON(attrs.CustomArrangementSequenceFull) != `{"Verse 1":"Verse 1","Chorus 1":"Chorus 1"}` {
		t.Fatalf("expected object raw message, got %s", attrs.CustomArrangementSequenceFull)
	}
	if string(attrs.CustomArrangementSequenceShort) != `"V1, C1"` {
		t.Fatalf("expected string raw message, got %s", attrs.CustomArrangementSequenceShort)
	}
}

func TestPlanItemAttrsAllowsNullCustomArrangementSequences(t *testing.T) {
	data := []byte(`{
		"title": "Song",
		"custom_arrangement_sequence": null,
		"custom_arrangement_sequence_full": null,
		"custom_arrangement_sequence_short": null
	}`)

	var attrs PlanItemAttrs
	if err := json.Unmarshal(data, &attrs); err != nil {
		t.Fatalf("PlanItemAttrs did not tolerate null custom sequence fields: %v", err)
	}

	if string(attrs.CustomArrangementSequence) != `null` {
		t.Fatalf("expected null raw message, got %s", attrs.CustomArrangementSequence)
	}
	if string(attrs.CustomArrangementSequenceFull) != `null` {
		t.Fatalf("expected null raw message, got %s", attrs.CustomArrangementSequenceFull)
	}
	if string(attrs.CustomArrangementSequenceShort) != `null` {
		t.Fatalf("expected null raw message, got %s", attrs.CustomArrangementSequenceShort)
	}
}

func compactJSON(raw json.RawMessage) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return string(raw)
	}
	return buf.String()
}
