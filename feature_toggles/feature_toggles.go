package feature_toggles

import (
	"encoding/json"
	"fmt"
	"os"

	gofeatureflag "github.com/thomaspoignant/go-feature-flag"
	"github.com/thomaspoignant/go-feature-flag/ffcontext"
	"github.com/thomaspoignant/go-feature-flag/retriever/fileretriever"
)

type FeatureService struct {
	client    *gofeatureflag.GoFeatureFlag
	userID    string
	filePath  string
	retrieved bool
}

func NewFeatureService(filePath string, userID string) (*FeatureService, error) {
	fs := &FeatureService{
		userID:   userID,
		filePath: filePath,
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("features file not found: %s", filePath)
	}

	client, err := gofeatureflag.New(gofeatureflag.Config{
		PollingInterval: 3,
		Retriever: &fileretriever.Retriever{
			Path: filePath,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize feature flag client: %w", err)
	}

	fs.client = client
	fs.retrieved = true
	return fs, nil
}

func (fs *FeatureService) Close() error {
	if fs.client != nil {
		fs.client.Close()
	}
	return nil
}

func (fs *FeatureService) getUserContext() ffcontext.Context {
	ctx := ffcontext.NewEvaluationContext(fs.userID)
	return ctx
}

func (fs *FeatureService) BooleanFeature(featureName string, defaultValue bool) bool {
	if !fs.retrieved || fs.client == nil {
		return defaultValue
	}

	ctx := fs.getUserContext()
	value, err := fs.client.BoolVariation(featureName, ctx, defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

func (fs *FeatureService) StringFeature(featureName string, defaultValue string) string {
	if !fs.retrieved || fs.client == nil {
		return defaultValue
	}

	ctx := fs.getUserContext()
	value, err := fs.client.StringVariation(featureName, ctx, defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

func (fs *FeatureService) IntFeature(featureName string, defaultValue int) int {
	if !fs.retrieved || fs.client == nil {
		return defaultValue
	}

	ctx := fs.getUserContext()
	value, err := fs.client.IntVariation(featureName, ctx, defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

func (fs *FeatureService) FloatFeature(featureName string, defaultValue float64) float64 {
	if !fs.retrieved || fs.client == nil {
		return defaultValue
	}

	ctx := fs.getUserContext()
	value, err := fs.client.Float64Variation(featureName, ctx, defaultValue)
	if err != nil {
		return defaultValue
	}
	return value
}

func (fs *FeatureService) JSONFeature(featureName string, defaultValue interface{}, valuePointer interface{}) error {
	if !fs.retrieved || fs.client == nil {
		return fmt.Errorf("feature service not ready")
	}

	ctx := fs.getUserContext()
	result, err := fs.client.JSONVariation(featureName, ctx, nil)
	if err != nil {
		if defaultValue != nil {
			defaultJSON, err := json.Marshal(defaultValue)
			if err != nil {
				return fmt.Errorf("feature flag and default value failed: %w", err)
			}
			return json.Unmarshal(defaultJSON, valuePointer)
		}
		return err
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal feature flag value: %w", err)
	}

	return json.Unmarshal(resultJSON, valuePointer)
}
