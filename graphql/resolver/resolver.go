package resolver

import (
	"context"
	"fmt"

	gqlcontext "github.com/JIeeiroSst/utils/graphql/context"
	"github.com/JIeeiroSst/utils/graphql/errors"
)

type BaseResolver struct{}

func NewBaseResolver() *BaseResolver {
	return &BaseResolver{}
}

func (r *BaseResolver) RequireAuth(ctx context.Context) (*gqlcontext.UserContext, error) {
	if !gqlcontext.IsAuthenticated(ctx) {
		return nil, errors.Unauthorized("Authentication required")
	}
	return gqlcontext.GetUserContext(ctx), nil
}

func (r *BaseResolver) RequireRole(ctx context.Context, role string) (*gqlcontext.UserContext, error) {
	uc, err := r.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	if !gqlcontext.HasRole(ctx, role) {
		return nil, errors.Forbidden(fmt.Sprintf("Role %s required", role))
	}

	return uc, nil
}

func (r *BaseResolver) RequireAnyRole(ctx context.Context, roles ...string) (*gqlcontext.UserContext, error) {
	uc, err := r.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	if !gqlcontext.HasAnyRole(ctx, roles...) {
		return nil, errors.Forbidden("Insufficient permissions")
	}

	return uc, nil
}

func (r *BaseResolver) CheckOwnership(ctx context.Context, ownerID string) error {
	uc, err := r.RequireAuth(ctx)
	if err != nil {
		return err
	}

	if uc.UserID != ownerID {
		return errors.Forbidden("You don't have permission to access this resource")
	}

	return nil
}

func (r *BaseResolver) CheckOwnershipOrRole(ctx context.Context, ownerID string, role string) error {
	uc, err := r.RequireAuth(ctx)
	if err != nil {
		return err
	}

	if uc.UserID == ownerID {
		return nil
	}

	if gqlcontext.HasRole(ctx, role) {
		return nil
	}

	return errors.Forbidden("You don't have permission to access this resource")
}

func (r *BaseResolver) GetCurrentUserID(ctx context.Context) (string, error) {
	userID, ok := gqlcontext.GetUserID(ctx)
	if !ok {
		return "", errors.Unauthorized("Authentication required")
	}
	return userID, nil
}

func (r *BaseResolver) GetCurrentTenantID(ctx context.Context) (string, error) {
	tenantID, ok := gqlcontext.GetTenantID(ctx)
	if !ok {
		return "", errors.BadRequest("Tenant ID not found")
	}
	return tenantID, nil
}

func (r *BaseResolver) WrapNotFound(resource string, err error) error {
	if err != nil {
		return errors.NotFound(resource)
	}
	return nil
}

func (r *BaseResolver) WrapNotFoundWithID(resource, id string, err error) error {
	if err != nil {
		return errors.NotFoundWithID(resource, id)
	}
	return nil
}

func (r *BaseResolver) WrapError(err error) error {
	if err == nil {
		return nil
	}
	return errors.WrapError(err)
}

type ResolverFunc func(ctx context.Context) (interface{}, error)

func WithAuth(fn ResolverFunc) ResolverFunc {
	return func(ctx context.Context) (interface{}, error) {
		if !gqlcontext.IsAuthenticated(ctx) {
			return nil, errors.Unauthorized("Authentication required")
		}
		return fn(ctx)
	}
}

func WithRole(role string, fn ResolverFunc) ResolverFunc {
	return func(ctx context.Context) (interface{}, error) {
		if !gqlcontext.IsAuthenticated(ctx) {
			return nil, errors.Unauthorized("Authentication required")
		}
		if !gqlcontext.HasRole(ctx, role) {
			return nil, errors.Forbidden(fmt.Sprintf("Role %s required", role))
		}
		return fn(ctx)
	}
}

func WithAnyRole(roles []string, fn ResolverFunc) ResolverFunc {
	return func(ctx context.Context) (interface{}, error) {
		if !gqlcontext.IsAuthenticated(ctx) {
			return nil, errors.Unauthorized("Authentication required")
		}
		if !gqlcontext.HasAnyRole(ctx, roles...) {
			return nil, errors.Forbidden("Insufficient permissions")
		}
		return fn(ctx)
	}
}

type EntityLoader interface {
	Load(ctx context.Context, id string) (interface{}, error)
	LoadMany(ctx context.Context, ids []string) ([]interface{}, error)
}

type BatchLoader struct {
	LoadFn func(ctx context.Context, ids []string) (map[string]interface{}, error)
}

func (bl *BatchLoader) Load(ctx context.Context, id string) (interface{}, error) {
	results, err := bl.LoadFn(ctx, []string{id})
	if err != nil {
		return nil, err
	}

	if result, ok := results[id]; ok {
		return result, nil
	}

	return nil, fmt.Errorf("entity not found")
}

func (bl *BatchLoader) LoadMany(ctx context.Context, ids []string) ([]interface{}, error) {
	results, err := bl.LoadFn(ctx, ids)
	if err != nil {
		return nil, err
	}

	entities := make([]interface{}, len(ids))
	for i, id := range ids {
		entities[i] = results[id]
	}

	return entities, nil
}
