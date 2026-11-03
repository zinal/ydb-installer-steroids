// Package session provides an in-memory store for the single active installer
// session. The installer is a local single-user tool, so everything is kept
// in memory and protected with a single sync.RWMutex. On page reload the
// frontend calls GET /api/session and rebuilds its UI from the backend state.
package session

import (
	"encoding/json"
	"sync"

	"ydb-installer/internal/models"
	"ydb-installer/internal/version"
)

var Flows = map[models.Flow][]string{
	models.FlowYDB:       {"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
	models.FlowDeleteYDB: {"1", "30", "31"},
}

// FlowSteps returns the canonical list of step IDs for a flow, or {"1"} for
// an unknown / empty flow.
func FlowSteps(f models.Flow) []string {
	if steps, ok := Flows[f]; ok {
		out := make([]string, len(steps))
		copy(out, steps)
		return out
	}
	return []string{"1"}
}

type Store struct {
	mu      sync.RWMutex
	session models.Session
}

func New(defaultLang string) *Store {
	steps := FlowSteps(models.FlowNone)
	stepObjs := make([]models.Step, len(steps))
	for i, id := range steps {
		status := models.StatusDisabled
		if i == 0 {
			status = models.StatusActive
		}
		stepObjs[i] = models.Step{ID: id, Status: status}
	}
	return &Store{
		session: models.Session{
			Flow:        models.FlowNone,
			CurrentStep: "1",
			Steps:       stepObjs,
			DefaultLang: defaultLang,
			Version:     version.Version,
			UIState: &models.UIState{
				CurrentStep: "1",
				Visited:     []string{"1"},
				Errored:     []string{},
			},
		},
	}
}

func (s *Store) Get() models.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSession(s.session)
}

// Update applies the non-nil fields of u to the stored session. Switching the
// flow rebuilds the step list from the canonical flow definition, preserving
// the "first step active, rest disabled" layout the sidebar expects.
func (s *Store) Update(u models.SessionUpdate) models.Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	if u.Flow != nil && *u.Flow != s.session.Flow {
		s.session.Flow = *u.Flow
		ids := FlowSteps(*u.Flow)
		s.session.Steps = make([]models.Step, len(ids))
		for i, id := range ids {
			status := models.StatusDisabled
			if i == 0 {
				status = models.StatusActive
			}
			s.session.Steps[i] = models.Step{ID: id, Status: status}
		}
		if len(ids) > 0 {
			s.session.CurrentStep = ids[0]
		}
	}

	if u.Steps != nil {
		s.session.Steps = make([]models.Step, len(*u.Steps))
		copy(s.session.Steps, *u.Steps)
	}

	if u.CurrentStep != nil {
		s.session.CurrentStep = *u.CurrentStep
	}

	if u.UIState != nil {
		visited := make([]string, len(u.UIState.Visited))
		copy(visited, u.UIState.Visited)
		errored := make([]string, len(u.UIState.Errored))
		copy(errored, u.UIState.Errored)
		s.session.UIState = &models.UIState{
			CurrentStep:     u.UIState.CurrentStep,
			Visited:         visited,
			Errored:         errored,
			ConfigGenerated: u.UIState.ConfigGenerated,
		}
	}

	return cloneSession(s.session)
}

// Reset returns the store to its initial post-New state, keeping the
// previously detected default language.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	lang := s.session.DefaultLang
	s.session = New(lang).session
}

func cloneSession(in models.Session) models.Session {
	out := in
	out.Steps = make([]models.Step, len(in.Steps))
	copy(out.Steps, in.Steps)
	if in.Inventory != nil {
		invBytes, _ := json.Marshal(in.Inventory)
		out.Inventory = &models.InventoryData{}
		_ = json.Unmarshal(invBytes, out.Inventory)
	}
	if in.UIState != nil {
		visited := make([]string, len(in.UIState.Visited))
		copy(visited, in.UIState.Visited)
		errored := make([]string, len(in.UIState.Errored))
		copy(errored, in.UIState.Errored)
		out.UIState = &models.UIState{
			CurrentStep:     in.UIState.CurrentStep,
			Visited:         visited,
			Errored:         errored,
			ConfigGenerated: in.UIState.ConfigGenerated,
		}
	}
	return out
}

func (s *Store) SetInventory(data *models.InventoryData, invFound, cfgFound bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.session.Inventory = data
	s.session.InventoryFound = invFound
	s.session.ConfigFound = cfgFound
}

func (s *Store) GetInventory() *models.InventoryData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.session.Inventory == nil {
		return nil
	}
	invBytes, _ := json.Marshal(s.session.Inventory)
	out := &models.InventoryData{}
	_ = json.Unmarshal(invBytes, out)
	return out
}

func (s *Store) UpdateInventory(data *models.InventoryData, invFound, cfgFound bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if data != nil {
		s.session.Inventory = data
	}
	s.session.InventoryFound = invFound
	s.session.ConfigFound = cfgFound
}

func (s *Store) SetInventoryFound(found bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.session.InventoryFound = found
}

func (s *Store) SetConfigFound(found bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.session.ConfigFound = found
}
