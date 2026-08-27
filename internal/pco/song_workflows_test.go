package pco

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/micahlee/pco-cli/internal/api"
	"github.com/micahlee/pco-cli/internal/config"
)

type workflowResponse struct {
	status int
	body   string
}

type workflowHTTPClient struct {
	t         *testing.T
	responses map[string][]workflowResponse
	requests  []songFakeRequest
}

func (f *workflowHTTPClient) Do(req *http.Request) (*http.Response, error) {
	body := ""
	if req.Body != nil {
		data, err := io.ReadAll(req.Body)
		if err != nil {
			f.t.Fatal(err)
		}
		body = string(data)
	}
	f.requests = append(f.requests, songFakeRequest{method: req.Method, path: req.URL.RequestURI(), body: body})
	key := req.Method + " " + req.URL.RequestURI()
	queue := f.responses[key]
	if len(queue) == 0 {
		f.t.Fatalf("unexpected request: %s", key)
	}
	response := queue[0]
	f.responses[key] = queue[1:]
	status := response.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(response.body)), Header: make(http.Header)}, nil
}

func (f *workflowHTTPClient) mutationCount() int {
	count := 0
	for _, request := range f.requests {
		if request.method == http.MethodPatch || request.method == http.MethodPost || request.method == http.MethodDelete {
			count++
		}
	}
	return count
}

func (f *workflowHTTPClient) requestBody(t *testing.T, method, path string) map[string]any {
	t.Helper()
	for _, request := range f.requests {
		if request.method == method && request.path == path {
			var body map[string]any
			if err := json.Unmarshal([]byte(request.body), &body); err != nil {
				t.Fatal(err)
			}
			return body
		}
	}
	t.Fatalf("missing request %s %s", method, path)
	return nil
}

func workflowService(t *testing.T, responses map[string][]workflowResponse) (*Service, *workflowHTTPClient) {
	t.Helper()
	fake := &workflowHTTPClient{t: t, responses: responses}
	client := api.New("client", "secret")
	client.HTTPClient = fake
	return &Service{Client: client, Config: &config.Config{ServiceTypeID: "643436"}}, fake
}

func response(body string) []workflowResponse { return []workflowResponse{{body: body}} }

const (
	planSearchPath = "GET /services/v2/service_types/643436/plans?after=2026-09-12&filter=after&order=sort_date&per_page=100"
	newSongPath    = "GET /services/v2/songs?per_page=100&where%5Btitle%5D=New+Song"
	planSongsPath  = "GET /services/v2/service_types/643436/plans/plan-1/items?filter=songs&per_page=50"
	itemPath       = "GET /services/v2/service_types/643436/plans/plan-1/items/item-1"
	patchItemPath  = "PATCH /services/v2/service_types/643436/plans/plan-1/items/item-1"
)

func planSearch(plans string) []workflowResponse {
	return response(`{"data":` + plans + `}`)
}

func songList(songs string) []workflowResponse { return response(`{"data":` + songs + `}`) }

func planItemJSON(title, songID, arrangementID string) string {
	arrangement := `null`
	if arrangementID != "" {
		arrangement = `{"type":"Arrangement","id":"` + arrangementID + `"}`
	}
	return `{"type":"Item","id":"item-1","attributes":{"title":"` + title + `","item_type":"song"},"relationships":{"song":{"data":{"type":"Song","id":"` + songID + `"}},"arrangement":{"data":` + arrangement + `}}}`
}

func baseReplaceResponses(arrangements string) map[string][]workflowResponse {
	return map[string][]workflowResponse{
		planSearchPath:                    planSearch(`[{"type":"Plan","id":"plan-1","attributes":{"sort_date":"2026-09-13T09:00:00-04:00"}}]`),
		newSongPath:                       songList(`[{"type":"Song","id":"song-new","attributes":{"title":"New Song","author":"New Author"}}]`),
		planSongsPath:                     songList(`[` + planItemJSON("Old Song", "song-old", "arr-old") + `]`),
		"GET /services/v2/songs/song-old": response(`{"data":{"type":"Song","id":"song-old","attributes":{"title":"Old Song","author":"Old Author"}}}`),
		"GET /services/v2/songs/song-new/arrangements?per_page=100": songList(arrangements),
	}
}

