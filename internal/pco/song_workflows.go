package pco

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/micahlee/pco-cli/internal/models"
)

const songWorkflowSchemaVersion = "1"

// OperationError is an agent-friendly failure that is also useful as a normal Go error.
type OperationError struct {
	Result OperationErrorResult
}

func (e *OperationError) Error() string {
	message := e.Result.Error.Message
	if len(e.Result.CreatedResources) > 0 {
		parts := make([]string, len(e.Result.CreatedResources))
		for i, resource := range e.Result.CreatedResources {
			parts[i] = fmt.Sprintf("%s %s (%s, verified=%t)", resource.Type, resource.Name, resource.ID, resource.Verified)
		}
		message += "; created resources: " + strings.Join(parts, ", ")
	}
	if e.Result.ResumeCommand != "" {
		message += "; resume with: " + e.Result.ResumeCommand
	}
	return message
}

// StructuredResult lets the command layer emit this error as JSON when --json is set.
func (e *OperationError) StructuredResult() any { return e.Result }

// OperationErrorResult is the stable failure envelope for intent-level operations.
type OperationErrorResult struct {
	SchemaVersion    string            `json:"schema_version"`
	Operation        string            `json:"operation"`
	Status           string            `json:"status"`
	Error            OperationProblem  `json:"error"`
	CreatedResources []CreatedResource `json:"created_resources"`
	ResumeCommand    string            `json:"resume_command"`
}

type OperationProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CreatedResource struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Verified bool   `json:"verified"`
}

func operationError(operation, code, message string) error {
	return &OperationError{Result: OperationErrorResult{
		SchemaVersion:    songWorkflowSchemaVersion,
		Operation:        operation,
		Status:           "error",
		Error:            OperationProblem{Code: code, Message: message},
		CreatedResources: []CreatedResource{},
	}}
}

type SongIdentity struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Author     string `json:"author,omitempty"`
	CCLINumber *int   `json:"ccli_number,omitempty"`
}

type PlanIdentity struct {
	ID   string `json:"id"`
	Date string `json:"date"`
}

type PlanItemIdentity struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	SongID        string `json:"song_id,omitempty"`
	ArrangementID string `json:"arrangement_id,omitempty"`
}

type ArrangementIdentity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ReplaceSongOptions identifies a guarded, intent-level plan song replacement.
type ReplaceSongOptions struct {
	Date          string
	PlanID        string
	CurrentTitle  string
	Replacement   string
	ArrangementID string
	SongOnly      bool
	DryRun        bool
}

// SongReplaceResult is the stable preview/no-op/success result shape.
type SongReplaceResult struct {
	SchemaVersion string               `json:"schema_version"`
	Operation     string               `json:"operation"`
	Status        string               `json:"status"`
	Mutated       bool                 `json:"mutated"`
	Verified      bool                 `json:"verified"`
	Plan          PlanIdentity         `json:"plan"`
	Item          PlanItemIdentity     `json:"item"`
	Current       SongIdentity         `json:"current"`
	Replacement   SongIdentity         `json:"replacement"`
	Arrangement   *ArrangementIdentity `json:"arrangement"`
	Warnings      []string             `json:"warnings"`
}

