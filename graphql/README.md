# Quick Start Guide - GraphQL Utils

## 5 phút đầu tiên

### 1. Cài đặt (30 giây)

```bash
# Configure private repo
export GOPRIVATE=github.com/JIeeiroSst/utils/*

# Install
go get github.com/JIeeiroSst/utils/graphql
```

### 2. Setup Context (2 phút)

**main.go**
```go
package main

import (
    "net/http"
    graphql "github.com/graph-gophers/graphql-go"
    "github.com/graph-gophers/graphql-go/relay"
    gqlcontext "github.com/JIeeiroSst/utils/graphql/context"
)

func main() {
    schema := graphql.MustParseSchema(schemaStr, &resolver{})
    
    http.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        // Add user info
        userID, role := extractFromJWT(r) // Your JWT logic
        ctx = gqlcontext.WithUserID(ctx, userID)
        ctx = gqlcontext.WithUserRole(ctx, role)
        
        relay.Handler{Schema: schema}.ServeHTTP(w, r.WithContext(ctx))
    })
    
    http.ListenAndServe(":8080", nil)
}
```

### 3. Tạo Resolver với Auth (2 phút)

**resolver.go**
```go
package main

import (
    "context"
    "github.com/JIeeiroSst/utils/graphql/resolver"
    "github.com/JIeeiroSst/utils/graphql/errors"
    "github.com/JIeeiroSst/utils/graphql/validation"
)

type Resolver struct {
    *resolver.BaseResolver
}

// Query cần auth
func (r *Resolver) Me(ctx context.Context) (*UserResolver, error) {
    // RequireAuth tự động check & return error nếu chưa login
    uc, err := r.RequireAuth(ctx)
    if err != nil {
        return nil, err
    }
    
    user := getUser(uc.UserID)
    return &UserResolver{user: user}, nil
}

// Mutation với validation
func (r *Resolver) CreateUser(ctx context.Context, args struct{
    Input CreateUserInput
}) (*UserResolver, error) {
    // Validate input
    result := validation.NewValidationResult().
        AddField(validation.NewFieldValidator("email", args.Input.Email).Required().Email()).
        AddField(validation.NewFieldValidator("name", args.Input.Name).Required())
    
    if !result.IsValid() {
        return nil, result.Error()
    }
    
    user := createUser(args.Input)
    return &UserResolver{user: user}, nil
}

// Admin-only mutation
func (r *Resolver) DeleteUser(ctx context.Context, args struct{ ID string }) (bool, error) {
    // RequireRole auto check role
    _, err := r.RequireRole(ctx, "admin")
    if err != nil {
        return false, err // Returns "Forbidden" error
    }
    
    deleteUser(args.ID)
    return true, nil
}
```

### 4. DONE! 🎉

Bạn đã có:
- ✅ Authentication
- ✅ Authorization (role-based)
- ✅ Input validation
- ✅ Chuẩn hóa errors

## Các tính năng thường dùng

### Authentication Check

```go
// Option 1: Tự động return error
uc, err := r.RequireAuth(ctx)

// Option 2: Chỉ check
if gqlcontext.IsAuthenticated(ctx) {
    // ...
}

// Option 3: Get user ID
userID, ok := gqlcontext.GetUserID(ctx)
```

### Role-Based Access

```go
// Check single role
_, err := r.RequireRole(ctx, "admin")

// Check multiple roles
_, err := r.RequireAnyRole(ctx, "admin", "moderator")

// Manual check
if gqlcontext.HasRole(ctx, "admin") {
    // ...
}
```

### Validation

```go
// Single field
fv := validation.NewFieldValidator("email", email).
    Required().
    Email().
    MaxLength(100)

// Multiple fields
result := validation.NewValidationResult().
    AddField(emailValidator).
    AddField(passwordValidator).
    AddField(nameValidator)

if !result.IsValid() {
    return nil, result.Error()
}
```

### Error Handling

```go
// Not found
return nil, errors.NotFound("User")

// Not found with ID
return nil, errors.NotFoundWithID("User", id)

// Unauthorized
return nil, errors.Unauthorized("Token expired")

// Forbidden
return nil, errors.Forbidden("Admin only")

// Validation
return nil, errors.ValidationError("email", "Invalid format")

// Multiple validation errors
return nil, errors.ValidationErrors(map[string]string{
    "email": "Required",
    "password": "Too short",
})
```

### Pagination

```go
func (r *Resolver) Users(ctx context.Context, args struct{
    First *int32
    After *string
}) (*UserConnectionResolver, error) {
    // Validate
    input := &pagination.CursorPaginationInput{
        First: args.First,
        After: args.After,
    }
    pagination.ValidateCursorPagination(input, 100)
    
    // Get data
    users, hasMore := getUsers(input)
    
    // Create connection
    items := toInterfaces(users)
    conn := pagination.NewConnection(items, hasMore, func(item interface{}) string {
        return pagination.EncodeCursor(item.(*User).ID)
    })
    
    return &UserConnectionResolver{conn: conn}, nil
}
```

### DataLoader (giải quyết N+1)

