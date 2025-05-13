package badword

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

type BadWordsOptions struct {
	BlackList   func([]string) []string
	Replacement string
	Validate    bool
}

type BadWordsConfig struct {
	BlackList   []string
	Replacement string
	Validate    bool
}

type BadWordsCallback func([]string, int)

var DEFAULT_OPTIONS = BadWordsOptions{
	BlackList:   func(defaultBlacklist []string) []string { return defaultBlacklist },
	Replacement: "*",
	Validate:    false,
}

var DEFAULT_BLACKLIST []string

func NewBadWordsOptions(badWord []string) []string {
	DEFAULT_BLACKLIST = append(DEFAULT_BLACKLIST, badWord...)
	return DEFAULT_BLACKLIST
}

func CreateBadWordConfig(extraConfig interface{}) BadWordsConfig {
	if extraConfig == nil {
		return BadWordsConfig{
			BlackList:   DEFAULT_BLACKLIST,
			Replacement: DEFAULT_OPTIONS.Replacement,
			Validate:    DEFAULT_OPTIONS.Validate,
		}
	}

	if replacement, ok := extraConfig.(string); ok {
		return BadWordsConfig{
			BlackList:   DEFAULT_BLACKLIST,
			Replacement: replacement,
			Validate:    DEFAULT_OPTIONS.Validate,
		}
	}

	if options, ok := extraConfig.(BadWordsOptions); ok {
		customBlackList := DEFAULT_BLACKLIST
		if options.BlackList != nil {
			customBlackList = options.BlackList(DEFAULT_BLACKLIST)
		}

		if len(customBlackList) <= 0 {
			customBlackList = DEFAULT_BLACKLIST
		}

		return BadWordsConfig{
			BlackList:   customBlackList,
			Replacement: getStringWithDefault(options.Replacement, DEFAULT_OPTIONS.Replacement),
			Validate:    options.Validate,
		}
	}

	return BadWordsConfig{
		BlackList:   DEFAULT_BLACKLIST,
		Replacement: DEFAULT_OPTIONS.Replacement,
		Validate:    DEFAULT_OPTIONS.Validate,
	}
}

func getStringWithDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func BadWords(input string, optionsOrReplacement interface{}, callback BadWordsCallback) interface{} {
	if input == "" {
		return errors.New("[vn-badwords] string argument expected")
	}
	config := CreateBadWordConfig(optionsOrReplacement)
	pattern := `(\s|^)(` + strings.Join(config.BlackList, `|`) + `)(\s|$)`
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return err
	}
	input = norm(input)
	badWordsMatched := []string{}
	strReplace := re.ReplaceAllStringFunc(input, func(match string) string {
		badWordsMatched = append(badWordsMatched, match)
		return strings.Repeat(config.Replacement, utf8.RuneCountInString(match))
	})
	if callback != nil {
		callback(badWordsMatched, len(badWordsMatched))
	}
	if config.Validate {
		return re.MatchString(input)
	}
	return strReplace
}

func norm(s string) string {
	return s
}