// ReplaceSong resolves human intent, checks stale state immediately before mutation, and verifies afterward.
func (s *Service) ReplaceSong(ctx context.Context, opts ReplaceSongOptions) (*SongReplaceResult, error) {
	const operation = "songs.replace"
	if strings.TrimSpace(opts.CurrentTitle) == "" {
		return nil, operationError(operation, "missing_current", "--current is required")
	}
	if strings.TrimSpace(opts.Replacement) == "" {
		return nil, operationError(operation, "missing_replacement", "--with is required")
	}
	if opts.ArrangementID != "" && opts.SongOnly {
		return nil, operationError(operation, "invalid_options", "--arrangement-id and --song-only cannot be used together")
	}

	plan, planDate, err := s.resolveReplacementPlan(ctx, opts.PlanID, opts.Date)
	if err != nil {
		return nil, err
	}

	replacement, err := s.resolveExactSong(ctx, opts.Replacement, "replacement")
	if err != nil {
		return nil, err
	}

	items, err := s.ListPlanSongs(ctx, plan.ID)
	if err != nil {
		return nil, operationError(operation, "plan_items_read_failed", fmt.Sprintf("reading song items for plan %s: %v", plan.ID, err))
	}
	currentMatches := matchingPlanItems(items, opts.CurrentTitle)
	replacementItems := matchingReplacementItems(items, *replacement)

	if len(currentMatches) > 1 {
		return nil, operationError(operation, "current_song_ambiguous", fmt.Sprintf("found %d song items titled %q in plan %s; use the low-level songs set command with explicit IDs", len(currentMatches), opts.CurrentTitle, plan.ID))
	}
	if len(replacementItems) > 1 {
		return nil, operationError(operation, "replacement_item_ambiguous", fmt.Sprintf("found %d plan items already assigned to %q in plan %s; inspect the plan and use the low-level songs set command with explicit IDs", len(replacementItems), replacement.Attrs.Title, plan.ID))
	}
	if len(currentMatches) == 0 && len(replacementItems) == 1 {
		item := replacementItems[0]
		return &SongReplaceResult{
			SchemaVersion: songWorkflowSchemaVersion, Operation: operation, Status: "noop",
			Mutated: false, Verified: true, Plan: PlanIdentity{ID: plan.ID, Date: planDate},
			Item: planItemIdentity(item), Current: songIdentity(*replacement), Replacement: songIdentity(*replacement),
			Arrangement: arrangementIdentityFromItem(item), Warnings: []string{},
		}, nil
	}
	if len(currentMatches) == 0 {
		return nil, operationError(operation, "current_song_not_found", fmt.Sprintf("no song item titled %q exists in plan %s", opts.CurrentTitle, plan.ID))
	}
	if len(replacementItems) == 1 && currentMatches[0].ID != replacementItems[0].ID {
		return nil, operationError(operation, "replacement_already_elsewhere", fmt.Sprintf("%q is already present elsewhere in plan %s; refusing to create a duplicate plan song", replacement.Attrs.Title, plan.ID))
	}

	currentItem := currentMatches[0]
	if currentItem.SongID == replacement.ID {
		return &SongReplaceResult{
			SchemaVersion: songWorkflowSchemaVersion, Operation: operation, Status: "noop",
			Mutated: false, Verified: true, Plan: PlanIdentity{ID: plan.ID, Date: planDate},
			Item: planItemIdentity(currentItem), Current: songIdentity(*replacement), Replacement: songIdentity(*replacement),
			Arrangement: arrangementIdentityFromItem(currentItem), Warnings: []string{},
		}, nil
	}

	currentSong := SongIdentity{ID: currentItem.SongID, Title: currentItem.Attrs.Title}
	if currentItem.SongID != "" {
		if resolved, getErr := s.GetSong(ctx, currentItem.SongID); getErr == nil {
			currentSong = songIdentity(*resolved)
		}
	}

	arrangement, err := s.resolveSongArrangement(ctx, replacement.ID, SongAssignmentOptions{ArrangementID: opts.ArrangementID, SongOnly: opts.SongOnly})
	if err != nil {
		return nil, operationError(operation, "arrangement_resolution_failed", err.Error())
	}
	result := &SongReplaceResult{
		SchemaVersion: songWorkflowSchemaVersion, Operation: operation, Status: "preview",
		Mutated: false, Verified: false, Plan: PlanIdentity{ID: plan.ID, Date: planDate},
		Item: planItemIdentity(currentItem), Current: currentSong, Replacement: songIdentity(*replacement),
		Arrangement: arrangementIdentity(arrangement), Warnings: []string{},
	}
	if opts.SongOnly {
		result.Warnings = append(result.Warnings, "replacement will not have an arrangement attached (--song-only)")
	}
	if opts.DryRun {
		return result, nil
	}

	// Planning Center does not expose an atomic conditional update for this resource.
	// Re-read immediately before PATCH and treat the current relationship as a CAS precondition.
	fresh, err := s.GetPlanItem(ctx, plan.ID, currentItem.ID)
	if err != nil {
		return nil, operationError(operation, "precondition_read_failed", fmt.Sprintf("could not re-read item %s before mutation: %v", currentItem.ID, err))
	}
	if fresh.SongID != currentItem.SongID || normalizeTitle(fresh.Attrs.Title) != normalizeTitle(opts.CurrentTitle) {
		return nil, operationError(operation, "stale_state", fmt.Sprintf("plan item %s changed after preflight (expected %q, found %q); no mutation was made", currentItem.ID, opts.CurrentTitle, fresh.Attrs.Title))
	}

	if err := s.patchSongItem(ctx, plan.ID, currentItem.ID, *replacement, arrangement, opts.SongOnly); err != nil {
		return nil, operationError(operation, "replacement_failed", fmt.Sprintf("updating plan item %s: %v", currentItem.ID, err))
	}
	verified, err := s.GetPlanItem(ctx, plan.ID, currentItem.ID)
	if err != nil {
		return nil, operationError(operation, "verification_failed", fmt.Sprintf("replacement request succeeded but item %s could not be re-read: %v; inspect with `pco plans items %s`", currentItem.ID, err, plan.ID))
	}
	wantArrangement := ""
	if arrangement != nil {
		wantArrangement = arrangement.ID
	}
	if verified.SongID != replacement.ID || verified.ArrangementID != wantArrangement {
		return nil, operationError(operation, "verification_failed", fmt.Sprintf("replacement request succeeded but item %s has song %q and arrangement %q; expected song %q and arrangement %q", currentItem.ID, verified.SongID, verified.ArrangementID, replacement.ID, wantArrangement))
	}

	result.Status = "success"
	result.Mutated = true
	result.Verified = true
	result.Item = planItemIdentity(*verified)
	return result, nil
}

