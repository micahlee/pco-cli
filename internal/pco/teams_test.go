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

func TestListTeamSignupsIncludesTeamAndRawAttributes(t *testing.T) {
	service, _ := newTeamSignupTestService(t, map[string]string{
		"/services/v2/service_types/643436/plans/plan-1/team_signups?include=team&per_page=100": `{
			"data": [
				{
					"type": "TeamSignup",
					"id": "signup-1",
					"attributes": {
						"signups_enabled": true,
						"max_signups": 8
					},
					"relationships": {
						"team": {"data": {"type": "Team", "id": "2461416"}}
					}
				}
			],
			"included": [
				{
					"type": "Team",
					"id": "2461416",
					"attributes": {"name": "Band"}
				}
			]
		}`,
	})

	signups, err := service.ListTeamSignups(context.Background(), "plan-1", "")
	if err != nil {
		t.Fatalf("ListTeamSignups returned error: %v", err)
	}

	if len(signups) != 1 {
		t.Fatalf("expected 1 signup, got %#v", signups)
	}
	signup := signups[0]
	if signup.ID != "signup-1" || signup.PlanID != "plan-1" || signup.TeamID != "2461416" || signup.TeamName != "Band" {
		t.Fatalf("expected signup identity, got %#v", signup)
	}
	if signup.Attrs.SignupsEnabled == nil || !*signup.Attrs.SignupsEnabled {
		t.Fatalf("expected signups enabled, got %#v", signup.Attrs.SignupsEnabled)
	}
	if signup.Attrs.Raw["max_signups"].(float64) != 8 {
		t.Fatalf("expected raw max_signups, got %#v", signup.Attrs.Raw)
	}
}

func TestEnableSignupsReturnsExistingSignupWithoutPosting(t *testing.T) {
	service, fake := newTeamSignupTestService(t, map[string]string{
		"/services/v2/service_types/643436/plans/plan-1/team_signups?include=team&per_page=100": `{
			"data": [
				{
					"type": "TeamSignup",
					"id": "signup-1",
					"attributes": {"signups_enabled": true},
					"relationships": {
						"team": {"data": {"type": "Team", "id": "2461416"}}
					}
				}
			],
			"included": [
				{"type": "Team", "id": "2461416", "attributes": {"name": "Band"}}
			]
		}`,
	})

	result, err := service.EnableSignups(context.Background(), "plan-1", "")
	if err != nil {
		t.Fatalf("EnableSignups returned error: %v", err)
	}

	if result.Created || result.Updated {
		t.Fatalf("expected idempotent no-op, got %#v", result)
	}
	if result.TeamSignup.ID != "signup-1" || result.TeamSignup.TeamName != "Band" {
		t.Fatalf("expected existing signup, got %#v", result)
	}
	if fake.sawMethod(http.MethodPost) {
		t.Fatal("did not expect POST when signup already exists")
	}
}

func TestEnableSignupsPatchesExistingDisabledSignup(t *testing.T) {
	service, fake := newTeamSignupTestService(t, map[string]string{
		"/services/v2/service_types/643436/plans/plan-1/team_signups?include=team&per_page=100": `{
			"data": [
				{
					"type": "TeamSignup",
					"id": "signup-1",
					"attributes": {"signups_enabled": false},
					"relationships": {
						"team": {"data": {"type": "Team", "id": "2461416"}}
					}
				}
			]
		}`,
		"/services/v2/service_types/643436/plans/plan-1/team_signups/signup-1": `{
			"data": {
				"type": "TeamSignup",
				"id": "signup-1",
				"attributes": {"signups_enabled": true},
				"relationships": {
					"team": {"data": {"type": "Team", "id": "2461416"}}
				}
			}
		}`,
	})

	result, err := service.EnableSignups(context.Background(), "plan-1", "")
	if err != nil {
		t.Fatalf("EnableSignups returned error: %v", err)
	}

	if !result.Updated || result.Created {
		t.Fatalf("expected existing signup to be updated, got %#v", result)
	}
	patch := fake.requestJSON(t, http.MethodPatch, "/services/v2/service_types/643436/plans/plan-1/team_signups/signup-1")
	data := patch["data"].(map[string]any)
	attrs := data["attributes"].(map[string]any)
	if attrs["signups_enabled"] != true {
		t.Fatalf("expected signups_enabled true patch, got %#v", attrs)
	}
	if fake.sawMethod(http.MethodPost) {
		t.Fatal("did not expect POST when disabled signup already exists")
	}
}

