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

```
library: grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
GRPC circuitbreaker
type server struct {
	pb.UnimplementedGreeterServer

	duringError int
	counter     int32
}

var _ pb.GreeterServer = (*server)(nil)

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	atomic.AddInt32(&s.counter, 1)
	fmt.Println("[Server] Request handled.")

	if n := s.duringError - int(atomic.LoadInt32(&s.counter)); n >= 0 {
		return nil, status.New(codes.Internal, fmt.Sprintf("fail %d times left", n)).Err()
	}

	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func ExampleUnaryClientInterceptor() {
	lnAddress := "127.0.0.1:13000"

	// server
	go func() {
		listener, _ := net.Listen("tcp", lnAddress)
		defer listener.Close()

		s := grpc.NewServer()
		defer s.Stop()

		pb.RegisterGreeterServer(s, &server{
			duringError: 5,
		})

		_ = s.Serve(listener)
	}()

	// client
	go func() {
		time.Sleep(2 * time.Second) // waiting for serve
		ctx := context.Background()

		cb := circuitbreaker.New(
			circuitbreaker.WithCounterResetInterval(time.Minute),
			circuitbreaker.WithTripFunc(circuitbreaker.NewTripFuncThreshold(3)),
			circuitbreaker.WithOpenTimeout(2500*time.Millisecond),
			circuitbreaker.WithHalfOpenMaxSuccesses(3),
		)

		client, _ := grpc.DialContext(
			ctx,
			lnAddress,
			grpc.WithInsecure(),
			grpc.WithUnaryInterceptor(
				grpc_middleware.ChainUnaryClient(
					UnaryClientInterceptor(
						cb,
						func(ctx context.Context, method string, req interface{}) {
							fmt.Printf("[Client] Circuit breaker is open.\n")
						},
					),
				),
			),
		)
		defer client.Close()

		greeterClient := pb.NewGreeterClient(client)

		timer := time.NewTicker(time.Second)
		for {
			select {
			case <-timer.C:
				time.Sleep(50 * time.Millisecond)
				response, err := greeterClient.SayHello(ctx, &pb.HelloRequest{Name: "foo"})
				fmt.Printf("[Client] Response = %v, cb.State() = %v, Err = %v\n", response, cb.State(), err)
			default:
			}
		}
	}()
}
```

```
use library copy

var from []SourceStruct
var to []DestStruct
err := CopyStructArrays(&from, &to)

var from []SourceStruct
var to []DestStruct
err := CopyObject(&from, &to)
if err != nil {
    fmt.Printf("Error copying with reflection: %v\n", err)
}
```


```

func ExampleConsumer() {
	brokers := []string{"localhost:9092"}
	topics := []string{"example-topic"}

	handler := func(message *sarama.ConsumerMessage) error {
		fmt.Printf("Consumer received message: Topic=%s, Partition=%d, Offset=%d, Key=%s, Value=%s\n",
			message.Topic, message.Partition, message.Offset, string(message.Key), string(message.Value))
		return nil
	}

	consumer, err := NewConsumer(brokers, topics, handler)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		log.Println("Shutting down consumer...")
		cancel()
	}()

	if err := consumer.Consume(ctx); err != nil {
		log.Fatalf("Error consuming: %v", err)
	}
}

func ExampleSubscriber() {
	brokers := []string{"localhost:9092"}
	topics := []string{"example-topic"}
	groupID := "example-group"

	handler := func(message *sarama.ConsumerMessage) error {
		fmt.Printf("Subscriber received message: Topic=%s, Partition=%d, Offset=%d, Key=%s, Value=%s\n",
			message.Topic, message.Partition, message.Offset, string(message.Key), string(message.Value))
		return nil
	}

	subscriber, err := NewSubscriber(brokers, topics, groupID, handler)
	if err != nil {
		log.Fatalf("Failed to create subscriber: %v", err)
	}
	defer subscriber.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		log.Println("Shutting down subscriber...")
		cancel()
	}()

	if err := subscriber.Subscribe(ctx); err != nil {
		log.Fatalf("Error subscribing: %v", err)
	}
}

```