```go
// Setup trong context
func setupContext(r *http.Request) context.Context {
    ctx := r.Context()
    
    loaderMap := loader.NewLoaderMap()
    loaderMap.Register("user", loader.NewLoader(func(ctx context.Context, ids []string) (map[string]interface{}, error) {
        users := db.GetUsersByIDs(ids) // 1 query cho nhiều users
        result := make(map[string]interface{})
        for _, u := range users {
            result[u.ID] = u
        }
        return result, nil
    }))
    
    return context.WithValue(ctx, "loaders", loaderMap)
}

// Use trong resolver
func (pr *PostResolver) Author(ctx context.Context) (*UserResolver, error) {
    loaders := ctx.Value("loaders").(*loader.LoaderMap)
    userLoader, _ := loaders.Get("user")
    
    user, err := userLoader.Load(ctx, pr.post.AuthorID)
    return &UserResolver{user: user.(*User)}, err
}
```

## Common Patterns

### Pattern 1: Protected Query

```go
func (r *Resolver) ProtectedData(ctx context.Context) (*Data, error) {
    uc, err := r.RequireAuth(ctx)
    if err != nil {
        return nil, err
    }
    
    data := getData(uc.UserID)
    return data, nil
}
```

### Pattern 2: Admin-Only Mutation

```go
func (r *Resolver) AdminAction(ctx context.Context, args Args) (bool, error) {
    _, err := r.RequireRole(ctx, "admin")
    if err != nil {
        return false, err
    }
    
    doAdminStuff()
    return true, nil
}
```

### Pattern 3: Ownership Check

```go
func (r *Resolver) UpdatePost(ctx context.Context, args struct{ ID string }) (*Post, error) {
    uc, err := r.RequireAuth(ctx)
    if err != nil {
        return nil, err
    }
    
    post := getPost(args.ID)
    
    // Check if user owns this post
    if err := r.CheckOwnership(ctx, post.AuthorID); err != nil {
        return nil, err
    }
    
    updatePost(post)
    return post, nil
}
```

### Pattern 4: Validated Mutation

```go
func (r *Resolver) CreateItem(ctx context.Context, args struct{ Input Input }) (*Item, error) {
    // Validate
    result := validation.NewValidationResult().
        AddField(validation.NewFieldValidator("name", args.Input.Name).Required().MinLength(3)).
        AddField(validation.NewFieldValidator("price", args.Input.Price).Required().Min(0))
    
    if !result.IsValid() {
        return nil, result.Error()
    }
    
    // Create
    item := createItem(args.Input)
    return item, nil
}
```

### Pattern 5: Paginated List

```go
func (r *Resolver) Items(ctx context.Context, args struct{
    First *int32
    After *string
}) (*ItemConnectionResolver, error) {
    input := &pagination.CursorPaginationInput{First: args.First, After: args.After}
    pagination.ValidateCursorPagination(input, 100)
    
    items, hasMore := getItems(input)
    conn := pagination.NewConnection(toInterfaces(items), hasMore, cursorFn)
    
    return &ItemConnectionResolver{conn: conn}, nil
}
```

## Testing

```go
package main

import (
    "context"
    "testing"
    
    gqlcontext "github.com/JIeeiroSst/utils/graphql/context"
    "github.com/JIeeiroSst/utils/graphql/testing"
)

func TestProtectedQuery(t *testing.T) {
    r := &Resolver{BaseResolver: resolver.NewBaseResolver()}
    
    // Test without auth - should fail
    ctx := context.Background()
    _, err := r.ProtectedData(ctx)
    if err == nil {
        t.Fatal("Expected error for unauthenticated request")
    }
    
    // Test with auth - should succeed
    ctx = gqlcontext.WithUserID(ctx, "user-123")
    data, err := r.ProtectedData(ctx)
    testing.AssertNoError(t, err)
    testing.AssertNotNil(t, data)
}
```

## Next Steps

1. ✅ Đọc [USAGE.md](./USAGE.md) để hiểu rõ hơn
2. ✅ Xem [examples/server.go](./examples/server.go) để học từ ví dụ
3. ✅ Đọc [ARCHITECTURE.md](./ARCHITECTURE.md) để hiểu design patterns
4. ✅ Tham khảo [README.md](./README.md) cho API đầy đủ

## Troubleshooting

### "Cannot find package"
```bash
export GOPRIVATE=github.com/yourcompany/*
go get github.com/JIeeiroSst/utils/graphql
```

### "Unauthorized" errors
```go
// Make sure to add user to context
ctx = gqlcontext.WithUserID(ctx, userID)
ctx = gqlcontext.WithUserRole(ctx, role)
```

### Import errors
```go
// Use alias để tránh conflict
import gqlcontext "github.com/JIeeiroSst/utils/graphql/context"
```

## Cheat Sheet

```go
// Auth
uc, err := r.RequireAuth(ctx)
_, err := r.RequireRole(ctx, "admin")

// Errors
errors.NotFound("User")
errors.Unauthorized("...")
errors.Forbidden("...")
errors.ValidationError("field", "msg")

// Validation
validation.NewFieldValidator("field", value).Required().Email()

// Pagination
pagination.NewConnection(items, hasMore, cursorFn)

// Loader
userLoader.Load(ctx, id)
userLoader.LoadMany(ctx, ids)

// Context
gqlcontext.WithUserID(ctx, id)
gqlcontext.GetUserID(ctx)
gqlcontext.IsAuthenticated(ctx)
gqlcontext.HasRole(ctx, "admin")
```

Happy coding! 🚀
