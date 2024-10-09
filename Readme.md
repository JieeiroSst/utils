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
