package pco

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"

	"github.com/micahlee/pco-cli/internal/models"
)

// ListPlans returns upcoming or past plans.
func (s *Service) ListPlans(ctx context.Context, filter string, count int) ([]models.Plan, error) {
	params := url.Values{
		"filter":   {filter},
		"per_page": {strconv.Itoa(count)},
		"order":    {"sort_date"},
	}
	if filter == "past" {
		params.Set("order", "-sort_date")
	}

	data, err := s.Client.Get(ctx, s.servicePath()+"/plans", params)
	if err != nil {
		return nil, err
	}

	resources, _, err := models.ParseList(data)
	if err != nil {
		return nil, err
	}

	plans := make([]models.Plan, len(resources))
	for i, r := range resources {
		var attrs models.PlanAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, err
		}
		plans[i] = models.Plan{ID: r.ID, Attrs: attrs}
	}
	return plans, nil
}

// GetPlan returns a single plan by ID.
func (s *Service) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	resource, err := s.getPlanResource(ctx, planID)
	if err != nil {
		return nil, err
	}

	var attrs models.PlanAttrs
	if err := json.Unmarshal(resource.Attributes, &attrs); err != nil {
		return nil, err
	}

	return &models.Plan{ID: resource.ID, Attrs: attrs}, nil
}

func (s *Service) getPlanResource(ctx context.Context, planID string) (*models.Resource, error) {
	data, err := s.Client.Get(ctx, s.servicePath()+"/plans/"+planID, nil)
	if err != nil {
		return nil, err
	}

	return models.ParseOne(data)
}

// ListPlanItems returns all items for a plan.
func (s *Service) ListPlanItems(ctx context.Context, planID string) ([]models.PlanItem, error) {
	return s.listPlanItems(ctx, planID, nil)
}

// ListPlanSongs returns only song items for a plan.
func (s *Service) ListPlanSongs(ctx context.Context, planID string) ([]models.PlanItem, error) {
	params := url.Values{"filter": {"songs"}}
	return s.listPlanItems(ctx, planID, params)
}

func (s *Service) listPlanItems(ctx context.Context, planID string, extra url.Values) ([]models.PlanItem, error) {
	params := url.Values{"per_page": {"50"}}
	for k, v := range extra {
		params[k] = v
	}

	data, err := s.Client.Get(ctx, s.servicePath()+"/plans/"+planID+"/items", params)
	if err != nil {
		return nil, err
	}

	resources, _, err := models.ParseList(data)
	if err != nil {
		return nil, err
	}

	items := make([]models.PlanItem, len(resources))
	for i, r := range resources {
		var attrs models.PlanItemAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, err
		}

		songID := ""
		if rel, ok := r.Relationships["song"]; ok {
			rid, _ := rel.One()
			if rid != nil {
				songID = rid.ID
			}
		}

		arrangementID := ""
		if rel, ok := r.Relationships["arrangement"]; ok {
			rid, _ := rel.One()
			if rid != nil {
				arrangementID = rid.ID
			}
		}

		keyID := ""
		if rel, ok := r.Relationships["key"]; ok {
			rid, _ := rel.One()
			if rid != nil {
				keyID = rid.ID
			}
		}

		items[i] = models.PlanItem{
			ID:            r.ID,
			SongID:        songID,
			ArrangementID: arrangementID,
			KeyID:         keyID,
			MediaIDs:      relationshipIDs(r, "media"),
			Attrs:         attrs,
		}
	}
	return items, nil
}