func (s *Service) resolveReplacementPlan(ctx context.Context, planID, dateText string) (*models.Plan, string, error) {
	const operation = "songs.replace"
	var requestedDate time.Time
	var err error
	if dateText != "" {
		requestedDate, err = parseDate(dateText)
		if err != nil {
			return nil, "", operationError(operation, "invalid_date", err.Error())
		}
	}
	if planID == "" && dateText == "" {
		return nil, "", operationError(operation, "missing_plan", "provide either --date YYYY-MM-DD or --plan-id")
	}
	if planID != "" {
		plan, getErr := s.GetPlan(ctx, planID)
		if getErr != nil {
			return nil, "", operationError(operation, "plan_not_found", fmt.Sprintf("could not resolve plan %s in configured service type: %v", planID, getErr))
		}
		actualDate, dateErr := datePart(plan.Attrs.SortDate)
		if dateErr != nil {
			return nil, "", operationError(operation, "invalid_plan_date", fmt.Sprintf("plan %s has malformed or missing sort_date: %v", planID, dateErr))
		}
		if dateText != "" && actualDate != requestedDate.Format("2006-01-02") {
			return nil, "", operationError(operation, "plan_date_mismatch", fmt.Sprintf("plan %s is dated %s, not %s", planID, actualDate, dateText))
		}
		return plan, actualDate, nil
	}

	// Use the documented after filter from the prior day, then filter locally to
	// the exact date. This avoids relying on an undocumented multi-filter syntax.
	searchAfter := requestedDate.AddDate(0, 0, -1).Format("2006-01-02")
	var matches []models.Plan
	for resource, getErr := range s.Client.PageIterator(ctx, s.servicePath()+"/plans", url.Values{
		"filter": {"after"}, "after": {searchAfter}, "per_page": {"100"}, "order": {"sort_date"},
	}) {
		if getErr != nil {
			return nil, "", operationError(operation, "plan_search_failed", fmt.Sprintf("searching plans for %s: %v", dateText, getErr))
		}
		var attrs models.PlanAttrs
		if err := json.Unmarshal(resource.Attributes, &attrs); err != nil {
			return nil, "", operationError(operation, "invalid_plan", fmt.Sprintf("decoding plan %s: %v", resource.ID, err))
		}
		actual, dateErr := datePart(attrs.SortDate)
		if dateErr != nil {
			return nil, "", operationError(operation, "invalid_plan_date", fmt.Sprintf("plan %s has malformed or missing sort_date: %v", resource.ID, dateErr))
		}
		if actual > dateText {
			break
		}
		if actual == dateText {
			matches = append(matches, models.Plan{ID: resource.ID, Attrs: attrs})
		}
	}
	if len(matches) == 0 {
		return nil, "", operationError(operation, "plan_not_found", fmt.Sprintf("no plan in configured service type was found for %s", dateText))
	}
	if len(matches) > 1 {
		ids := make([]string, len(matches))
		for i := range matches {
			ids[i] = matches[i].ID
		}
		return nil, "", operationError(operation, "plan_ambiguous", fmt.Sprintf("found %d plans for %s (%s); rerun with --plan-id", len(matches), dateText, strings.Join(ids, ", ")))
	}
	return &matches[0], dateText, nil
}

