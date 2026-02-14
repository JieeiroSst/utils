package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/JIeeiroSst/utils/graphql/errors"
	"github.com/JIeeiroSst/utils/graphql/loader"
	"github.com/JIeeiroSst/utils/graphql/middleware"
	"github.com/JIeeiroSst/utils/graphql/pagination"
	"github.com/JIeeiroSst/utils/graphql/resolver"
	"github.com/JIeeiroSst/utils/graphql/validation"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"

	gqlcontext "github.com/JIeeiroSst/utils/graphql/context"
)

const schemaString = `
	schema {
		query: Query
		mutation: Mutation
	}

	type Query {
		user(id: ID!): User
		users(first: Int, after: String): UserConnection!
		me: User
	}

	type Mutation {
		createUser(input: CreateUserInput!): User!
		updateUser(id: ID!, input: UpdateUserInput!): User!
		deleteUser(id: ID!): Boolean!
	}

	type User {
		id: ID!
		email: String!
		name: String!
		role: String!
		createdAt: String!
		posts: [Post!]!
	}

	type Post {
		id: ID!
		title: String!
		content: String!
		author: User!
	}

	type UserConnection {
		edges: [UserEdge!]!
		pageInfo: PageInfo!
		totalCount: Int
	}

	type UserEdge {
		cursor: String!
		node: User!
	}

	type PageInfo {
		hasNextPage: Boolean!
		hasPreviousPage: Boolean!
		startCursor: String
		endCursor: String
	}

	input CreateUserInput {
		email: String!
		name: String!
		password: String!
	}

	input UpdateUserInput {
		email: String
		name: String
	}
`

type User struct {
	ID        string
	Email     string
	Name      string
	Role      string
	CreatedAt time.Time
}

type Post struct {
	ID       string
	Title    string
	Content  string
	AuthorID string
}

type RootResolver struct {
	*resolver.BaseResolver
}

func NewRootResolver() *RootResolver {
	return &RootResolver{
		BaseResolver: resolver.NewBaseResolver(),
	}
}

func (r *RootResolver) User(ctx context.Context, args struct{ ID graphql.ID }) (*UserResolver, error) {
	loaders := getLoadersFromContext(ctx)
	userLoader, _ := loaders.Get("user")

	result, err := userLoader.Load(ctx, string(args.ID))
	if err != nil {
		return nil, r.WrapNotFoundWithID("User", string(args.ID), err)
	}

	return &UserResolver{user: result.(*User)}, nil
}

func (r *RootResolver) Users(ctx context.Context, args struct {
	First *int32
	After *string
}) (*UserConnectionResolver, error) {
	input := &pagination.CursorPaginationInput{
		First: args.First,
		After: args.After,
	}

	if err := pagination.ValidateCursorPagination(input, pagination.MaxLimit); err != nil {
		return nil, errors.BadRequest(err.Error())
	}

	limit, offset, err := pagination.GetLimitOffset(input, pagination.DefaultLimit)
	if err != nil {
		return nil, errors.BadRequest(err.Error())
	}

	users, total, err := queryUsers(ctx, limit, offset)
	if err != nil {
		return nil, errors.Internal("Failed to fetch users")
	}

	hasMore := len(users) > int(limit)-1
	if hasMore {
		users = users[:len(users)-1] 
	}

	items := make([]interface{}, len(users))
	for i, u := range users {
		items[i] = u
	}

	conn := pagination.NewConnectionWithTotal(items, total, hasMore, func(item interface{}) string {
		return pagination.EncodeCursor(item.(*User).ID)
	})

	return &UserConnectionResolver{conn: conn}, nil
}

func (r *RootResolver) Me(ctx context.Context) (*UserResolver, error) {
	uc, err := r.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	loaders := getLoadersFromContext(ctx)
	userLoader, _ := loaders.Get("user")

	result, err := userLoader.Load(ctx, uc.UserID)
	if err != nil {
		return nil, r.WrapNotFoundWithID("User", uc.UserID, err)
	}

	return &UserResolver{user: result.(*User)}, nil
}

func (r *RootResolver) CreateUser(ctx context.Context, args struct {
	Input struct {
		Email    string
		Name     string
		Password string
	}
}) (*UserResolver, error) {
	result := validation.NewValidationResult().
		AddField(validation.NewFieldValidator("email", args.Input.Email).Required().Email()).
		AddField(validation.NewFieldValidator("name", args.Input.Name).Required().MinLength(2)).
		AddField(validation.NewFieldValidator("password", args.Input.Password).Required().MinLength(8))

	if !result.IsValid() {
		return nil, result.Error()
	}

	user := &User{
		ID:        "new-user-id",
		Email:     args.Input.Email,
		Name:      args.Input.Name,
		Role:      "user",
		CreatedAt: time.Now(),
	}

	return &UserResolver{user: user}, nil
}

