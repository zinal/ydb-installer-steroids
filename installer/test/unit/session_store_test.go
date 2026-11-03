package unit_test

import (
	"sync"
	"testing"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/session"
)

func TestNew_FreshSession(t *testing.T) {
	s := session.New("ru")
	got := s.Get()
	if got.DefaultLang != "ru" {
		t.Errorf("default_lang = %q, want ru", got.DefaultLang)
	}
	if got.Flow != models.FlowNone {
		t.Errorf("flow = %q, want empty", got.Flow)
	}
	if got.CurrentStep != "1" {
		t.Errorf("current_step = %q, want 1", got.CurrentStep)
	}
	if len(got.Steps) == 0 || got.Steps[0].Status != models.StatusActive {
		t.Errorf("expected first step active, got %+v", got.Steps)
	}
}

func TestUpdate_FlowSwitchRebuildsSteps(t *testing.T) {
	s := session.New("en")
	flow := models.FlowYDB
	got := s.Update(models.SessionUpdate{Flow: &flow})

	wantSteps := session.FlowSteps(models.FlowYDB)
	if len(got.Steps) != len(wantSteps) {
		t.Fatalf("steps len = %d, want %d", len(got.Steps), len(wantSteps))
	}
	for i, want := range wantSteps {
		if got.Steps[i].ID != want {
			t.Errorf("step[%d].id = %q, want %q", i, got.Steps[i].ID, want)
		}
	}
	if got.Steps[0].Status != models.StatusActive {
		t.Errorf("first step should be active, got %q", got.Steps[0].Status)
	}
	for _, st := range got.Steps[1:] {
		if st.Status != models.StatusDisabled {
			t.Errorf("step %s should be disabled after fresh flow switch, got %q", st.ID, st.Status)
		}
	}
}

func TestUpdate_StepsAndCurrentStep(t *testing.T) {
	s := session.New("en")
	flow := models.FlowYDB
	s.Update(models.SessionUpdate{Flow: &flow})

	cur := "3"
	newSteps := []models.Step{
		{ID: "1", Status: models.StatusDone},
		{ID: "2", Status: models.StatusDone},
		{ID: "3", Status: models.StatusActive},
		{ID: "4", Status: models.StatusClickable},
	}
	got := s.Update(models.SessionUpdate{
		CurrentStep: &cur,
		Steps:       &newSteps,
	})
	if got.CurrentStep != "3" {
		t.Errorf("current_step = %q, want 3", got.CurrentStep)
	}
	if len(got.Steps) != 4 || got.Steps[2].Status != models.StatusActive {
		t.Errorf("unexpected steps: %+v", got.Steps)
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	s := session.New("en")
	flow := models.FlowYDB
	s.Update(models.SessionUpdate{Flow: &flow})

	const goroutines = 16
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if (i+j)%2 == 0 {
					_ = s.Get()
				} else {
					step := "1"
					s.Update(models.SessionUpdate{CurrentStep: &step})
				}
			}
		}()
	}
	wg.Wait()

	final := s.Get()
	if final.Flow != models.FlowYDB {
		t.Errorf("expected flow preserved, got %q", final.Flow)
	}
}

func TestReset_RestoresDefaults(t *testing.T) {
	s := session.New("ru")
	flow := models.FlowYDB
	s.Update(models.SessionUpdate{Flow: &flow})

	s.Reset()
	got := s.Get()
	if got.Flow != models.FlowNone {
		t.Errorf("flow after reset = %q, want empty", got.Flow)
	}
	if got.DefaultLang != "ru" {
		t.Errorf("default_lang after reset should be preserved, got %q", got.DefaultLang)
	}
}

func TestFlowSteps_KnownAndUnknown(t *testing.T) {
	if got := session.FlowSteps(models.FlowYDB); len(got) != 11 {
		t.Errorf("FlowYDB should have 11 steps, got %d", len(got))
	}
	if got := session.FlowSteps(models.FlowDeleteYDB); len(got) != 3 {
		t.Errorf("FlowDeleteYDB should have 3 steps, got %d", len(got))
	}
	if got := session.FlowSteps("nonexistent"); len(got) != 1 || got[0] != "1" {
		t.Errorf("unknown flow → expected just [1], got %v", got)
	}
}