func TestEnableSignupsCreatesWhenMissing(t *testing.T) {
	service, fake := newTeamSignupTestService(t, map[string]string{
		"/services/v2/service_types/643436/plans/plan-1/team_signups?include=team&per_page=100": `{"data": []}`,
		"/services/v2/service_types/643436/plans/plan-1/team_signups": `{
			"data": {
				"type": "TeamSignup",
				"id": "signup-1",
				"attributes": {"signups_enabled": true},
				"relationships": {
					"team": {"data": {"type": "Team", "id": "2461416"}}
				}
			}
		}`,
	})

	result, err := service.EnableSignups(context.Background(), "plan-1", "")
	if err != nil {
		t.Fatalf("EnableSignups returned error: %v", err)
	}

	if !result.Created || result.Updated || result.TeamSignup.ID != "signup-1" {
		t.Fatalf("expected created signup, got %#v", result)
	}
	post := fake.requestJSON(t, http.MethodPost, "/services/v2/service_types/643436/plans/plan-1/team_signups")
	data := post["data"].(map[string]any)
	relationships := data["relationships"].(map[string]any)
	team := relationships["team"].(map[string]any)
	teamData := team["data"].(map[string]any)
	if teamData["id"] != "2461416" {
		t.Fatalf("expected default band team relationship, got %#v", teamData)
	}
}

func TestEnableSignupsErrorsOnDuplicateTeamSignups(t *testing.T) {
	service, fake := newTeamSignupTestService(t, map[string]string{
		"/services/v2/service_types/643436/plans/plan-1/team_signups?include=team&per_page=100": `{
			"data": [
				{
					"type": "TeamSignup",
					"id": "signup-1",
					"attributes": {"signups_enabled": true},
					"relationships": {"team": {"data": {"type": "Team", "id": "2461416"}}}
				},
				{
					"type": "TeamSignup",
					"id": "signup-2",
					"attributes": {"signups_enabled": false},
					"relationships": {"team": {"data": {"type": "Team", "id": "2461416"}}}
				}
			]
		}`,
	})

	_, err := service.EnableSignups(context.Background(), "plan-1", "")
	if err == nil {
		t.Fatal("expected duplicate TeamSignup error")
	}
	if !strings.Contains(err.Error(), "2 TeamSignup records") {
		t.Fatalf("expected duplicate count in error, got %v", err)
	}
	if fake.sawMethod(http.MethodPost) || fake.sawMethod(http.MethodPatch) {
		t.Fatal("did not expect mutation when duplicates exist")
	}
}

func TestListPlansForMonthFollowsPagination(t *testing.T) {
	service, _ := newTeamSignupTestService(t, map[string]string{
		"/services/v2/service_types/643436/plans?filter=future&order=sort_date&per_page=100": `{
			"data": [
				{
					"type": "Plan",
					"id": "june-plan",
					"attributes": {"title": "June", "sort_date": "2026-06-28T09:00:00Z"}
				}
			],
			"links": {"next": "https://api.planningcenteronline.com/services/v2/service_types/643436/plans?page=2"}
		}`,
		"/services/v2/service_types/643436/plans?page=2": `{
			"data": [
				{
					"type": "Plan",
					"id": "july-plan",
					"attributes": {"title": "July", "sort_date": "2026-07-05T09:00:00Z"}
				}
			]
		}`,
		"/services/v2/service_types/643436/plans?filter=past&order=-sort_date&per_page=100": `{"data": []}`,
	})

	plans, err := service.ListPlansForMonth(context.Background(), "2026-07")
	if err != nil {
		t.Fatalf("ListPlansForMonth returned error: %v", err)
	}
	if len(plans) != 1 || plans[0].ID != "july-plan" {
		t.Fatalf("expected paged July plan, got %#v", plans)
	}
}

func TestEnableSignupsMonthReportsPerPlanErrors(t *testing.T) {
	service, fake := newTeamSignupTestService(t, map[string]string{
		"/services/v2/service_types/643436/plans?filter=future&order=sort_date&per_page=100": `{
			"data": [
				{
					"type": "Plan",
					"id": "plan-1",
					"attributes": {"title": "July 1", "sort_date": "2026-07-05T09:00:00Z"}
				},
				{
					"type": "Plan",
					"id": "plan-2",
					"attributes": {"title": "July 2", "sort_date": "2026-07-12T09:00:00Z"}
				}
			]
		}`,
		"/services/v2/service_types/643436/plans?filter=past&order=-sort_date&per_page=100": `{"data": []}`,
		"/services/v2/service_types/643436/plans/plan-1/team_signups?include=team&per_page=100": `{
			"data": [
				{
					"type": "TeamSignup",
					"id": "signup-1",
					"attributes": {"signups_enabled": true},
					"relationships": {"team": {"data": {"type": "Team", "id": "2461416"}}}
				}
			]
		}`,
		"/services/v2/service_types/643436/plans/plan-2/team_signups?include=team&per_page=100": `{
			"data": [
				{
					"type": "TeamSignup",
					"id": "signup-2",
					"attributes": {"signups_enabled": true},
					"relationships": {"team": {"data": {"type": "Team", "id": "2461416"}}}
				},
				{
					"type": "TeamSignup",
					"id": "signup-3",
					"attributes": {"signups_enabled": true},
					"relationships": {"team": {"data": {"type": "Team", "id": "2461416"}}}
				}
			]
		}`,
	})

	results, err := service.EnableSignupsMonth(context.Background(), "2026-07", "")
	if err != nil {
		t.Fatalf("EnableSignupsMonth returned top-level error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected two results, got %#v", results)
	}
	if results[0].PlanID != "plan-1" || results[0].Error != "" || results[0].TeamSignup.ID != "signup-1" {
		t.Fatalf("expected successful first result, got %#v", results[0])
	}
	if results[1].PlanID != "plan-2" || !strings.Contains(results[1].Error, "2 TeamSignup records") {
		t.Fatalf("expected duplicate error second result, got %#v", results[1])
	}
	if fake.sawMethod(http.MethodPost) || fake.sawMethod(http.MethodPatch) {
		t.Fatal("did not expect mutations for already-enabled or duplicate plans")
	}
}