func (s *Service) resolveExactSong(ctx context.Context, title, role string) (*models.Song, error) {
	const operation = "songs.replace"
	songs, err := s.SearchSongs(ctx, strings.Join(strings.Fields(title), " "))
	if err != nil {
		return nil, operationError(operation, role+"_song_search_failed", fmt.Sprintf("searching for %s song %q: %v", role, title, err))
	}
	var exact []models.Song
	for _, song := range songs {
		if normalizeTitle(song.Attrs.Title) == normalizeTitle(title) {
			exact = append(exact, song)
		}
	}
	if len(exact) == 0 {
		return nil, operationError(operation, role+"_song_not_found", fmt.Sprintf("no library song exactly matching %q was found", title))
	}
	if len(exact) > 1 {
		ids := make([]string, len(exact))
		for i := range exact {
			ids[i] = exact[i].ID
		}
		return nil, operationError(operation, role+"_song_ambiguous", fmt.Sprintf("found %d library songs exactly matching %q (%s); use the low-level ID-based command", len(exact), title, strings.Join(ids, ", ")))
	}
	return &exact[0], nil
}

func matchingPlanItems(items []models.PlanItem, title string) []models.PlanItem {
	var matches []models.PlanItem
	for _, item := range items {
		if normalizeTitle(item.Attrs.Title) == normalizeTitle(title) {
			matches = append(matches, item)
		}
	}
	return matches
}

func matchingReplacementItems(items []models.PlanItem, song models.Song) []models.PlanItem {
	var matches []models.PlanItem
	for _, item := range items {
		if item.SongID == song.ID || normalizeTitle(item.Attrs.Title) == normalizeTitle(song.Attrs.Title) {
			matches = append(matches, item)
		}
	}
	return matches
}

func normalizeTitle(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func parseDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return time.Time{}, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", value)
	}
	return parsed, nil
}

func datePart(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("date is empty")
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		parsed, err = time.Parse("2006-01-02", value)
		if err != nil {
			return "", fmt.Errorf("invalid date %q", value)
		}
	}
	return parsed.Format("2006-01-02"), nil
}

