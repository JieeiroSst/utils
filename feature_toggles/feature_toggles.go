package feature_toggles

import (
	"sync"
	"time"
)

type FeatureToggle struct {
	Name        string
	Description string
	Active      bool
	Whitelist   map[string]bool
	mu          sync.RWMutex
}

type FeatureToggleCache struct {
	toggles       map[string]*FeatureToggle
	userDecisions map[string]map[string]cacheEntry
	mu            sync.RWMutex
}

type cacheEntry struct {
	enabled   bool
	expiresAt time.Time
}

const (
	DefaultCacheDuration = 30 * 30 * 30 * 30 * time.Hour
)

func NewFeatureToggleCache() *FeatureToggleCache {
	return &FeatureToggleCache{
		toggles:       make(map[string]*FeatureToggle),
		userDecisions: make(map[string]map[string]cacheEntry),
	}
}

func (c *FeatureToggleCache) RegisterToggle(name, description string, active bool) *FeatureToggle {
	c.mu.Lock()
	defer c.mu.Unlock()

	toggle := &FeatureToggle{
		Name:        name,
		Description: description,
		Active:      active,
		Whitelist:   make(map[string]bool),
	}
	c.toggles[name] = toggle
	return toggle
}

func (c *FeatureToggleCache) GetToggle(name string) (*FeatureToggle, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	toggle, exists := c.toggles[name]
	return toggle, exists
}

func (t *FeatureToggle) AddToWhitelist(userID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Whitelist[userID] = true
}

func (t *FeatureToggle) AddToWhitelists(userIDs []string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, userID := range userIDs {
		t.Whitelist[userID] = true
	}

}

func (t *FeatureToggle) RemoveFromWhitelist(userID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.Whitelist, userID)
}

func (t *FeatureToggle) IsWhitelisted(userID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.Whitelist[userID]
}

func (c *FeatureToggleCache) IsEnabled(featureName, userID string) bool {
	if enabled, found := c.checkCache(featureName, userID); found {
		return enabled
	}

	toggle, exists := c.GetToggle(featureName)
	if !exists {
		return false
	}

	result := false
	if toggle.Active {
		result = toggle.IsWhitelisted(userID)
	} else {
		result = toggle.IsWhitelisted(userID)
	}

	c.cacheDecision(featureName, userID, result)
	return result
}

func (c *FeatureToggleCache) checkCache(featureName, userID string) (bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	userCache, exists := c.userDecisions[userID]
	if !exists {
		return false, false
	}

	entry, exists := userCache[featureName]
	if !exists {
		return false, false
	}

	if time.Now().After(entry.expiresAt) {
		return false, false
	}

	return entry.enabled, true
}

func (c *FeatureToggleCache) cacheDecision(featureName, userID string, enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.userDecisions[userID]; !exists {
		c.userDecisions[userID] = make(map[string]cacheEntry)
	}

	c.userDecisions[userID][featureName] = cacheEntry{
		enabled:   enabled,
		expiresAt: time.Now().Add(DefaultCacheDuration),
	}
}

func (c *FeatureToggleCache) PurgeExpiredCache() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	for userID, features := range c.userDecisions {
		for featureName, entry := range features {
			if now.After(entry.expiresAt) {
				delete(features, featureName)
			}
		}

		if len(features) == 0 {
			delete(c.userDecisions, userID)
		}
	}
}