func defaultArrangementJSON() string {
	return `[{"type":"Arrangement","id":"arr-new","attributes":{"name":"Default Arrangement","archived_at":null}}]`
}

func TestReplaceSongSuccessAndVerification(t *testing.T) {
	responses := baseReplaceResponses(defaultArrangementJSON())
	responses[itemPath] = []workflowResponse{
		{body: `{"data":` + planItemJSON("Old Song", "song-old", "arr-old") + `}`},
		{body: `{"data":` + planItemJSON("New Song", "song-new", "arr-new") + `}`},
	}
	responses[patchItemPath] = response(`{"data":` + planItemJSON("New Song", "song-new", "arr-new") + `}`)
	service, fake := workflowService(t, responses)
	result, err := service.ReplaceSong(context.Background(), ReplaceSongOptions{Date: "2026-09-13", CurrentTitle: "Old Song", Replacement: "New Song"})
	if err != nil {
		t.Fatalf("ReplaceSong: %v", err)
	}
	if result.Status != "success" || !result.Mutated || !result.Verified || result.Item.SongID != "song-new" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if fake.mutationCount() != 1 {
		t.Fatalf("expected one mutation, got %d", fake.mutationCount())
	}
}

func TestReplaceSongDryRun(t *testing.T) {
	service, fake := workflowService(t, baseReplaceResponses(defaultArrangementJSON()))
	result, err := service.ReplaceSong(context.Background(), ReplaceSongOptions{Date: "2026-09-13", CurrentTitle: "Old Song", Replacement: "New Song", DryRun: true})
	if err != nil {
		t.Fatalf("ReplaceSong dry-run: %v", err)
	}
	if result.Status != "preview" || result.Mutated || result.Verified {
		t.Fatalf("unexpected preview: %#v", result)
	}
	if fake.mutationCount() != 0 {
		t.Fatal("dry-run mutated state")
	}
}

func TestReplaceSongAlreadyCompleteNoOp(t *testing.T) {
	responses := baseReplaceResponses(defaultArrangementJSON())
	delete(responses, "GET /services/v2/songs/song-old")
	delete(responses, "GET /services/v2/songs/song-new/arrangements?per_page=100")
	responses[planSongsPath] = songList(`[` + planItemJSON("New Song", "song-new", "arr-new") + `]`)
	service, fake := workflowService(t, responses)
	result, err := service.ReplaceSong(context.Background(), ReplaceSongOptions{Date: "2026-09-13", CurrentTitle: "Old Song", Replacement: "New Song"})
	if err != nil {
		t.Fatalf("ReplaceSong no-op: %v", err)
	}
	if result.Status != "noop" || result.Mutated || !result.Verified {
		t.Fatalf("unexpected no-op: %#v", result)
	}
	if fake.mutationCount() != 0 {
		t.Fatal("no-op mutated state")
	}
}

func TestReplaceSongAlreadyPresentIsAmbiguousWhenDuplicated(t *testing.T) {
	responses := baseReplaceResponses(defaultArrangementJSON())
	delete(responses, "GET /services/v2/songs/song-old")
	delete(responses, "GET /services/v2/songs/song-new/arrangements?per_page=100")
	second := strings.Replace(planItemJSON("New Song", "song-new", "arr-two"), `"item-1"`, `"item-2"`, 1)
	responses[planSongsPath] = songList(`[` + planItemJSON("New Song", "song-new", "arr-one") + `,` + second + `]`)
	service, fake := workflowService(t, responses)
	_, err := service.ReplaceSong(context.Background(), ReplaceSongOptions{Date: "2026-09-13", CurrentTitle: "Old Song", Replacement: "New Song"})
	assertOperationErrorCode(t, err, "replacement_item_ambiguous")
	if fake.mutationCount() != 0 {
		t.Fatal("ambiguous existing replacement mutated state")
	}
}

func TestReplaceSongPlanResolutionFailures(t *testing.T) {
	tests := []struct{ name, plans, code string }{
		{"missing", `[]`, "plan_not_found"},
		{"ambiguous", `[{"type":"Plan","id":"one","attributes":{"sort_date":"2026-09-13T09:00:00Z"}},{"type":"Plan","id":"two","attributes":{"sort_date":"2026-09-13T11:00:00Z"}}]`, "plan_ambiguous"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, fake := workflowService(t, map[string][]workflowResponse{planSearchPath: planSearch(test.plans)})
			_, err := service.ReplaceSong(context.Background(), ReplaceSongOptions{Date: "2026-09-13", CurrentTitle: "Old Song", Replacement: "New Song"})
			assertOperationErrorCode(t, err, test.code)
			if fake.mutationCount() != 0 {
				t.Fatal("resolution failure mutated state")
			}
		})
	}
}

