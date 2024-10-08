package time_custom

import (
	"time"

	"github.com/JIeeiroSst/utils/logger"
)

var CountryTz = map[string]string{
	"Hungary": "Europe/Budapest",
	"Egypt":   "Africa/Cairo",
}

func TimeIn(country string) time.Time {
	name := CountryTz[country]
	loc, err := time.LoadLocation(name)
	if err != nil {
		logger.ConfigZap().Error(err)
	}
	return time.Now().In(loc)
}
