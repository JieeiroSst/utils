```
func doSomething(ctx context.Context) {
	// Extract the tracer ID from the context
	span := trace.SpanFromContext(ctx)

	// Log the tracer ID and other information
	log.Printf("Tracer ID: %s", span.SpanContext().TraceID().String())
	// ... do other things
}

```

```
var p1, p2 struct {
	Title  string `redis:"title"`
	Author string `redis:"author"`
	Body   string `redis:"body"`
}

p1.Title = "Example"
p1.Author = "Gary"
p1.Body = "Hello"

if _, err := c.Do("HMSET", redis.Args{}.Add("id1").AddFlat(&p1)...); err != nil {
	fmt.Println(err)
	return
}
```

```
	// Generate a 256-bit AES key (store securely in production!)
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}

	vault := NewTokenVault(key)

	// Tokenize sensitive data
	token, err := vault.Tokenize([]byte("4111"))
	if err != nil {
		panic(err)
	}
	fmt.Println("Token:", token)

	// Retrieve original data
	decrypted, err := vault.Retrieve(token)
	if err != nil {
		panic(err)
	}
	fmt.Println("Decrypted:", string(decrypted))

	// Validate token
	fmt.Println("Token exists:", vault.ValidateToken(token))
```

```
func main() {
	tokenManager := NewTokenManager(
		[]byte("access-secret-key"),
		[]byte("refresh-secret-key"),
		15*time.Minute, // Access token TTL
		24*7*time.Hour, // Refresh token TTL (1 week)
	)

	accessToken, refreshToken, err := tokenManager.CreateTokenPair("user123")
	if err != nil {
		log.Fatal(err)
	}

	claims, err := tokenManager.ValidateToken(accessToken, false)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Access token for user %s is valid\n", claims.UserID)

	newAccess, newRefresh, err := tokenManager.RefreshTokens(refreshToken)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("New access token: %s, newRefresh: %s\n", newAccess, newRefresh)
}
```

```
	LRU
	Initialize Redis client
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	Create a new LRU cache with maximum 1000 entries
	cache := NewLRUCache(client, "my-cache", 1000)

	ctx := context.Background()

	Set a value
	err := cache.Set(ctx, "key1", "value1")

	Get a value
	value, err := cache.Get(ctx, "key1")
```

```
	LFU
	ctx := context.Background()
    cache, err := NewCache("localhost:6379", "", "my-hash")
    if err != nil {
        log.Fatal(err)
    }
    defer cache.Close()
 	
	Example usage
    type User struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
    }

    Store a user
    user := User{ID: 1, Name: "John"}
    err = cache.Set(ctx, "user:1", user, time.Hour)
    
     Retrieve a user
    var retrievedUser User
    err = cache.Get(ctx, "user:1", &retrievedUser)
```

```
	FIFO
	Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	Create a new FIFO cache with maximum 1000 items
	cache := NewFIFOCache(rdb, "my-cache", 1000)

	Set a value
	ctx := context.Background()
	err := cache.Set(ctx, "key1", "value1")

	Get a value
	value, err := cache.Get(ctx, "key1")

	Delete a value
	err = cache.Delete(ctx, "key1")
```