func TestReplaceSongSupportsExplicitPlanID(t *testing.T) {
	responses := baseReplaceResponses(defaultArrangementJSON())
	delete(responses, planSearchPath)
	responses["GET /services/v2/service_types/643436/plans/plan-1"] = response(`{"data":{"type":"Plan","id":"plan-1","attributes":{"sort_date":"2026-09-13T09:00:00Z"}}}`)
	service, _ := workflowService(t, responses)
	result, err := service.ReplaceSong(context.Background(), ReplaceSongOptions{PlanID: "plan-1", CurrentTitle: "Old Song", Replacement: "New Song", DryRun: true})
	if err != nil || result.Plan.ID != "plan-1" {
		t.Fatalf("explicit plan: %#v, %v", result, err)
	}
}

func TestReplaceSongPlanResolutionFollowsPagination(t *testing.T) {
	responses := baseReplaceResponses(defaultArrangementJSON())
	responses[planSearchPath] = response(`{"data":[{"type":"Plan","id":"prior","attributes":{"sort_date":"2026-09-12T09:00:00Z"}}],"links":{"next":"https://api.planningcenteronline.com/services/v2/service_types/643436/plans?page=2"}}`)
	responses["GET /services/v2/service_types/643436/plans?page=2"] = planSearch(`[{"type":"Plan","id":"plan-1","attributes":{"sort_date":"2026-09-13T09:00:00Z"}}]`)
	service, _ := workflowService(t, responses)
	result, err := service.ReplaceSong(context.Background(), ReplaceSongOptions{Date: "2026-09-13", CurrentTitle: "Old Song", Replacement: "New Song", DryRun: true})
	if err != nil || result.Plan.ID != "plan-1" {
		t.Fatalf("paginated plan resolution: %#v, %v", result, err)
	}
}

func TestReplaceSongCurrentItemFailures(t *testing.T) {
	for _, test := range []struct{ name, items, code string }{
		{"missing", `[]`, "current_song_not_found"},
		{"ambiguous", `[` + planItemJSON("Old Song", "old-1", "") + `,` + strings.Replace(planItemJSON("Old Song", "old-2", ""), `"item-1"`, `"item-2"`, 1) + `]`, "current_song_ambiguous"},
	} {
		t.Run(test.name, func(t *testing.T) {
			responses := baseReplaceResponses(defaultArrangementJSON())
			responses[planSongsPath] = songList(test.items)
			delete(responses, "GET /services/v2/songs/song-old")
			delete(responses, "GET /services/v2/songs/song-new/arrangements?per_page=100")
			service, fake := workflowService(t, responses)
			_, err := service.ReplaceSong(context.Background(), ReplaceSongOptions{Date: "2026-09-13", CurrentTitle: "Old Song", Replacement: "New Song"})
			assertOperationErrorCode(t, err, test.code)
			if fake.mutationCount() != 0 {
				t.Fatal("item failure mutated state")
			}
		})
	}
}

func TestReplaceSongReplacementLibraryFailures(t *testing.T) {
	for _, test := range []struct{ name, songs, code string }{
		{"missing", `[]`, "replacement_song_not_found"},
		{"ambiguous", `[{"type":"Song","id":"one","attributes":{"title":"New Song"}},{"type":"Song","id":"two","attributes":{"title":" new   song "}}]`, "replacement_song_ambiguous"},
	} {
		t.Run(test.name, func(t *testing.T) {
			responses := map[string][]workflowResponse{planSearchPath: planSearch(`[{"type":"Plan","id":"plan-1","attributes":{"sort_date":"2026-09-13T09:00:00Z"}}]`), newSongPath: songList(test.songs)}
			service, fake := workflowService(t, responses)
			_, err := service.ReplaceSong(context.Background(), ReplaceSongOptions{Date: "2026-09-13", CurrentTitle: "Old Song", Replacement: "New Song"})
			assertOperationErrorCode(t, err, test.code)
			if fake.mutationCount() != 0 {
				t.Fatal("song failure mutated state")
			}
		})
	}
}

