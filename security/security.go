package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"html"
	"net/url"
	"regexp"
	"strings"
)

type ThreatType string

const (
	SQLInjection     ThreatType = "SQL_INJECTION"
	NoSQLInjection   ThreatType = "NOSQL_INJECTION"
	CommandInjection ThreatType = "COMMAND_INJECTION"
	LDAPInjection    ThreatType = "LDAP_INJECTION"
	HeaderInjection  ThreatType = "HEADER_INJECTION"
	XSS              ThreatType = "XSS"
)

type ThreatInfo struct {
	Type   ThreatType
	Detail string
}

type ValidationResult struct {
	IsValid bool
	Threats []ThreatInfo
}

type ValidationOption func(*validationConfig)

type validationConfig struct {
	sql     bool
	noSQL   bool
	command bool
	ldap    bool
	header  bool
	xss     bool
}

func WithSQLCheck() ValidationOption     { return func(c *validationConfig) { c.sql = true } }
func WithNoSQLCheck() ValidationOption   { return func(c *validationConfig) { c.noSQL = true } }
func WithCommandCheck() ValidationOption { return func(c *validationConfig) { c.command = true } }
func WithLDAPCheck() ValidationOption    { return func(c *validationConfig) { c.ldap = true } }
func WithHeaderCheck() ValidationOption  { return func(c *validationConfig) { c.header = true } }
func WithXSSCheck() ValidationOption     { return func(c *validationConfig) { c.xss = true } }
func WithAllChecks() ValidationOption {
	return func(c *validationConfig) {
		c.sql = true
		c.noSQL = true
		c.command = true
		c.ldap = true
		c.header = true
		c.xss = true
	}
}

