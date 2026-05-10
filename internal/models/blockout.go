package models

// BlockoutAttrs holds the attributes of a PCO blockout date.
type BlockoutAttrs struct {
	StartsAt        string `json:"starts_at"`
	EndsAt          string `json:"ends_at"`
	Reason          string `json:"reason"`
	RepeatFrequency string `json:"repeat_frequency"`
}

// Blockout is a fully resolved blockout with ID.
type Blockout struct {
	ID    string
	Attrs BlockoutAttrs
}