func TestReplaceSongStaleStateProtection(t *testing.T) {
	responses := baseReplaceResponses(defaultArrangementJSON())
	responses[itemPath] = response(`{"data":` + planItemJSON("Unexpected Song", "other-song", "other-arr") + `}`)
	service, fake := workflowService(t, responses)
	_, err := service.ReplaceSong(context.Background(), ReplaceSongOptions{Date: "2026-09-13", CurrentTitle: "Old Song", Replacement: "New Song"})
	assertOperationErrorCode(t, err, "stale_state")
	if fake.mutationCount() != 0 {
		t.Fatal("stale state mutated plan")
	}
}

func TestReplaceSongArrangementResolution(t *testing.T) {
	tests := []struct {
		name, arrangements string
		arrangementID      string
		songOnly           bool
		wantID, code       string
	}{
		{"default", `[{"type":"Arrangement","id":"default","attributes":{"name":"Default Arrangement"}},{"type":"Arrangement","id":"other","attributes":{"name":"Band"}}]`, "", false, "default", ""},
		{"sole", `[{"type":"Arrangement","id":"sole","attributes":{"name":"Band"}}]`, "", false, "sole", ""},
		{"explicit", `[{"type":"Arrangement","id":"one","attributes":{"name":"One"}},{"type":"Arrangement","id":"two","attributes":{"name":"Two"}}]`, "two", false, "two", ""},
		{"missing", `[]`, "", false, "", "arrangement_resolution_failed"},
		{"ambiguous", `[{"type":"Arrangement","id":"one","attributes":{"name":"One"}},{"type":"Arrangement","id":"two","attributes":{"name":"Two"}}]`, "", false, "", "arrangement_resolution_failed"},
		{"duplicate-default", `[{"type":"Arrangement","id":"one","attributes":{"name":"Default Arrangement"}},{"type":"Arrangement","id":"two","attributes":{"name":"Default Arrangement"}}]`, "", false, "", "arrangement_resolution_failed"},
		{"song-only", `[]`, "", true, "", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			responses := baseReplaceResponses(test.arrangements)
			if test.songOnly {
				delete(responses, "GET /services/v2/songs/song-new/arrangements?per_page=100")
			}
			service, fake := workflowService(t, responses)
			result, err := service.ReplaceSong(context.Background(), ReplaceSongOptions{Date: "2026-09-13", CurrentTitle: "Old Song", Replacement: "New Song", ArrangementID: test.arrangementID, SongOnly: test.songOnly, DryRun: true})
			if test.code != "" {
				assertOperationErrorCode(t, err, test.code)
				return
			}
			if err != nil {
				t.Fatalf("ReplaceSong: %v", err)
			}
			got := ""
			if result.Arrangement != nil {
				got = result.Arrangement.ID
			}
			if got != test.wantID {
				t.Fatalf("want arrangement %q, got %q", test.wantID, got)
			}
			if fake.mutationCount() != 0 {
				t.Fatal("dry-run mutated state")
			}
		})
	}
}

func TestReplaceSongVerificationFailure(t *testing.T) {
	responses := baseReplaceResponses(defaultArrangementJSON())
	responses[itemPath] = []workflowResponse{{body: `{"data":` + planItemJSON("Old Song", "song-old", "arr-old") + `}`}, {body: `{"data":` + planItemJSON("Wrong Song", "wrong", "arr-new") + `}`}}
	responses[patchItemPath] = response(`{"data":` + planItemJSON("New Song", "song-new", "arr-new") + `}`)
	service, _ := workflowService(t, responses)
	_, err := service.ReplaceSong(context.Background(), ReplaceSongOptions{Date: "2026-09-13", CurrentTitle: "Old Song", Replacement: "New Song"})
	assertOperationErrorCode(t, err, "verification_failed")
}

func TestReplaceSongRejectsMalformedOrMissingDate(t *testing.T) {
	service, fake := workflowService(t, map[string][]workflowResponse{})
	for _, opts := range []ReplaceSongOptions{{CurrentTitle: "Old", Replacement: "New"}, {Date: "2026-9-13", CurrentTitle: "Old", Replacement: "New"}} {
		_, err := service.ReplaceSong(context.Background(), opts)
		if err == nil {
			t.Fatal("expected date error")
		}
	}
	if len(fake.requests) != 0 {
		t.Fatal("invalid dates should fail before requests")
	}
}

