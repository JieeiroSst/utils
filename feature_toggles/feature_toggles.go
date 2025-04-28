package feature_toggles

import (
	"sync"
)

type FeatureToggle struct {
	name      string
	enabled   bool
	whitelist map[string]bool
	mu        sync.RWMutex
}

type FeatureManager struct {
	toggles map[string]*FeatureToggle
	mu      sync.RWMutex
}

func NewFeatureManager() *FeatureManager {
	return &FeatureManager{
		toggles: make(map[string]*FeatureToggle),
	}
}

func (fm *FeatureManager) CreateFeature(name string, enabled bool) *FeatureToggle {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	toggle := &FeatureToggle{
		name:      name,
		enabled:   enabled,
		whitelist: make(map[string]bool),
	}

	fm.toggles[name] = toggle
	return toggle
}

func (fm *FeatureManager) GetFeature(name string) (*FeatureToggle, bool) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	toggle, ok := fm.toggles[name]
	return toggle, ok
}

func (ft *FeatureToggle) EnableFeature() {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.enabled = true
}

func (ft *FeatureToggle) DisableFeature() {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.enabled = false
}

func (ft *FeatureToggle) IsEnabled(userID string) bool {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	if ft.enabled {
		return true
	}

	return ft.whitelist[userID]
}

func (ft *FeatureToggle) AddToWhitelist(userID string) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.whitelist[userID] = true
}

func (ft *FeatureToggle) RemoveFromWhitelist(userID string) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	delete(ft.whitelist, userID)
}

func (ft *FeatureToggle) AddUsersToWhitelist(userIDs []string) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	for _, userID := range userIDs {
		ft.whitelist[userID] = true
	}
}

func (ft *FeatureToggle) GetWhitelistedUsers() []string {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	users := make([]string, 0, len(ft.whitelist))
	for userID := range ft.whitelist {
		users = append(users, userID)
	}
	return users
}

func (ft *FeatureToggle) IsGloballyEnabled() bool {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	return ft.enabled
}

func (ft *FeatureToggle) Name() string {
	return ft.name
}