func (r *RootResolver) UpdateUser(ctx context.Context, args struct {
	ID    graphql.ID
	Input struct {
		Email *string
		Name  *string
	}
}) (*UserResolver, error) {
	authMiddleware := middleware.AuthMiddleware(true)

	result, err := authMiddleware(ctx, func(ctx context.Context) (interface{}, error) {
		user, err := getUser(ctx, string(args.ID))
		if err != nil {
			return nil, r.WrapNotFoundWithID("User", string(args.ID), err)
		}

		if err := r.CheckOwnership(ctx, user.ID); err != nil {
			return nil, err
		}

		if args.Input.Email != nil {
			if err := validation.ValidateEmail(*args.Input.Email); err != nil {
				return nil, errors.ValidationError("email", err.Error())
			}
			user.Email = *args.Input.Email
		}

		if args.Input.Name != nil {
			user.Name = *args.Input.Name
		}

		return &UserResolver{user: user}, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*UserResolver), nil
}

func (r *RootResolver) DeleteUser(ctx context.Context, args struct{ ID graphql.ID }) (bool, error) {
	_, err := r.RequireRole(ctx, "admin")
	if err != nil {
		return false, err
	}

	return true, nil
}

type UserResolver struct {
	user *User
}

func (ur *UserResolver) ID() graphql.ID {
	return graphql.ID(ur.user.ID)
}

func (ur *UserResolver) Email() string {
	return ur.user.Email
}

func (ur *UserResolver) Name() string {
	return ur.user.Name
}

func (ur *UserResolver) Role() string {
	return ur.user.Role
}

func (ur *UserResolver) CreatedAt() string {
	return ur.user.CreatedAt.Format(time.RFC3339)
}

func (ur *UserResolver) Posts(ctx context.Context) ([]*PostResolver, error) {
	posts, err := getPostsByAuthorID(ctx, ur.user.ID)
	if err != nil {
		return nil, err
	}

	resolvers := make([]*PostResolver, len(posts))
	for i, post := range posts {
		resolvers[i] = &PostResolver{post: post}
	}

	return resolvers, nil
}

type PostResolver struct {
	post *Post
}

func (pr *PostResolver) ID() graphql.ID {
	return graphql.ID(pr.post.ID)
}

func (pr *PostResolver) Title() string {
	return pr.post.Title
}

func (pr *PostResolver) Content() string {
	return pr.post.Content
}

func (pr *PostResolver) Author(ctx context.Context) (*UserResolver, error) {
	loaders := getLoadersFromContext(ctx)
	userLoader, _ := loaders.Get("user")

	result, err := userLoader.Load(ctx, pr.post.AuthorID)
	if err != nil {
		return nil, err
	}

	return &UserResolver{user: result.(*User)}, nil
}

type UserConnectionResolver struct {
	conn *pagination.Connection
}

func (ucr *UserConnectionResolver) Edges() []*UserEdgeResolver {
	edges := make([]*UserEdgeResolver, len(ucr.conn.Edges))
	for i, edge := range ucr.conn.Edges {
		edges[i] = &UserEdgeResolver{edge: edge}
	}
	return edges
}

func (ucr *UserConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: ucr.conn.PageInfo}
}

func (ucr *UserConnectionResolver) TotalCount() *int32 {
	return ucr.conn.TotalCount
}

type UserEdgeResolver struct {
	edge *pagination.Edge
}

func (uer *UserEdgeResolver) Cursor() string {
	return uer.edge.Cursor
}

func (uer *UserEdgeResolver) Node() *UserResolver {
	return &UserResolver{user: uer.edge.Node.(*User)}
}

type PageInfoResolver struct {
	pageInfo *pagination.PageInfo
}

func (pir *PageInfoResolver) HasNextPage() bool {
	return pir.pageInfo.HasNextPage
}

func (pir *PageInfoResolver) HasPreviousPage() bool {
	return pir.pageInfo.HasPreviousPage
}

func (pir *PageInfoResolver) StartCursor() *string {
	return pir.pageInfo.StartCursor
}

func (pir *PageInfoResolver) EndCursor() *string {
	return pir.pageInfo.EndCursor
}

func setupLoaders() *loader.LoaderMap {
	loaderMap := loader.NewLoaderMap()

	userLoader := loader.NewLoader(func(ctx context.Context, ids []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, id := range ids {
			result[id] = &User{
				ID:        id,
				Email:     "user@example.com",
				Name:      "User " + id,
				Role:      "user",
				CreatedAt: time.Now(),
			}
		}
		return result, nil
	})

	loaderMap.Register("user", userLoader)
	return loaderMap
}

func getLoadersFromContext(ctx context.Context) *loader.LoaderMap {
	return ctx.Value("loaders").(*loader.LoaderMap)
}

func queryUsers(ctx context.Context, limit, offset int) ([]*User, int, error) {
	users := []*User{
		{ID: "1", Email: "user1@example.com", Name: "User 1", Role: "user"},
		{ID: "2", Email: "user2@example.com", Name: "User 2", Role: "user"},
	}
	return users, 100, nil
}

func getUser(ctx context.Context, id string) (*User, error) {
	return &User{
		ID:        id,
		Email:     "user@example.com",
		Name:      "User",
		Role:      "user",
		CreatedAt: time.Now(),
	}, nil
}

func getPostsByAuthorID(ctx context.Context, authorID string) ([]*Post, error) {
	return []*Post{
		{ID: "1", Title: "Post 1", Content: "Content", AuthorID: authorID},
	}, nil
}

func main() {
	schema := graphql.MustParseSchema(schemaString, NewRootResolver())

	http.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, role := extractUserFromToken(r)

		ctx = gqlcontext.WithUserID(ctx, userID)
		ctx = gqlcontext.WithUserRole(ctx, role)
		ctx = gqlcontext.WithRequestID(ctx, generateRequestID())

		ctx = context.WithValue(ctx, "loaders", setupLoaders())

		(&relay.Handler{Schema: schema}).ServeHTTP(w, r.WithContext(ctx))
	})

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func extractUserFromToken(r *http.Request) (string, string) {
	return "user-123", "user"
}

func generateRequestID() string {
	return "req-" + time.Now().Format("20060102150405")
}