func TestCreateSongAndArrangementSuccess(t *testing.T) {
	responses := createResponses(false)
	service, fake := workflowService(t, responses)
	result, err := service.CreateSong(context.Background(), CreateSongOptions{Title: "Brand New", Authors: "A Writer", ArrangementName: "Default Arrangement", NoCCLI: true})
	if err != nil {
		t.Fatalf("CreateSong: %v", err)
	}
	if result.Status != "success" || !result.Verified || len(result.CreatedResources) != 2 || len(result.Warnings) != 3 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if fake.mutationCount() != 2 {
		t.Fatalf("expected song and arrangement POSTs, got %d", fake.mutationCount())
	}
}

func TestCreateSongWithCCLIAndArrangementMetadata(t *testing.T) {
	ccli := 123456
	responses := map[string][]workflowResponse{
		"GET /services/v2/songs?per_page=100&where%5Bccli_number%5D=123456": songList(`[]`),
		"GET /services/v2/songs?per_page=100&where%5Btitle%5D=Brand+New":    songList(`[]`),
		"POST /services/v2/songs":                                       response(`{"data":{"type":"Song","id":"song-created","attributes":{"title":"Brand New","author":"A Writer","ccli_number":123456}}}`),
		"GET /services/v2/songs/song-created":                           response(`{"data":{"type":"Song","id":"song-created","attributes":{"title":"Brand New","author":"A Writer","ccli_number":123456}}}`),
		"GET /services/v2/songs/song-created/arrangements?per_page=100": songList(`[]`),
		"POST /services/v2/songs/song-created/arrangements":             response(`{"data":{"type":"Arrangement","id":"arr-created","attributes":{"name":"Default Arrangement","chord_chart_key":"D","bpm":72,"meter":"4/4"}}}`),
		"GET /services/v2/songs/song-created/arrangements/arr-created":  response(`{"data":{"type":"Arrangement","id":"arr-created","attributes":{"name":"Default Arrangement","chord_chart_key":"D","bpm":72,"meter":"4/4"}}}`),
	}
	service, fake := workflowService(t, responses)
	bpm := 72.0
	result, err := service.CreateSong(context.Background(), CreateSongOptions{Title: "Brand New", Authors: "A Writer", ArrangementName: "Default Arrangement", CCLINumber: &ccli, Key: "D", BPM: &bpm, Meter: "4/4"})
	if err != nil {
		t.Fatalf("CreateSong: %v", err)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", result.Warnings)
	}
	songBody := fake.requestBody(t, http.MethodPost, "/services/v2/songs")
	songAttrs := songBody["data"].(map[string]any)["attributes"].(map[string]any)
	if songAttrs["ccli_number"] != float64(123456) {
		t.Fatalf("missing CCLI payload: %#v", songAttrs)
	}
	arrBody := fake.requestBody(t, http.MethodPost, "/services/v2/songs/song-created/arrangements")
	arrAttrs := arrBody["data"].(map[string]any)["attributes"].(map[string]any)
	if arrAttrs["chord_chart_key"] != "D" || arrAttrs["bpm"] != float64(72) || arrAttrs["meter"] != "4/4" {
		t.Fatalf("missing arrangement metadata: %#v", arrAttrs)
	}
	for _, request := range fake.requests {
		if strings.Contains(request.path, "/plans/") {
			t.Fatalf("create must not place song in a plan: %#v", request)
		}
	}
}

func TestCreateSongRequiresExplicitMetadata(t *testing.T) {
	tests := []CreateSongOptions{
		{Authors: "A Writer", ArrangementName: "Default Arrangement", NoCCLI: true},
		{Title: "Brand New", ArrangementName: "Default Arrangement", NoCCLI: true},
		{Title: "Brand New", Authors: "A Writer", NoCCLI: true},
		{Title: "Brand New", Authors: "A Writer", ArrangementName: "Default Arrangement"},
		{Title: "Brand New", Authors: "A Writer", ArrangementName: "Default Arrangement", NoCCLI: true, Meter: "13/8"},
	}
	for i, opts := range tests {
		service, fake := workflowService(t, map[string][]workflowResponse{})
		_, err := service.CreateSong(context.Background(), opts)
		if err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
		if len(fake.requests) != 0 {
			t.Fatalf("case %d: validation should precede API requests", i)
		}
	}
}

