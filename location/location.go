package location

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/JIeeiroSst/utils/logger"
	"go.uber.org/zap"
)

type IPInfo struct {
	IP       string `json:"ip"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Timezone string `json:"timezone"`
}

var (
	IpInfoHost = "https://ipinfo.io/json"
)

func GetLocation(ctx context.Context) *IPInfo {
	resp, err := http.Get(IpInfoHost)
	if err != nil {
		logger.WithContext(ctx).Error("GetLocation", zap.Error(err))
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.WithContext(ctx).Error("Error reading response:", zap.Error(err))
		return nil
	}

	var ipInfo IPInfo
	if err := json.Unmarshal(body, &ipInfo); err != nil {
		logger.WithContext(ctx).Error("Error parsing JSON:", zap.Error(err))
		return nil
	}

	return &ipInfo
}