// ExportPlan returns a normalized plan document with enough item detail for comparisons.
func (s *Service) ExportPlan(ctx context.Context, planID string, includeRaw bool) (*models.PlanExport, error) {
	planResource, err := s.getPlanResource(ctx, planID)
	if err != nil {
		return nil, err
	}

	var planAttrs models.PlanAttrs
	if err := json.Unmarshal(planResource.Attributes, &planAttrs); err != nil {
		return nil, err
	}

	itemResources, included, err := s.listPlanItemResources(ctx, planID)
	if err != nil {
		return nil, err
	}

	includedByTypeID := indexResources(included)
	notesByItemID, err := itemNotesByItemID(included)
	if err != nil {
		return nil, err
	}

	sort.SliceStable(itemResources, func(i, j int) bool {
		left, right := itemSequence(itemResources[i]), itemSequence(itemResources[j])
		if left == right {
			return itemResources[i].ID < itemResources[j].ID
		}
		return left < right
	})

	export := &models.PlanExport{
		Plan: models.PlanExportPlan{
			ID:                planResource.ID,
			Title:             planAttrs.Title,
			Dates:             planAttrs.Dates,
			SortDate:          planAttrs.SortDate,
			PlanNotes:         planAttrs.PlanNotes,
			PlanningCenterURL: planAttrs.PlanningCenterURL,
		},
		Items: make([]models.PlanExportItem, 0, len(itemResources)),
	}

	currentHeader := ""
	for _, itemResource := range itemResources {
		item, err := exportItem(itemResource, includedByTypeID, notesByItemID, currentHeader)
		if err != nil {
			return nil, err
		}
		export.Items = append(export.Items, item)

		if item.Type == "header" {
			currentHeader = item.Title
		}
	}

	if includeRaw {
		export.Raw = &models.PlanExportRaw{
			Plan:     planResource,
			Items:    itemResources,
			Included: included,
		}
	}

	return export, nil
}

func (s *Service) listPlanItemResources(ctx context.Context, planID string) ([]models.Resource, []models.Resource, error) {
	params := url.Values{
		"include":  {"song,arrangement,item_notes,media"},
		"per_page": {"100"},
	}
	path := s.servicePath() + "/plans/" + planID + "/items"

	var items []models.Resource
	var included []models.Resource
	for path != "" {
		data, err := s.Client.Get(ctx, path, params)
		if err != nil {
			return nil, nil, err
		}
		params = nil

		pageItems, pageIncluded, links, err := models.ParseListDocument(data)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, pageItems...)
		included = append(included, pageIncluded...)

		path = ""
		if links != nil && links.Next != "" {
			path = links.Next
		}
	}

	return items, included, nil
}

func exportItem(
	resource models.Resource,
	included map[string]models.Resource,
	notesByItemID map[string][]models.PlanExportItemNote,
	headerContext string,
) (models.PlanExportItem, error) {
	var attrs models.PlanItemAttrs
	if err := json.Unmarshal(resource.Attributes, &attrs); err != nil {
		return models.PlanExportItem{}, err
	}

	title := attrs.Title
	if title == "" {
		title = attrs.Description
	}

	item := models.PlanExportItem{
		ID:                             resource.ID,
		Title:                          title,
		Type:                           attrs.ItemType,
		Sequence:                       attrs.Sequence,
		Length:                         attrs.Length,
		Description:                    attrs.Description,
		HTMLDetails:                    attrs.HTMLDetails,
		ServicePosition:                attrs.ServicePosition,
		HeaderContext:                  headerContext,
		KeyName:                        attrs.KeyName,
		CustomArrangementSequence:      attrs.CustomArrangementSequence,
		CustomArrangementSequenceFull:  attrs.CustomArrangementSequenceFull,
		CustomArrangementSequenceShort: attrs.CustomArrangementSequenceShort,
		Notes:                          notesByItemID[resource.ID],
		MediaIDs:                       relationshipIDs(resource, "media"),
	}
	if item.Type == "header" {
		item.HeaderContext = ""
	}

	if rid := relationshipID(resource, "song"); rid != "" {
		item.Song = &models.PlanExportSong{ID: rid}
		if song, ok := included["Song:"+rid]; ok {
			item.Song.Title = resourceAttrString(song, "title")
			item.Song.Author = resourceAttrString(song, "author")
		}
	}

	if rid := relationshipID(resource, "arrangement"); rid != "" {
		item.Arrangement = &models.PlanExportArrangement{ID: rid}
		if arrangement, ok := included["Arrangement:"+rid]; ok {
			item.Arrangement.Name = resourceAttrString(arrangement, "name")
			item.Arrangement.BPM = resourceAttrFloat(arrangement, "bpm")
			item.Arrangement.Meter = resourceAttrString(arrangement, "meter")
			item.Arrangement.Length = resourceAttrInt(arrangement, "length")
			item.Arrangement.ChordChartKey = resourceAttrString(arrangement, "chord_chart_key")
		}
	}

	return item, nil
}

func indexResources(resources []models.Resource) map[string]models.Resource {
	index := make(map[string]models.Resource, len(resources))
	for _, resource := range resources {
		index[resource.Type+":"+resource.ID] = resource
	}
	return index
}