func planItemIdentity(item models.PlanItem) PlanItemIdentity {
	return PlanItemIdentity{ID: item.ID, Title: item.Attrs.Title, SongID: item.SongID, ArrangementID: item.ArrangementID}
}
func songIdentity(song models.Song) SongIdentity {
	return SongIdentity{ID: song.ID, Title: song.Attrs.Title, Author: song.Attrs.Author, CCLINumber: song.Attrs.CCLINumber}
}
func arrangementIdentity(arrangement *models.Arrangement) *ArrangementIdentity {
	if arrangement == nil {
		return nil
	}
	return &ArrangementIdentity{ID: arrangement.ID, Name: arrangement.Attrs.Name}
}
func arrangementIdentityFromItem(item models.PlanItem) *ArrangementIdentity {
	if item.ArrangementID == "" {
		return nil
	}
	return &ArrangementIdentity{ID: item.ArrangementID}
}

// CreateSongOptions describes explicit library-song and initial-arrangement creation.
type CreateSongOptions struct {
	Title           string
	Authors         string
	ArrangementName string
	CCLINumber      *int
	NoCCLI          bool
	Key             string
	BPM             *float64
	Meter           string
	AllowDuplicate  bool
	ResumeSongID    string
}

type SongCreateResult struct {
	SchemaVersion    string              `json:"schema_version"`
	Operation        string              `json:"operation"`
	Status           string              `json:"status"`
	Mutated          bool                `json:"mutated"`
	Verified         bool                `json:"verified"`
	Song             SongIdentity        `json:"song"`
	Arrangement      ArrangementIdentity `json:"arrangement"`
	Warnings         []string            `json:"warnings"`
	CreatedResources []CreatedResource   `json:"created_resources"`
	Resumed          bool                `json:"resumed"`
}