func TestMusicMonthKeepsOverviewWhenSignupLookupFails(t *testing.T) {
	service, fake := newTeamSignupTestService(t, map[string]string{
		"/services/v2/service_types/643436/plans?filter=future&order=sort_date&per_page=100": `{
			"data": [
				{
					"type": "Plan",
					"id": "plan-1",
					"attributes": {"title": "July 1", "sort_date": "2026-07-05T09:00:00Z"}
				}
			]
		}`,
		"/services/v2/service_types/643436/plans?filter=past&order=-sort_date&per_page=100":     `{"data": []}`,
		"/services/v2/service_types/643436/teams/2461416/people?per_page=100":                   `{"data": []}`,
		"/services/v2/service_types/643436/plans/plan-1/team_members?per_page=50":               `{"data": []}`,
		"/services/v2/service_types/643436/plans/plan-1/team_signups?include=team&per_page=100": `{"errors": [{"detail": "not found"}]}`,
	})
	fake.statuses = map[string]int{
		"/services/v2/service_types/643436/plans/plan-1/team_signups?include=team&per_page=100": http.StatusNotFound,
	}

	month, err := service.MusicMonth(context.Background(), "2026-07")
	if err != nil {
		t.Fatalf("MusicMonth returned error: %v", err)
	}
	if len(month.Plans) != 1 {
		t.Fatalf("expected one plan, got %#v", month.Plans)
	}
	if month.Plans[0].BandSignupError == "" {
		t.Fatalf("expected signup lookup error on plan, got %#v", month.Plans[0])
	}
}

func newTeamSignupTestService(t *testing.T, responses map[string]string) (*Service, *teamSignupFakeHTTPClient) {
	t.Helper()
	client := api.New("client", "secret")
	fake := &teamSignupFakeHTTPClient{t: t, responses: responses}
	client.HTTPClient = fake
	return &Service{
		Client: client,
		Config: &config.Config{ServiceTypeID: "643436", BandTeamID: "2461416"},
	}, fake
}

type teamSignupFakeHTTPClient struct {
	t         *testing.T
	responses map[string]string
	statuses  map[string]int
	requests  []teamSignupFakeRequest
}

type teamSignupFakeRequest struct {
	method string
	path   string
	body   string
}

func (c *teamSignupFakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	key := req.URL.Path
	if req.URL.RawQuery != "" {
		key += "?" + req.URL.RawQuery
	}
	var requestBody string
	if req.Body != nil {
		data, err := io.ReadAll(req.Body)
		if err != nil {
			c.t.Fatalf("reading request body: %v", err)
		}
		requestBody = string(data)
	}
	c.requests = append(c.requests, teamSignupFakeRequest{
		method: req.Method,
		path:   req.URL.Path,
		body:   requestBody,
	})

	body, ok := c.responses[key]
	if !ok {
		c.t.Fatalf("unexpected request path: %s", req.URL.String())
	}
	status := c.statuses[key]
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}, nil
}

func (c *teamSignupFakeHTTPClient) sawMethod(method string) bool {
	return c.countMethod(method) > 0
}

func (c *teamSignupFakeHTTPClient) countMethod(method string) int {
	count := 0
	for _, req := range c.requests {
		if req.method == method {
			count++
		}
	}
	return count
}

func (c *teamSignupFakeHTTPClient) requestJSON(t *testing.T, method, path string) map[string]any {
	t.Helper()
	for _, req := range c.requests {
		if req.method == method && req.path == path {
			var body map[string]any
			if err := json.Unmarshal([]byte(req.body), &body); err != nil {
				t.Fatalf("unmarshaling request body %q: %v", req.body, err)
			}
			return body
		}
	}
	t.Fatalf("expected %s %s request in %#v", method, path, c.requests)
	return nil
}