func itemNotesByItemID(resources []models.Resource) (map[string][]models.PlanExportItemNote, error) {
	notes := map[string][]models.PlanExportItemNote{}
	for _, resource := range resources {
		if resource.Type != "ItemNote" {
			continue
		}

		itemID := relationshipID(resource, "item")
		if itemID == "" {
			continue
		}

		note := models.PlanExportItemNote{
			ID:           resource.ID,
			CategoryName: resourceAttrString(resource, "category_name"),
			Content:      resourceAttrString(resource, "content"),
		}
		notes[itemID] = append(notes[itemID], note)
	}
	return notes, nil
}

func itemSequence(resource models.Resource) int {
	var attrs struct {
		Sequence int `json:"sequence"`
	}
	_ = json.Unmarshal(resource.Attributes, &attrs)
	return attrs.Sequence
}

func relationshipID(resource models.Resource, name string) string {
	rel, ok := resource.Relationships[name]
	if !ok {
		return ""
	}
	rid, err := rel.One()
	if err != nil || rid == nil {
		return ""
	}
	return rid.ID
}

func relationshipIDs(resource models.Resource, name string) []string {
	rel, ok := resource.Relationships[name]
	if !ok {
		return nil
	}
	rids, err := rel.Many()
	if err != nil {
		return nil
	}
	ids := make([]string, 0, len(rids))
	for _, rid := range rids {
		ids = append(ids, rid.ID)
	}
	return ids
}

func resourceAttrString(resource models.Resource, name string) string {
	var attrs map[string]json.RawMessage
	if err := json.Unmarshal(resource.Attributes, &attrs); err != nil {
		return ""
	}
	value, ok := attrs[name]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(value, &s); err != nil {
		return ""
	}
	return s
}

func resourceAttrFloat(resource models.Resource, name string) *float64 {
	value := resourceAttr(resource, name)
	if len(value) == 0 || string(value) == "null" {
		return nil
	}
	var f float64
	if err := json.Unmarshal(value, &f); err != nil {
		return nil
	}
	return &f
}

func resourceAttrInt(resource models.Resource, name string) *int {
	value := resourceAttr(resource, name)
	if len(value) == 0 || string(value) == "null" {
		return nil
	}
	var i int
	if err := json.Unmarshal(value, &i); err != nil {
		return nil
	}
	return &i
}

func resourceAttr(resource models.Resource, name string) json.RawMessage {
	var attrs map[string]json.RawMessage
	if err := json.Unmarshal(resource.Attributes, &attrs); err != nil {
		return nil
	}
	return attrs[name]
}

// ListTemplates returns all plan templates.
func (s *Service) ListTemplates(ctx context.Context) ([]models.PlanTemplate, error) {
	data, err := s.Client.Get(ctx, s.servicePath()+"/plan_templates", nil)
	if err != nil {
		return nil, err
	}

	resources, _, err := models.ParseList(data)
	if err != nil {
		return nil, err
	}

	templates := make([]models.PlanTemplate, len(resources))
	for i, r := range resources {
		var attrs models.PlanTemplateAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, err
		}
		templates[i] = models.PlanTemplate{ID: r.ID, Attrs: attrs}
	}
	return templates, nil
}

// CreatePlan creates a new plan from a template on the given date.
func (s *Service) CreatePlan(ctx context.Context, date string, templateID string) ([]models.Plan, error) {
	if templateID == "" {
		templateID = s.Config.DefaultTemplateID
	}

	body := fmt.Sprintf(`{
		"data": {
			"type": "Plan",
			"attributes": {
				"base_date": %q,
				"copy_items": true,
				"copy_people": true,
				"copy_notes": true,
				"count": 1
			},
			"relationships": {
				"template": {
					"data": [{"type": "PlanTemplate", "id": %q}]
				}
			}
		}
	}`, date, templateID)

	data, err := s.Client.Post(ctx, s.servicePath()+"/create_plans", body)
	if err != nil {
		return nil, err
	}

	resources, _, err := models.ParseList(data)
	if err != nil {
		return nil, err
	}

	plans := make([]models.Plan, len(resources))
	for i, r := range resources {
		var attrs models.PlanAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, err
		}
		plans[i] = models.Plan{ID: r.ID, Attrs: attrs}
	}
	return plans, nil
}
