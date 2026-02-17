package main

import (
	"fmt"
	"log"

	"github.com/JIeeiroSst/utils/security"
)

func main() {
	sqlInjectionExample()
	noSQLInjectionExample()
	commandInjectionExample()
	ldapInjectionExample()
	headerInjectionExample()
	xssExample()
	csrfExample()
	validateInputExample()
}

func sqlInjectionExample() {
	fmt.Println("=== SQL Injection ===")

	inputs := []string{
		"admin' OR '1'='1",
		"1; DROP TABLE users--",
		"' UNION SELECT * FROM passwords--",
		"hello world",
	}

	for _, input := range inputs {
		detected := security.DetectSQLInjection(input)
		fmt.Printf("  Input: %-45s Detected: %v\n", input, detected)
		if detected {
			sanitized := security.SanitizeSQLInput(input)
			fmt.Printf("  Sanitized: %s\n", sanitized)
		}
	}
	fmt.Println()
}

func noSQLInjectionExample() {
	fmt.Println("=== NoSQL Injection ===")

	inputs := []string{
		`{"username": {"$ne": ""}}`,
		`{"$gt": ""}`,
		`db.users.find()`,
		"normal_username",
	}

	for _, input := range inputs {
		detected := security.DetectNoSQLInjection(input)
		fmt.Printf("  Input: %-45s Detected: %v\n", input, detected)
		if detected {
			sanitized := security.SanitizeNoSQLInput(input)
			fmt.Printf("  Sanitized: %s\n", sanitized)
		}
	}
	fmt.Println()
}

func commandInjectionExample() {
	fmt.Println("=== Command Injection ===")

	inputs := []string{
		"file.txt; rm -rf /",
		"test | cat /etc/passwd",
		"hello $(whoami)",
		"document.pdf",
	}

	for _, input := range inputs {
		detected := security.DetectCommandInjection(input)
		fmt.Printf("  Input: %-45s Detected: %v\n", input, detected)
		if detected {
			sanitized := security.SanitizeCommandInput(input)
			fmt.Printf("  Sanitized: %s\n", sanitized)
		}
	}
	fmt.Println()
}

func ldapInjectionExample() {
	fmt.Println("=== LDAP Injection ===")

	inputs := []string{
		"admin)(|(password=*))",
		"user\\name",
		"john.doe",
	}

	for _, input := range inputs {
		detected := security.DetectLDAPInjection(input)
		fmt.Printf("  Input: %-45s Detected: %v\n", input, detected)
		if detected {
			sanitized := security.SanitizeLDAPInput(input)
			fmt.Printf("  Sanitized: %s\n", sanitized)
		}
	}
	fmt.Println()
}

func headerInjectionExample() {
	fmt.Println("=== Header Injection ===")

	inputs := []string{
		"normal-value",
		"value\r\nX-Injected: true",
		"value\nSet-Cookie: malicious=1",
	}

	for _, input := range inputs {
		detected := security.DetectHeaderInjection(input)
		fmt.Printf("  Input: %-45q Detected: %v\n", input, detected)
		if detected {
			sanitized := security.SanitizeHeaderValue(input)
			fmt.Printf("  Sanitized: %q\n", sanitized)
		}
	}
	fmt.Println()
}

func xssExample() {
	fmt.Println("=== XSS Protection ===")

	inputs := []string{
		`<script>alert('xss')</script>`,
		`<img src=x onerror=alert(1)>`,
		`<a href="javascript:alert(1)">click</a>`,
		`<div onmouseover="steal()">hover</div>`,
		`Hello, world!`,
	}

	for _, input := range inputs {
		detected := security.DetectXSS(input)
		fmt.Printf("  Input: %-50s Detected: %v\n", input, detected)
		if detected {
			fmt.Printf("  SanitizeXSS:    %s\n", security.SanitizeXSS(input))
			fmt.Printf("  StripHTMLTags:  %s\n", security.StripHTMLTags(input))
		}
	}

	fmt.Println("\n  --- URL Sanitization ---")
	urls := []string{
		"https://example.com/page?q=hello",
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"/relative/path",
	}
	for _, u := range urls {
		result, err := security.SanitizeURL(u)
		if err != nil {
			fmt.Printf("  URL: %-50s Blocked: %v\n", u, err)
		} else {
			fmt.Printf("  URL: %-50s OK: %s\n", u, result)
		}
	}
	fmt.Println()
}

func csrfExample() {
	fmt.Println("=== CSRF Protection ===")

	token, err := security.GenerateCSRFToken()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Generated token: %s\n", token)
	fmt.Printf("  Valid (correct):  %v\n", security.ValidateCSRFToken(token, token))
	fmt.Printf("  Valid (wrong):    %v\n", security.ValidateCSRFToken(token, "wrong-token"))

	sessionID := "session-abc-123"
	secret := "my-secret-key"

	hmacToken := security.GenerateCSRFTokenWithHMAC(sessionID, secret)
	fmt.Printf("\n  HMAC token:       %s\n", hmacToken)
	fmt.Printf("  Valid (correct):  %v\n", security.ValidateCSRFTokenWithHMAC(hmacToken, sessionID, secret))
	fmt.Printf("  Valid (wrong id): %v\n", security.ValidateCSRFTokenWithHMAC(hmacToken, "other-session", secret))
	fmt.Println()
}

func validateInputExample() {
	fmt.Println("=== ValidateInput (combined checks) ===")

	inputs := []string{
		"normal input text",
		"admin' OR '1'='1",
		`<script>alert('xss')</script>`,
		"file.txt; rm -rf /",
		`{"$ne": ""}`,
	}

	for _, input := range inputs {
		result := security.ValidateInput(input, security.WithAllChecks())
		fmt.Printf("  Input: %-45s Valid: %v\n", input, result.IsValid)
		for _, t := range result.Threats {
			fmt.Printf("    -> [%s] %s\n", t.Type, t.Detail)
		}
	}

	fmt.Println("\n  --- Selective checks (SQL + XSS only) ---")
	input := "admin' OR '1'='1; <script>alert(1)</script>"
	result := security.ValidateInput(input, security.WithSQLCheck(), security.WithXSSCheck())
	fmt.Printf("  Input: %s\n", input)
	fmt.Printf("  Valid: %v\n", result.IsValid)
	for _, t := range result.Threats {
		fmt.Printf("    -> [%s] %s\n", t.Type, t.Detail)
	}
}