// CreateSong explicitly creates a library song and its initial arrangement, never a plan item.
func (s *Service) CreateSong(ctx context.Context, opts CreateSongOptions) (*SongCreateResult, error) {
	const operation = "songs.create"
	if err := validateCreateSongOptions(opts); err != nil {
		return nil, operationError(operation, "invalid_options", err.Error())
	}
	warnings := missingMetadataWarnings(opts)

	if opts.ResumeSongID == "" {
		matches, err := s.findDuplicateSongs(ctx, opts)
		if err != nil {
			return nil, operationError(operation, "duplicate_search_failed", err.Error())
		}
		if len(matches) > 0 && !opts.AllowDuplicate {
			parts := make([]string, len(matches))
			for i, match := range matches {
				ccli := "no CCLI"
				if match.Attrs.CCLINumber != nil {
					ccli = "CCLI " + strconv.Itoa(*match.Attrs.CCLINumber)
				}
				parts[i] = fmt.Sprintf("%s %q (%s)", match.ID, match.Attrs.Title, ccli)
			}
			return nil, operationError(operation, "duplicate_detected", fmt.Sprintf("plausible existing song matches found: %s; inspect them or rerun with --allow-duplicate", strings.Join(parts, ", ")))
		}
	}

	created := []CreatedResource{}
	var song *models.Song
	resumed := opts.ResumeSongID != ""
	if resumed {
		var err error
		song, err = s.GetSong(ctx, opts.ResumeSongID)
		if err != nil {
			return nil, operationError(operation, "resume_song_not_found", fmt.Sprintf("could not read resume song %s: %v", opts.ResumeSongID, err))
		}
		if !songMatches(*song, opts) {
			return nil, operationError(operation, "resume_song_mismatch", fmt.Sprintf("resume song %s is %q by %q, which does not match requested %q by %q", song.ID, song.Attrs.Title, song.Attrs.Author, opts.Title, opts.Authors))
		}
	} else {
		var err error
		song, err = s.createSongResource(ctx, opts)
		if err != nil {
			return nil, operationError(operation, "song_create_failed", fmt.Sprintf("creating song: %v", err))
		}
		verifiedSong, verifyErr := s.GetSong(ctx, song.ID)
		if verifyErr != nil || !songMatches(*verifiedSong, opts) {
			message := fmt.Sprintf("song %s was created but could not be verified", song.ID)
			if verifyErr != nil {
				message += ": " + verifyErr.Error()
			}
			return nil, partialCreateError(opts, "verification_failed", message, []CreatedResource{{Type: "song", ID: song.ID, Name: song.Attrs.Title, Verified: false}})
		}
		song = verifiedSong
		created = append(created, CreatedResource{Type: "song", ID: song.ID, Name: song.Attrs.Title, Verified: true})
	}

	arrangements, err := s.ListSongArrangements(ctx, song.ID)
	if err != nil {
		return nil, partialCreateError(opts, "arrangement_preflight_failed", fmt.Sprintf("song %s exists but arrangements could not be read: %v", song.ID, err), created)
	}
	var sameName []models.Arrangement
	for _, arrangement := range arrangements {
		if !arrangement.Archived && normalizeTitle(arrangement.Attrs.Name) == normalizeTitle(opts.ArrangementName) {
			sameName = append(sameName, arrangement)
		}
	}
	if len(sameName) > 1 {
		return nil, partialCreateError(opts, "arrangement_ambiguous", fmt.Sprintf("song %s has %d active arrangements named %q; no mutation was made", song.ID, len(sameName), opts.ArrangementName), created)
	}
	if len(sameName) == 1 {
		if !resumed {
			return nil, partialCreateError(opts, "arrangement_already_exists", fmt.Sprintf("new song %s unexpectedly already has arrangement %q; inspect before continuing", song.ID, opts.ArrangementName), created)
		}
		if !arrangementMatches(sameName[0], opts) {
			return nil, operationError(operation, "resume_arrangement_mismatch", fmt.Sprintf("arrangement %s already exists but its metadata does not match the requested resume values", sameName[0].ID))
		}
		return &SongCreateResult{SchemaVersion: songWorkflowSchemaVersion, Operation: operation, Status: "noop", Mutated: false, Verified: true,
			Song: songIdentity(*song), Arrangement: *arrangementIdentity(&sameName[0]), Warnings: warnings, CreatedResources: []CreatedResource{}, Resumed: true}, nil
	}

	arrangement, err := s.createArrangementResource(ctx, song.ID, opts)
	if err != nil {
		return nil, partialCreateError(opts, "arrangement_create_failed", fmt.Sprintf("song %s exists, but creating its arrangement failed: %v", song.ID, err), created)
	}
	created = append(created, CreatedResource{Type: "arrangement", ID: arrangement.ID, Name: arrangement.Attrs.Name, Verified: false})
	verifiedArrangement, err := s.GetSongArrangement(ctx, song.ID, arrangement.ID)
	if err != nil || !arrangementMatches(*verifiedArrangement, opts) {
		message := fmt.Sprintf("song %s and arrangement %s were created, but the arrangement could not be verified", song.ID, arrangement.ID)
		if err != nil {
			message += ": " + err.Error()
		}
		return nil, partialCreateError(opts, "verification_failed", message, created)
	}
	created[len(created)-1].Verified = true
	return &SongCreateResult{SchemaVersion: songWorkflowSchemaVersion, Operation: operation, Status: "success", Mutated: true, Verified: true,
		Song: songIdentity(*song), Arrangement: *arrangementIdentity(verifiedArrangement), Warnings: warnings, CreatedResources: created, Resumed: resumed}, nil
}

func validateCreateSongOptions(opts CreateSongOptions) error {
	if strings.TrimSpace(opts.Title) == "" {
		return fmt.Errorf("--title is required")
	}
	if strings.TrimSpace(opts.Authors) == "" {
		return fmt.Errorf("--authors is required")
	}
	if strings.TrimSpace(opts.ArrangementName) == "" {
		return fmt.Errorf("--arrangement-name is required")
	}
	if (opts.CCLINumber == nil) == !opts.NoCCLI {
		return fmt.Errorf("provide exactly one of --ccli or --no-ccli")
	}
	if opts.CCLINumber != nil && *opts.CCLINumber <= 0 {
		return fmt.Errorf("--ccli must be a positive number")
	}
	if opts.BPM != nil && *opts.BPM <= 0 {
		return fmt.Errorf("--bpm must be greater than zero")
	}
	if opts.Meter != "" {
		validMeters := map[string]bool{"2/2": true, "2/4": true, "3/2": true, "3/4": true, "4/2": true, "4/4": true, "5/4": true, "6/4": true, "3/8": true, "6/8": true, "7/4": true, "7/8": true, "9/8": true, "12/4": true, "12/8": true}
		if !validMeters[opts.Meter] {
			return fmt.Errorf("unsupported --meter %q", opts.Meter)
		}
	}
	return nil
}

