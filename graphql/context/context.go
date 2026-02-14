package context

import (
	"context"
)

type ContextKey string

const (
	UserIDKey ContextKey = "user_id"
	RequestIDKey ContextKey = "request_id"
	UserRoleKey ContextKey = "user_role"
	TenantIDKey ContextKey = "tenant_id"
)

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

func MustGetUserID(ctx context.Context) string {
	userID, ok := GetUserID(ctx)
	if !ok {
		panic("user ID not found in context")
	}
	return userID
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(RequestIDKey).(string)
	return requestID, ok
}

func WithUserRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, UserRoleKey, role)
}

func GetUserRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(UserRoleKey).(string)
	return role, ok
}

func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, TenantIDKey, tenantID)
}

func GetTenantID(ctx context.Context) (string, bool) {
	tenantID, ok := ctx.Value(TenantIDKey).(string)
	return tenantID, ok
}

type UserContext struct {
	UserID   string
	Role     string
	TenantID string
}

func GetUserContext(ctx context.Context) *UserContext {
	uc := &UserContext{}

	if userID, ok := GetUserID(ctx); ok {
		uc.UserID = userID
	}

	if role, ok := GetUserRole(ctx); ok {
		uc.Role = role
	}

	if tenantID, ok := GetTenantID(ctx); ok {
		uc.TenantID = tenantID
	}

	return uc
}

func IsAuthenticated(ctx context.Context) bool {
	_, ok := GetUserID(ctx)
	return ok
}

func HasRole(ctx context.Context, role string) bool {
	userRole, ok := GetUserRole(ctx)
	return ok && userRole == role
}

func HasAnyRole(ctx context.Context, roles ...string) bool {
	userRole, ok := GetUserRole(ctx)
	if !ok {
		return false
	}

	for _, role := range roles {
		if userRole == role {
			return true
		}
	}
	return false
}
