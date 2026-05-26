package models

import "encoding/json"

// Document is the top-level JSON:API response envelope.
type Document struct {
	Data     json.RawMessage `json:"data"`
	Included []Resource      `json:"included,omitempty"`
	Links    *Links          `json:"links,omitempty"`
	Meta     map[string]any  `json:"meta,omitempty"`
}

// Resource is a single JSON:API resource object.
type Resource struct {
	Type          string                  `json:"type"`
	ID            string                  `json:"id"`
	Attributes    json.RawMessage         `json:"attributes"`
	Relationships map[string]Relationship `json:"relationships,omitempty"`
	Links         *Links                  `json:"links,omitempty"`
}

// Relationship represents a JSON:API relationship.
type Relationship struct {
	Data json.RawMessage `json:"data,omitempty"`
}

// Links contains pagination and self links.
type Links struct {
	Self string `json:"self,omitempty"`
	Next string `json:"next,omitempty"`
	Prev string `json:"prev,omitempty"`
}

// RelationshipData extracts a single relationship resource identifier.
func (r Relationship) One() (*ResourceID, error) {
	if r.Data == nil || string(r.Data) == "null" {
		return nil, nil
	}
	var rid ResourceID
	if err := json.Unmarshal(r.Data, &rid); err != nil {
		return nil, err
	}
	return &rid, nil
}

// Many extracts a list of relationship resource identifiers.
func (r Relationship) Many() ([]ResourceID, error) {
	if r.Data == nil || string(r.Data) == "null" {
		return nil, nil
	}
	var ids []ResourceID
	if err := json.Unmarshal(r.Data, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// ResourceID is a type+id pair used in relationships.
type ResourceID struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// ParseOne parses a single-resource JSON:API response.
func ParseOne(data []byte) (*Resource, error) {
	var doc struct {
		Data Resource `json:"data"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc.Data, nil
}

// ParseList parses a list JSON:API response.
func ParseList(data []byte) ([]Resource, *Links, error) {
	resources, _, links, err := ParseListDocument(data)
	return resources, links, err
}

// ParseListDocument parses a list JSON:API response including side-loaded resources.
func ParseListDocument(data []byte) ([]Resource, []Resource, *Links, error) {
	var doc struct {
		Data     []Resource `json:"data"`
		Included []Resource `json:"included,omitempty"`
		Links    *Links     `json:"links,omitempty"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, nil, nil, err
	}
	return doc.Data, doc.Included, doc.Links, nil
}