var (
	sqlPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(\b(union\s+(all\s+)?select|select\s+.+\s+from|insert\s+into|update\s+.+\s+set|delete\s+from|drop\s+(table|database)|alter\s+table|create\s+table|truncate\s+table)\b)`),
		regexp.MustCompile(`(?i)(\bor\b\s+[\d'"]+=\s*[\d'"]+)`),
		regexp.MustCompile(`(?i)(\band\b\s+[\d'"]+=\s*[\d'"]+)`),
		regexp.MustCompile(`(?i)(;?\s*--)`),
		regexp.MustCompile(`(?i)(/\*.*\*/)`),
		regexp.MustCompile(`(?i)(\bexec(\s+|\s*\()\s*\w)`),
		regexp.MustCompile(`(?i)(\bwaitfor\s+delay\b)`),
		regexp.MustCompile(`(?i)(\bbenchmark\s*\()`),
		regexp.MustCompile(`(?i)(\bsleep\s*\()`),
		regexp.MustCompile(`(?i)(\bload_file\s*\()`),
		regexp.MustCompile(`(?i)(\binto\s+(out|dump)file\b)`),
		regexp.MustCompile(`(?i)(';\s*\w)`),
	}

	noSQLPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\$(?:gt|gte|lt|lte|ne|eq|in|nin|and|or|not|nor|exists|type|regex|where|all|elemMatch|size|mod|slice|meta|text|search|nearSphere|near|geoWithin|geoIntersects|maxDistance|minDistance)`),
		regexp.MustCompile(`(?i)\{\s*["\']?\$`),
		regexp.MustCompile(`(?i)this\s*\.\s*\w+\s*==`),
		regexp.MustCompile(`(?i)function\s*\(`),
		regexp.MustCompile(`(?i)db\.\w+\.\w+\(`),
		regexp.MustCompile(`(?i)mapReduce|group\s*\(`),
	}

	commandPatterns = []*regexp.Regexp{
		regexp.MustCompile("(;|\\||&&|\\|\\||`|\\$\\()"),
		regexp.MustCompile(`(?i)\b(cat|ls|rm|mv|cp|chmod|chown|wget|curl|bash|sh|zsh|python|perl|ruby|nc|netcat|nmap|whoami|id|uname|passwd|sudo|su|kill|pkill)\b`),
		regexp.MustCompile(`>\s*/`),
		regexp.MustCompile(`<\s*/`),
		regexp.MustCompile(`\.\./`),
	}

	ldapSpecialChars = []string{"\\", "*", "(", ")", "\x00", "/"}

	xssPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<\s*script[\s>]`),
		regexp.MustCompile(`(?i)<\s*/\s*script\s*>`),
		regexp.MustCompile(`(?i)<\s*iframe[\s>]`),
		regexp.MustCompile(`(?i)<\s*object[\s>]`),
		regexp.MustCompile(`(?i)<\s*embed[\s>]`),
		regexp.MustCompile(`(?i)<\s*applet[\s>]`),
		regexp.MustCompile(`(?i)<\s*form[\s>]`),
		regexp.MustCompile(`(?i)<\s*img[^>]+\bon\w+\s*=`),
		regexp.MustCompile(`(?i)<\s*svg[^>]*\bon\w+\s*=`),
		regexp.MustCompile(`(?i)<\s*body[^>]*\bon\w+\s*=`),
		regexp.MustCompile(`(?i)<\s*\w+[^>]*\bon(load|error|click|mouseover|mouseout|focus|blur|submit|change|input|keydown|keyup|keypress|dblclick|contextmenu|abort|unload|resize|scroll)\s*=`),
		regexp.MustCompile(`(?i)javascript\s*:`),
		regexp.MustCompile(`(?i)vbscript\s*:`),
		regexp.MustCompile(`(?i)data\s*:\s*text/html`),
		regexp.MustCompile(`(?i)expression\s*\(`),
		regexp.MustCompile(`(?i)url\s*\(\s*['"]?\s*javascript`),
	}

	htmlTagPattern = regexp.MustCompile(`<[^>]*>`)
)

func DetectSQLInjection(input string) bool {
	for _, p := range sqlPatterns {
		if p.MatchString(input) {
			return true
		}
	}
	return false
}

func SanitizeSQLInput(input string) string {
	replacer := strings.NewReplacer(
		"'", "''",
		"\\", "\\\\",
		"\x00", "",
		"\n", "\\n",
		"\r", "\\r",
		"\x1a", "\\Z",
	)
	return replacer.Replace(input)
}

func DetectNoSQLInjection(input string) bool {
	for _, p := range noSQLPatterns {
		if p.MatchString(input) {
			return true
		}
	}
	return false
}

func SanitizeNoSQLInput(input string) string {
	result := strings.ReplaceAll(input, "$", "")
	result = strings.ReplaceAll(result, "{", "")
	result = strings.ReplaceAll(result, "}", "")
	return result
}

func DetectCommandInjection(input string) bool {
	for _, p := range commandPatterns {
		if p.MatchString(input) {
			return true
		}
	}
	return false
}

func SanitizeCommandInput(input string) string {
	dangerous := []string{";", "|", "&", "`", "$", "(", ")", "{", "}", "<", ">", "!", "#", "~", "\n", "\r"}
	result := input
	for _, ch := range dangerous {
		result = strings.ReplaceAll(result, ch, "")
	}
	return result
}

func DetectLDAPInjection(input string) bool {
	for _, ch := range ldapSpecialChars {
		if strings.Contains(input, ch) {
			return true
		}
	}
	return false
}

func SanitizeLDAPInput(input string) string {
	var b strings.Builder
	b.Grow(len(input))
	for _, r := range input {
		switch r {
		case '\\':
			b.WriteString("\\5c")
		case '*':
			b.WriteString("\\2a")
		case '(':
			b.WriteString("\\28")
		case ')':
			b.WriteString("\\29")
		case '\x00':
			b.WriteString("\\00")
		case '/':
			b.WriteString("\\2f")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func DetectHeaderInjection(input string) bool {
	return strings.ContainsAny(input, "\r\n")
}

func SanitizeHeaderValue(input string) string {
	result := strings.ReplaceAll(input, "\r", "")
	result = strings.ReplaceAll(result, "\n", "")
	return result
}

func DetectXSS(input string) bool {
	for _, p := range xssPatterns {
		if p.MatchString(input) {
			return true
		}
	}
	return false
}

func SanitizeXSS(input string) string {
	result := html.EscapeString(input)
	result = strings.ReplaceAll(result, "`", "&#96;")
	return result
}

func StripHTMLTags(input string) string {
	return htmlTagPattern.ReplaceAllString(input, "")
}

func SanitizeURL(input string) (string, error) {
	trimmed := strings.TrimSpace(input)

	lower := strings.ToLower(trimmed)
	dangerousSchemes := []string{"javascript:", "vbscript:", "data:text/html", "data:application"}
	for _, scheme := range dangerousSchemes {
		if strings.HasPrefix(lower, scheme) {
			return "", &SecurityError{Message: "dangerous URL scheme detected: " + scheme}
		}
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", &SecurityError{Message: "invalid URL: " + err.Error()}
	}

	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" && parsed.Scheme != "mailto" {
		return "", &SecurityError{Message: "disallowed URL scheme: " + parsed.Scheme}
	}

	return parsed.String(), nil
}

func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func ValidateCSRFToken(token, expected string) bool {
	if len(token) == 0 || len(expected) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

func GenerateCSRFTokenWithHMAC(sessionID, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sessionID))
	return hex.EncodeToString(mac.Sum(nil))
}

func ValidateCSRFTokenWithHMAC(token, sessionID, secret string) bool {
	expected := GenerateCSRFTokenWithHMAC(sessionID, secret)
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}


func ValidateInput(input string, opts ...ValidationOption) ValidationResult {
	cfg := &validationConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	result := ValidationResult{IsValid: true}

	if cfg.sql && DetectSQLInjection(input) {
		result.IsValid = false
		result.Threats = append(result.Threats, ThreatInfo{
			Type:   SQLInjection,
			Detail: "potential SQL injection pattern detected",
		})
	}

	if cfg.noSQL && DetectNoSQLInjection(input) {
		result.IsValid = false
		result.Threats = append(result.Threats, ThreatInfo{
			Type:   NoSQLInjection,
			Detail: "potential NoSQL injection pattern detected",
		})
	}

	if cfg.command && DetectCommandInjection(input) {
		result.IsValid = false
		result.Threats = append(result.Threats, ThreatInfo{
			Type:   CommandInjection,
			Detail: "potential OS command injection pattern detected",
		})
	}

	if cfg.ldap && DetectLDAPInjection(input) {
		result.IsValid = false
		result.Threats = append(result.Threats, ThreatInfo{
			Type:   LDAPInjection,
			Detail: "potential LDAP injection pattern detected",
		})
	}

	if cfg.header && DetectHeaderInjection(input) {
		result.IsValid = false
		result.Threats = append(result.Threats, ThreatInfo{
			Type:   HeaderInjection,
			Detail: "potential header (CRLF) injection detected",
		})
	}

	if cfg.xss && DetectXSS(input) {
		result.IsValid = false
		result.Threats = append(result.Threats, ThreatInfo{
			Type:   XSS,
			Detail: "potential XSS pattern detected",
		})
	}

	return result
}

type SecurityError struct {
	Message string
}

func (e *SecurityError) Error() string {
	return e.Message
}