func TestNew_UIStateInitialized(t *testing.T) {
	s := session.New("en")
	got := s.Get()
	if got.UIState == nil {
		t.Fatal("expected UIState to be initialized on fresh session")
	}
	if got.UIState.CurrentStep != "1" {
		t.Errorf("UIState.CurrentStep = %q, want 1", got.UIState.CurrentStep)
	}
	if len(got.UIState.Visited) != 1 || got.UIState.Visited[0] != "1" {
		t.Errorf("UIState.Visited = %v, want [1]", got.UIState.Visited)
	}
	if len(got.UIState.Errored) != 0 {
		t.Errorf("UIState.Errored = %v, want empty", got.UIState.Errored)
	}
	if got.UIState.ShowEMStep || got.UIState.ConfigGenerated {
		t.Errorf("UIState flags should default to false, got %+v", got.UIState)
	}
}

func TestUpdate_UIStatePersists(t *testing.T) {
	s := session.New("en")
	upd := models.SessionUpdate{
		UIState: &models.UIState{
			CurrentStep:     "6",
			Visited:         []string{"1", "2", "3", "4", "5", "6"},
			Errored:         []string{"4"},
			ShowEMStep:      true,
			ConfigGenerated: true,
		},
	}
	got := s.Update(upd)
	if got.UIState == nil {
		t.Fatal("expected UIState to be persisted")
	}
	if got.UIState.CurrentStep != "6" {
		t.Errorf("UIState.CurrentStep = %q, want 6", got.UIState.CurrentStep)
	}
	if len(got.UIState.Visited) != 6 {
		t.Errorf("UIState.Visited len = %d, want 6", len(got.UIState.Visited))
	}
	if len(got.UIState.Errored) != 1 || got.UIState.Errored[0] != "4" {
		t.Errorf("UIState.Errored = %v, want [4]", got.UIState.Errored)
	}
	if !got.UIState.ShowEMStep || !got.UIState.ConfigGenerated {
		t.Errorf("UIState flags not persisted, got %+v", got.UIState)
	}

	// Verify it survives a subsequent Get() (i.e. actually stored, not just returned).
	again := s.Get()
	if again.UIState.CurrentStep != "6" || !again.UIState.ShowEMStep {
		t.Errorf("UIState not retained across Get(), got %+v", again.UIState)
	}
}

func TestUpdate_UIStateNilLeavesExistingUnchanged(t *testing.T) {
	s := session.New("en")
	s.Update(models.SessionUpdate{
		UIState: &models.UIState{CurrentStep: "3", Visited: []string{"1", "2", "3"}},
	})

	// A subsequent update with UIState == nil (e.g. only Flow changes) must not
	// wipe out the previously stored UIState.
	flow := models.FlowYDB
	got := s.Update(models.SessionUpdate{Flow: &flow})
	if got.UIState == nil || got.UIState.CurrentStep != "3" {
		t.Errorf("UIState should be preserved when update omits it, got %+v", got.UIState)
	}
}

func TestClone_UIStateIsDeepCopy(t *testing.T) {
	s := session.New("en")
	s.Update(models.SessionUpdate{
		UIState: &models.UIState{CurrentStep: "2", Visited: []string{"1", "2"}},
	})

	got := s.Get()
	// Mutate the returned slice — must not affect the store's internal state.
	got.UIState.Visited[0] = "mutated"

	again := s.Get()
	if again.UIState.Visited[0] == "mutated" {
		t.Error("UIState.Visited was not deep-copied — mutation leaked into store")
	}
}

func TestReset_ClearsUIState(t *testing.T) {
	s := session.New("ru")
	s.Update(models.SessionUpdate{
		UIState: &models.UIState{
			CurrentStep: "8",
			Visited:     []string{"1", "2", "3", "4", "5", "6", "7", "8"},
			ShowEMStep:  true,
		},
	})

	s.Reset()
	got := s.Get()
	if got.UIState == nil {
		t.Fatal("expected UIState to be reinitialized after reset")
	}
	if got.UIState.CurrentStep != "1" {
		t.Errorf("UIState.CurrentStep after reset = %q, want 1", got.UIState.CurrentStep)
	}
	if len(got.UIState.Visited) != 1 || got.UIState.Visited[0] != "1" {
		t.Errorf("UIState.Visited after reset = %v, want [1]", got.UIState.Visited)
	}
	if got.UIState.ShowEMStep {
		t.Error("UIState.ShowEMStep should be false after reset")
	}
}