func createResponses(includeTitleDuplicate bool) map[string][]workflowResponse {
	titleData := `[]`
	if includeTitleDuplicate {
		titleData = `[{"type":"Song","id":"existing","attributes":{"title":"Brand New","author":"Someone"}}]`
	}
	return map[string][]workflowResponse{
		"GET /services/v2/songs?per_page=100&where%5Btitle%5D=Brand+New": songList(titleData),
		"POST /services/v2/songs":                                       response(`{"data":{"type":"Song","id":"song-created","attributes":{"title":"Brand New","author":"A Writer","ccli_number":null}}}`),
		"GET /services/v2/songs/song-created":                           response(`{"data":{"type":"Song","id":"song-created","attributes":{"title":"Brand New","author":"A Writer","ccli_number":null}}}`),
		"GET /services/v2/songs/song-created/arrangements?per_page=100": songList(`[]`),
		"POST /services/v2/songs/song-created/arrangements":             response(`{"data":{"type":"Arrangement","id":"arr-created","attributes":{"name":"Default Arrangement"}}}`),
		"GET /services/v2/songs/song-created/arrangements/arr-created":  response(`{"data":{"type":"Arrangement","id":"arr-created","attributes":{"name":"Default Arrangement"}}}`),
	}
}

func TestCreateSongDuplicateTitleAndCCLIProtection(t *testing.T) {
	t.Run("title", func(t *testing.T) {
		service, fake := workflowService(t, createResponses(true))
		_, err := service.CreateSong(context.Background(), CreateSongOptions{Title: " Brand   New ", Authors: "A Writer", ArrangementName: "Default Arrangement", NoCCLI: true})
		assertOperationErrorCode(t, err, "duplicate_detected")
		if fake.mutationCount() != 0 {
			t.Fatal("duplicate title mutated state")
		}
	})
	t.Run("ccli", func(t *testing.T) {
		ccli := 123
		service, fake := workflowService(t, map[string][]workflowResponse{
			"GET /services/v2/songs?per_page=100&where%5Bccli_number%5D=123": songList(`[{"type":"Song","id":"existing","attributes":{"title":"Other","ccli_number":123}}]`),
			"GET /services/v2/songs?per_page=100&where%5Btitle%5D=Brand+New": songList(`[]`),
		})
		_, err := service.CreateSong(context.Background(), CreateSongOptions{Title: "Brand New", Authors: "A Writer", ArrangementName: "Default Arrangement", CCLINumber: &ccli})
		assertOperationErrorCode(t, err, "duplicate_detected")
		if fake.mutationCount() != 0 {
			t.Fatal("duplicate CCLI mutated state")
		}
	})
}

func TestCreateSongAllowDuplicate(t *testing.T) {
	service, _ := workflowService(t, createResponses(true))
	result, err := service.CreateSong(context.Background(), CreateSongOptions{Title: "Brand New", Authors: "A Writer", ArrangementName: "Default Arrangement", NoCCLI: true, AllowDuplicate: true})
	if err != nil || result.Status != "success" {
		t.Fatalf("allow duplicate: %#v, %v", result, err)
	}
}

