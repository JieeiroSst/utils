package testing

import (
	"context"
	"testing"

	gqlcontext "github.com/JIeeiroSst/utils/graphql/context"
)

func MockContext() context.Context {
	return context.Background()
}

func MockAuthContext(userID, role string) context.Context {
	ctx := context.Background()
	ctx = gqlcontext.WithUserID(ctx, userID)
	ctx = gqlcontext.WithUserRole(ctx, role)
	return ctx
}

func MockTenantContext(userID, role, tenantID string) context.Context {
	ctx := MockAuthContext(userID, role)
	ctx = gqlcontext.WithTenantID(ctx, tenantID)
	return ctx
}

func AssertError(t *testing.T, err error, expectedType string) {
	t.Helper()

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

}

func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func AssertEqual(t *testing.T, got, want interface{}) {
	t.Helper()

	if got != want {
		t.Fatalf("Got %v, want %v", got, want)
	}
}

func AssertNotNil(t *testing.T, value interface{}) {
	t.Helper()

	if value == nil {
		t.Fatal("Expected non-nil value, got nil")
	}
}

func AssertNil(t *testing.T, value interface{}) {
	t.Helper()

	if value != nil {
		t.Fatalf("Expected nil, got: %v", value)
	}
}

func AssertContains(t *testing.T, str, substr string) {
	t.Helper()

	if !contains(str, substr) {
		t.Fatalf("Expected %q to contain %q", str, substr)
	}
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(substr) == 0 || findSubstring(str, substr))
}

func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type TestHelper struct {
	T *testing.T
}

func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{T: t}
}

func (h *TestHelper) AssertError(err error, expectedType string) {
	AssertError(h.T, err, expectedType)
}

func (h *TestHelper) AssertNoError(err error) {
	AssertNoError(h.T, err)
}

func (h *TestHelper) AssertEqual(got, want interface{}) {
	AssertEqual(h.T, got, want)
}