func missingMetadataWarnings(opts CreateSongOptions) []string {
	warnings := []string{}
	if strings.TrimSpace(opts.Key) == "" {
		warnings = append(warnings, "key was not supplied")
	}
	if opts.BPM == nil {
		warnings = append(warnings, "BPM was not supplied")
	}
	if strings.TrimSpace(opts.Meter) == "" {
		warnings = append(warnings, "meter was not supplied")
	}
	return warnings
}

func (s *Service) findDuplicateSongs(ctx context.Context, opts CreateSongOptions) ([]models.Song, error) {
	byID := map[string]models.Song{}
	if opts.CCLINumber != nil {
		resources, err := s.Client.GetAll(ctx, "/services/v2/songs", url.Values{"where[ccli_number]": {strconv.Itoa(*opts.CCLINumber)}, "per_page": {"100"}})
		if err != nil {
			return nil, fmt.Errorf("searching by CCLI %d: %w", *opts.CCLINumber, err)
		}
		songs, err := songsFromResources(resources)
		if err != nil {
			return nil, err
		}
		for _, song := range songs {
			if song.Attrs.CCLINumber != nil && *song.Attrs.CCLINumber == *opts.CCLINumber {
				byID[song.ID] = song
			}
		}
	}
	titleMatches, err := s.SearchSongs(ctx, strings.Join(strings.Fields(opts.Title), " "))
	if err != nil {
		return nil, fmt.Errorf("searching by title %q: %w", opts.Title, err)
	}
	for _, song := range titleMatches {
		if normalizeTitle(song.Attrs.Title) == normalizeTitle(opts.Title) {
			byID[song.ID] = song
		}
	}
	matches := make([]models.Song, 0, len(byID))
	for _, song := range byID {
		matches = append(matches, song)
	}
	sort.Slice(matches, func(i, j int) bool {
		left, right := normalizeTitle(matches[i].Attrs.Title), normalizeTitle(matches[j].Attrs.Title)
		if left == right {
			return matches[i].ID < matches[j].ID
		}
		return left < right
	})
	return matches, nil
}

func (s *Service) createSongResource(ctx context.Context, opts CreateSongOptions) (*models.Song, error) {
	attrs := map[string]any{"title": strings.TrimSpace(opts.Title), "author": strings.TrimSpace(opts.Authors)}
	if opts.CCLINumber != nil {
		attrs["ccli_number"] = *opts.CCLINumber
	}
	body, err := json.Marshal(map[string]any{"data": map[string]any{"type": "Song", "attributes": attrs}})
	if err != nil {
		return nil, err
	}
	data, err := s.Client.Post(ctx, "/services/v2/songs", string(body))
	if err != nil {
		return nil, err
	}
	resource, err := models.ParseOne(data)
	if err != nil {
		return nil, err
	}
	var songAttrs models.SongAttrs
	if err := json.Unmarshal(resource.Attributes, &songAttrs); err != nil {
		return nil, err
	}
	return &models.Song{ID: resource.ID, Attrs: songAttrs}, nil
}

