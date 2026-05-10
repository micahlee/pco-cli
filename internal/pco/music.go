package pco

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/micahlee/pco-cli/internal/models"
)

// musicPositions are the positions relevant to music scheduling.
var musicPositions = map[string]bool{
	"Music Lead": true, "Acoustic Guitar": true, "Electric Guitar": true,
	"Bass Guitar": true, "Keys": true, "Percussion": true,
	"Piano": true, "Vocals": true, "Worship Leader": true,
}

// ListBandMembers returns all members of the Band team.
func (s *Service) ListBandMembers(ctx context.Context) ([]models.BandMember, error) {
	data, err := s.Client.Get(ctx,
		s.servicePath()+"/teams/"+s.Config.BandTeamID+"/people",
		url.Values{"per_page": {"100"}})
	if err != nil {
		return nil, err
	}

	resources, _, err := models.ParseList(data)
	if err != nil {
		return nil, err
	}

	members := make([]models.BandMember, len(resources))
	for i, r := range resources {
		var attrs models.TeamPersonAttrs
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			return nil, err
		}
		members[i] = models.BandMember{
			PersonID:         r.ID,
			Name:             attrs.FullName,
			TypicalPositions: s.Config.TypicalPositions[r.ID],
		}
	}

	sort.Slice(members, func(i, j int) bool {
		return members[i].Name < members[j].Name
	})

	return members, nil
}

// CheckAvailability checks band member availability on a date.
func (s *Service) CheckAvailability(ctx context.Context, dateStr string) ([]models.Availability, error) {
	checkDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}

	members, err := s.ListBandMembers(ctx)
	if err != nil {
		return nil, err
	}

	var results []models.Availability
	for _, m := range members {
		blockouts, err := s.ListBlockoutsForPerson(ctx, m.PersonID)
		if err != nil {
			return nil, err
		}

		avail := models.Availability{Member: m, Available: true}
		for _, b := range blockouts {
			start, _ := time.Parse("2006-01-02", b.Attrs.StartsAt[:10])
			end, _ := time.Parse("2006-01-02", b.Attrs.EndsAt[:10])
			if !checkDate.Before(start) && !checkDate.After(end) {
				avail.Available = false
				avail.Reason = b.Attrs.Reason
				break
			}
		}
		results = append(results, avail)
	}
	return results, nil
}

// MusicMonth returns a full music scheduling overview for a month.
func (s *Service) MusicMonth(ctx context.Context, yearMonth string) (*models.MusicMonth, error) {
	year, _ := strconv.Atoi(yearMonth[:4])
	month, _ := strconv.Atoi(yearMonth[5:7])

	// Fetch future and recent past plans, filter to this month
	futurePlans, err := s.Client.Get(ctx, s.servicePath()+"/plans",
		url.Values{"filter": {"future"}, "per_page": {"25"}, "order": {"sort_date"}})
	if err != nil {
		return nil, err
	}
	pastPlans, err := s.Client.Get(ctx, s.servicePath()+"/plans",
		url.Values{"filter": {"past"}, "per_page": {"10"}, "order": {"-sort_date"}})
	if err != nil {
		return nil, err
	}

	futureResources, _, _ := models.ParseList(futurePlans)
	pastResources, _, _ := models.ParseList(pastPlans)

	// Reverse past so oldest first, then append future
	for i, j := 0, len(pastResources)-1; i < j; i, j = i+1, j-1 {
		pastResources[i], pastResources[j] = pastResources[j], pastResources[i]
	}
	allResources := append(pastResources, futureResources...)

	var monthPlans []models.Resource
	for _, r := range allResources {
		var attrs models.PlanAttrs
		json.Unmarshal(r.Attributes, &attrs)
		dateStr := attrs.SortDate
		if len(dateStr) < 10 {
			continue
		}
		d, _ := time.Parse("2006-01-02", dateStr[:10])
		if d.Year() == year && int(d.Month()) == month {
			monthPlans = append(monthPlans, r)
		}
	}

	if len(monthPlans) == 0 {
		return &models.MusicMonth{YearMonth: yearMonth}, nil
	}

	// Get band members and their blockouts
	bandMembers, err := s.ListBandMembers(ctx)
	if err != nil {
		return nil, err
	}
	bandMap := make(map[string]string) // pid -> name
	for _, m := range bandMembers {
		bandMap[m.PersonID] = m.Name
	}

	blockoutsByPerson := make(map[string][]models.Blockout)
	for _, m := range bandMembers {
		bos, err := s.ListBlockoutsForPerson(ctx, m.PersonID)
		if err != nil {
			return nil, err
		}
		blockoutsByPerson[m.PersonID] = bos
	}

	// Build plan data
	appearanceCounts := make(map[string]map[string]bool) // personName -> set of dates
	var resultPlans []models.MusicMonthPlan

	for _, plan := range monthPlans {
		var pattrs models.PlanAttrs
		json.Unmarshal(plan.Attributes, &pattrs)
		dateStr := pattrs.SortDate[:10]
		title := pattrs.Title
		if title == "" {
			title = "(no title)"
		}

		members, err := s.ListTeamMembers(ctx, plan.ID)
		if err != nil {
			return nil, err
		}

		mp := models.MusicMonthPlan{
			Date:   dateStr,
			PlanID: plan.ID,
			Title:  title,
		}

		for _, tm := range members {
			if !musicPositions[tm.Attrs.TeamPositionName] {
				continue
			}

			if tm.Attrs.TeamPositionName == "Music Lead" {
				mp.MusicLead = &tm
			} else {
				mp.BandMembers = append(mp.BandMembers, tm)
			}

			name := tm.Attrs.Name
			if _, ok := appearanceCounts[name]; !ok {
				appearanceCounts[name] = make(map[string]bool)
			}
			appearanceCounts[name][dateStr] = true
		}

		// Sort band members by position
		sort.Slice(mp.BandMembers, func(i, j int) bool {
			return mp.BandMembers[i].Attrs.TeamPositionName < mp.BandMembers[j].Attrs.TeamPositionName
		})

		// Check blockouts
		checkDate, _ := time.Parse("2006-01-02", dateStr)
		for _, m := range bandMembers {
			for _, b := range blockoutsByPerson[m.PersonID] {
				start, _ := time.Parse("2006-01-02", b.Attrs.StartsAt[:10])
				end, _ := time.Parse("2006-01-02", b.Attrs.EndsAt[:10])
				if !checkDate.Before(start) && !checkDate.After(end) {
					entry := m.Name
					if b.Attrs.Reason != "" {
						entry += " (" + b.Attrs.Reason + ")"
					}
					mp.BlockedNames = append(mp.BlockedNames, entry)
					break
				}
			}
		}

		resultPlans = append(resultPlans, mp)
	}

	// Flatten appearance counts
	counts := make(map[string]int)
	for name, dates := range appearanceCounts {
		counts[name] = len(dates)
	}

	return &models.MusicMonth{
		YearMonth:        yearMonth,
		Plans:            resultPlans,
		AppearanceCounts: counts,
	}, nil
}