func TestCreateSongPartialFailureAndResume(t *testing.T) {
	t.Run("partial", func(t *testing.T) {
		responses := createResponses(false)
		responses["POST /services/v2/songs/song-created/arrangements"] = []workflowResponse{{status: http.StatusUnprocessableEntity, body: `{"errors":[{"detail":"bad arrangement"}]}`}}
		delete(responses, "GET /services/v2/songs/song-created/arrangements/arr-created")
		service, fake := workflowService(t, responses)
		_, err := service.CreateSong(context.Background(), CreateSongOptions{Title: "Brand New", Authors: "A Writer", ArrangementName: "Default Arrangement", NoCCLI: true})
		var operationErr *OperationError
		if err == nil || !asOperationError(err, &operationErr) {
			t.Fatalf("expected operation error, got %v", err)
		}
		if operationErr.Result.Status != "partial" || len(operationErr.Result.CreatedResources) != 1 || !strings.Contains(operationErr.Result.ResumeCommand, "--resume-song-id song-created") {
			t.Fatalf("unexpected partial result: %#v", operationErr.Result)
		}
		if fake.mutationCount() != 2 {
			t.Fatalf("expected two attempted creates, got %d", fake.mutationCount())
		}
	})
	t.Run("resume", func(t *testing.T) {
		service, fake := workflowService(t, map[string][]workflowResponse{
			"GET /services/v2/songs/song-created":                           response(`{"data":{"type":"Song","id":"song-created","attributes":{"title":"Brand New","author":"A Writer","ccli_number":null}}}`),
			"GET /services/v2/songs/song-created/arrangements?per_page=100": songList(`[]`),
			"POST /services/v2/songs/song-created/arrangements":             response(`{"data":{"type":"Arrangement","id":"arr-created","attributes":{"name":"Default Arrangement"}}}`),
			"GET /services/v2/songs/song-created/arrangements/arr-created":  response(`{"data":{"type":"Arrangement","id":"arr-created","attributes":{"name":"Default Arrangement"}}}`),
		})
		result, err := service.CreateSong(context.Background(), CreateSongOptions{Title: "Brand New", Authors: "A Writer", ArrangementName: "Default Arrangement", NoCCLI: true, ResumeSongID: "song-created"})
		if err != nil || !result.Resumed || result.Status != "success" {
			t.Fatalf("resume: %#v, %v", result, err)
		}
		if fake.mutationCount() != 1 {
			t.Fatalf("resume should create only arrangement, got %d mutations", fake.mutationCount())
		}
	})
}

func TestCreateSongVerificationFailure(t *testing.T) {
	responses := createResponses(false)
	responses["GET /services/v2/songs/song-created/arrangements/arr-created"] = response(`{"data":{"type":"Arrangement","id":"arr-created","attributes":{"name":"Wrong"}}}`)
	service, _ := workflowService(t, responses)
	_, err := service.CreateSong(context.Background(), CreateSongOptions{Title: "Brand New", Authors: "A Writer", ArrangementName: "Default Arrangement", NoCCLI: true})
	assertOperationErrorCode(t, err, "verification_failed")
}

func TestSearchSongsFollowsPagination(t *testing.T) {
	service, _ := workflowService(t, map[string][]workflowResponse{
		"GET /services/v2/songs?per_page=100&where%5Btitle%5D=Song": response(`{"data":[{"type":"Song","id":"one","attributes":{"title":"Song One"}}],"links":{"next":"https://api.planningcenteronline.com/services/v2/songs?page=2"}}`),
		"GET /services/v2/songs?page=2":                             response(`{"data":[{"type":"Song","id":"two","attributes":{"title":"Song Two"}}]}`),
	})
	songs, err := service.SearchSongs(context.Background(), "Song")
	if err != nil || len(songs) != 2 {
		t.Fatalf("pagination: %#v, %v", songs, err)
	}
}

func TestSongWorkflowJSONShapesAreStable(t *testing.T) {
	result := SongReplaceResult{SchemaVersion: "1", Operation: "songs.replace", Status: "preview", Warnings: []string{}}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var shape map[string]any
	if err := json.Unmarshal(data, &shape); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"schema_version", "operation", "status", "mutated", "verified", "plan", "item", "current", "replacement", "arrangement", "warnings"} {
		if _, ok := shape[key]; !ok {
			t.Fatalf("missing stable key %q in %s", key, data)
		}
	}
	errResult := OperationErrorResult{SchemaVersion: "1", Operation: "songs.create", Status: "error", Error: OperationProblem{Code: "x", Message: "y"}}
	data, _ = json.Marshal(errResult)
	shape = nil
	_ = json.Unmarshal(data, &shape)
	for _, key := range []string{"schema_version", "operation", "status", "error"} {
		if _, ok := shape[key]; !ok {
			t.Fatalf("missing error key %q", key)
		}
	}
}

func assertOperationErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	var operationErr *OperationError
	if !asOperationError(err, &operationErr) {
		t.Fatalf("expected OperationError %q, got %T: %v", code, err, err)
	}
	if operationErr.Result.Error.Code != code {
		t.Fatalf("expected code %q, got %#v", code, operationErr.Result)
	}
}

func asOperationError(err error, target **OperationError) bool {
	if err == nil {
		return false
	}
	value, ok := err.(*OperationError)
	if ok {
		*target = value
	}
	return ok
}
