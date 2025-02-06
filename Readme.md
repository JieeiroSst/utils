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