func (s *Service) createArrangementResource(ctx context.Context, songID string, opts CreateSongOptions) (*models.Arrangement, error) {
	attrs := map[string]any{"name": strings.TrimSpace(opts.ArrangementName)}
	if opts.Key != "" {
		attrs["chord_chart_key"] = opts.Key
	}
	if opts.BPM != nil {
		attrs["bpm"] = *opts.BPM
	}
	if opts.Meter != "" {
		attrs["meter"] = opts.Meter
	}
	body, err := json.Marshal(map[string]any{"data": map[string]any{"type": "Arrangement", "attributes": attrs}})
	if err != nil {
		return nil, err
	}
	data, err := s.Client.Post(ctx, "/services/v2/songs/"+songID+"/arrangements", string(body))
	if err != nil {
		return nil, err
	}
	resource, err := models.ParseOne(data)
	if err != nil {
		return nil, err
	}
	return arrangementFromResource(*resource)
}

func (s *Service) GetSongArrangement(ctx context.Context, songID, arrangementID string) (*models.Arrangement, error) {
	data, err := s.Client.Get(ctx, "/services/v2/songs/"+songID+"/arrangements/"+arrangementID, nil)
	if err != nil {
		return nil, err
	}
	resource, err := models.ParseOne(data)
	if err != nil {
		return nil, err
	}
	return arrangementFromResource(*resource)
}

func arrangementFromResource(resource models.Resource) (*models.Arrangement, error) {
	var attrs models.ArrangementAttrs
	if err := json.Unmarshal(resource.Attributes, &attrs); err != nil {
		return nil, err
	}
	return &models.Arrangement{ID: resource.ID, Attrs: attrs, Archived: attrs.ArchivedAt != ""}, nil
}

func arrangementMatches(arrangement models.Arrangement, opts CreateSongOptions) bool {
	if normalizeTitle(arrangement.Attrs.Name) != normalizeTitle(opts.ArrangementName) {
		return false
	}
	if opts.Key != "" && arrangement.Attrs.ChordChartKey != opts.Key {
		return false
	}
	if opts.BPM != nil && (arrangement.Attrs.BPM == nil || *arrangement.Attrs.BPM != *opts.BPM) {
		return false
	}
	if opts.Meter != "" && arrangement.Attrs.Meter != opts.Meter {
		return false
	}
	return true
}

func songMatches(song models.Song, opts CreateSongOptions) bool {
	if normalizeTitle(song.Attrs.Title) != normalizeTitle(opts.Title) || normalizeTitle(song.Attrs.Author) != normalizeTitle(opts.Authors) {
		return false
	}
	if opts.CCLINumber == nil {
		return song.Attrs.CCLINumber == nil
	}
	return song.Attrs.CCLINumber != nil && *song.Attrs.CCLINumber == *opts.CCLINumber
}

func partialCreateError(opts CreateSongOptions, code, message string, created []CreatedResource) error {
	songID := opts.ResumeSongID
	if songID == "" {
		for _, resource := range created {
			if resource.Type == "song" {
				songID = resource.ID
				break
			}
		}
	}
	resume := ""
	if songID != "" {
		resume = fmt.Sprintf("pco songs create --resume-song-id %s --title %q --authors %q --arrangement-name %q", songID, opts.Title, opts.Authors, opts.ArrangementName)
		if opts.CCLINumber != nil {
			resume += fmt.Sprintf(" --ccli %d", *opts.CCLINumber)
		} else {
			resume += " --no-ccli"
		}
		if opts.Key != "" {
			resume += fmt.Sprintf(" --key %q", opts.Key)
		}
		if opts.BPM != nil {
			resume += fmt.Sprintf(" --bpm %s", strconv.FormatFloat(*opts.BPM, 'f', -1, 64))
		}
		if opts.Meter != "" {
			resume += fmt.Sprintf(" --meter %q", opts.Meter)
		}
	}
	return &OperationError{Result: OperationErrorResult{SchemaVersion: songWorkflowSchemaVersion, Operation: "songs.create", Status: "partial",
		Error: OperationProblem{Code: code, Message: message}, CreatedResources: created, ResumeCommand: resume}}
}
