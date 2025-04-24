```
git tag v1.0.0
git push origin v1.0.0
```
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


func ExampleProducer() {
	brokers := []string{"localhost:9092"}
	topic := "example-topic"

	client, err := NewKafkaClient(brokers)
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %v", err)
	}
	defer client.Close()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan bool)

	go func() {
		messageCount := 1
		for {
			select {
			case <-done:
				return
			default:
				key := []byte(fmt.Sprintf("key-%d", messageCount))
				value := []byte(fmt.Sprintf("message-%d-sent-at-%s", messageCount, time.Now().Format(time.RFC3339)))

				partition, offset, err := client.SendMessage(topic, key, value)
				if err != nil {
					log.Printf("Failed to send message: %v", err)
				} else {
					log.Printf("Message sent successfully: topic=%s, partition=%d, offset=%d, key=%s, value=%s",
						topic, partition, offset, string(key), string(value))
				}

				messageCount++
				time.Sleep(2 * time.Second)
			}
		}
	}()

	<-signals
	log.Println("Shutting down...")

	close(done)

	time.Sleep(1 * time.Second)
	log.Println("Application stopped")
}

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


```
RABBITMQ

func ExampleRabbitMQ() {
	config := Config{
		Host:     "localhost",
		Port:     5672,
		Username: "guest",
		Password: "guest",
		Vhost:    "/",
	}

	// Create a connection
	conn, err := NewConnection(config)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	directExchangeExample(ctx, conn)

	defaultExchangeExample(ctx, conn)

	fanoutExchangeExample(ctx, conn)

	topicExchangeExample(ctx, conn)

	headersExchangeExample(ctx, conn)

	workerExample(ctx, conn)

	batchPublisherExample(ctx, conn)
}

// Direct Exchange Example
func directExchangeExample(ctx context.Context, conn *Connection) {
	fmt.Println("\n--- Direct Exchange Example ---")

	producer, err := NewProducer(conn, "direct_exchange", DirectExchange)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	producer.SetRoutingKey("direct_key")

	consumer, err := NewConsumer(conn, "direct_exchange", DirectExchange, ConsumerOptions{
		QueueName: "direct_queue",
		AutoAck:   true,
	})
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	consumer.SetRoutingKey("direct_key")

	if err := consumer.BindQueue(); err != nil {
		log.Fatalf("Failed to bind queue: %v", err)
	}

	deliveries, err := consumer.Consume(ctx)
	if err != nil {
		log.Fatalf("Failed to start consuming: %v", err)
	}

	go func() {
		for delivery := range deliveries {
			fmt.Printf("Direct Exchange - Received: %s\n", delivery.Body)
		}
	}()

	message := []byte("Hello from Direct Exchange!")
	if err := producer.Publish(ctx, message, nil); err != nil {
		log.Fatalf("Failed to publish message: %v", err)
	}

	fmt.Println("Direct Exchange - Published message")
	time.Sleep(time.Second)
}

func defaultExchangeExample(ctx context.Context, conn *Connection) {
	fmt.Println("\n--- Default Exchange Example ---")

	queue, err := conn.QueueDeclare(
		"default_queue",
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	producer, err := NewProducer(conn, "", DefaultExchange)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}

	producer.SetRoutingKey(queue.Name)

	consumer, err := NewConsumer(conn, "", DefaultExchange, ConsumerOptions{
		QueueName: "default_queue",
		AutoAck:   true,
	})
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}

	deliveries, err := consumer.Consume(ctx)
	if err != nil {
		log.Fatalf("Failed to start consuming: %v", err)
	}

	go func() {
		for delivery := range deliveries {
			fmt.Printf("Default Exchange - Received: %s\n", delivery.Body)
		}
	}()

	message := []byte("Hello from Default Exchange!")
	if err := producer.Publish(ctx, message, nil); err != nil {
		log.Fatalf("Failed to publish message: %v", err)
	}

	fmt.Println("Default Exchange - Published message")
	time.Sleep(time.Second)
}

func fanoutExchangeExample(ctx context.Context, conn *Connection) {
	fmt.Println("\n--- Fanout Exchange Example ---")

	producer, err := NewProducer(conn, "fanout_exchange", FanoutExchange)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}

	subscriber1, err := NewSubscriber(conn, "fanout_exchange", FanoutExchange, SubscriberOptions{
		QueueName:   "fanout_queue_1",
		ConsumerTag: "consumer_1",
		AutoAck:     true,
	})
	if err != nil {
		log.Fatalf("Failed to create subscriber 1: %v", err)
	}

	subscriber2, err := NewSubscriber(conn, "fanout_exchange", FanoutExchange, SubscriberOptions{
		QueueName:   "fanout_queue_2",
		ConsumerTag: "consumer_2",
		AutoAck:     true,
	})
	if err != nil {
		log.Fatalf("Failed to create subscriber 2: %v", err)
	}

	if err := subscriber1.BindQueue(); err != nil {
		log.Fatalf("Failed to bind queue 1: %v", err)
	}

	if err := subscriber2.BindQueue(); err != nil {
		log.Fatalf("Failed to bind queue 2: %v", err)
	}

	deliveries1, err := subscriber1.Subscribe(ctx)
	if err != nil {
		log.Fatalf("Failed to start subscriber 1: %v", err)
	}

	deliveries2, err := subscriber2.Subscribe(ctx)
	if err != nil {
		log.Fatalf("Failed to start subscriber 2: %v", err)
	}

	go func() {
		for delivery := range deliveries1 {
			fmt.Printf("Fanout Exchange (Queue 1) - Received: %s\n", delivery.Body)
		}
	}()

	go func() {
		for delivery := range deliveries2 {
			fmt.Printf("Fanout Exchange (Queue 2) - Received: %s\n", delivery.Body)
		}
	}()

	message := []byte("Hello from Fanout Exchange!")
	if err := producer.Publish(ctx, message, nil); err != nil {
		log.Fatalf("Failed to publish message: %v", err)
	}

	fmt.Println("Fanout Exchange - Published message")
	time.Sleep(time.Second)
}

func topicExchangeExample(ctx context.Context, conn *Connection) {
	fmt.Println("\n--- Topic Exchange Example ---")

	producer, err := NewProducer(conn, "topic_exchange", TopicExchange)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	allLogsSubscriber, err := NewSubscriber(conn, "topic_exchange", TopicExchange, SubscriberOptions{
		QueueName:   "all_logs_queue",
		BindingKeys: []string{"log.#"},
		ConsumerTag: "all_logs_consumer",
		AutoAck:     true,
	})
	if err != nil {
		log.Fatalf("Failed to create all logs subscriber: %v", err)
	}

	errorLogsSubscriber, err := NewSubscriber(conn, "topic_exchange", TopicExchange, SubscriberOptions{
		QueueName:   "error_logs_queue",
		BindingKeys: []string{"log.error.*"},
		ConsumerTag: "error_logs_consumer",
		AutoAck:     true,
	})
	if err != nil {
		log.Fatalf("Failed to create error logs subscriber: %v", err)
	}

	if err := allLogsSubscriber.BindQueue(); err != nil {
		log.Fatalf("Failed to bind all logs queue: %v", err)
	}

	if err := errorLogsSubscriber.BindQueue(); err != nil {
		log.Fatalf("Failed to bind error logs queue: %v", err)
	}

	allLogsDeliveries, err := allLogsSubscriber.Subscribe(ctx)
	if err != nil {
		log.Fatalf("Failed to start all logs subscriber: %v", err)
	}

	errorLogsDeliveries, err := errorLogsSubscriber.Subscribe(ctx)
	if err != nil {
		log.Fatalf("Failed to start error logs subscriber: %v", err)
	}

	go func() {
		for delivery := range allLogsDeliveries {
			fmt.Printf("Topic Exchange (All Logs) - Received: %s\n", delivery.Body)
		}
	}()

	go func() {
		for delivery := range errorLogsDeliveries {
			fmt.Printf("Topic Exchange (Error Logs) - Received: %s\n", delivery.Body)
		}
	}()

	producer.SetRoutingKey("log.info.application")
	infoMessage := []byte("This is an info log message")
	if err := producer.Publish(ctx, infoMessage, nil); err != nil {
		log.Fatalf("Failed to publish info message: %v", err)
	}

	producer.SetRoutingKey("log.error.database")
	errorMessage := []byte("This is an error log message")
	if err := producer.Publish(ctx, errorMessage, nil); err != nil {
		log.Fatalf("Failed to publish error message: %v", err)
	}

	fmt.Println("Topic Exchange - Published messages")
	time.Sleep(time.Second)
}

func headersExchangeExample(ctx context.Context, conn *Connection) {
	fmt.Println("\n--- Headers Exchange Example ---")

	producer, err := NewProducer(conn, "headers_exchange", HeadersExchange)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}

	allMatchSubscriber, err := NewSubscriber(conn, "headers_exchange", HeadersExchange, SubscriberOptions{
		QueueName:   "all_match_queue",
		ConsumerTag: "all_match_consumer",
		AutoAck:     true,
		Args: HeadersBinding(All, map[string]interface{}{
			"format": "json",
			"type":   "log",
		}),
	})
	if err != nil {
		log.Fatalf("Failed to create all match subscriber: %v", err)
	}

	// Subscriber for messages with EITHER format=xml OR type=report
	anyMatchSubscriber, err := NewSubscriber(conn, "headers_exchange", HeadersExchange, SubscriberOptions{
		QueueName:   "any_match_queue",
		ConsumerTag: "any_match_consumer",
		AutoAck:     true,
		Args: HeadersBinding(Any, map[string]interface{}{
			"format": "xml",
			"type":   "report",
		}),
	})
	if err != nil {
		log.Fatalf("Failed to create any match subscriber: %v", err)
	}

	// Bind the queues
	if err := allMatchSubscriber.BindQueue(); err != nil {
		log.Fatalf("Failed to bind all match queue: %v", err)
	}

	if err := anyMatchSubscriber.BindQueue(); err != nil {
		log.Fatalf("Failed to bind any match queue: %v", err)
	}

	// Start consuming in goroutines
	allMatchDeliveries, err := allMatchSubscriber.Subscribe(ctx)
	if err != nil {
		log.Fatalf("Failed to start all match subscriber: %v", err)
	}

	anyMatchDeliveries, err := anyMatchSubscriber.Subscribe(ctx)
	if err != nil {
		log.Fatalf("Failed to start any match subscriber: %v", err)
	}

	go func() {
		for delivery := range allMatchDeliveries {
			fmt.Printf("Headers Exchange (All Match) - Received: %s\n", delivery.Body)
		}
	}()

	go func() {
		for delivery := range anyMatchDeliveries {
			fmt.Printf("Headers Exchange (Any Match) - Received: %s\n", delivery.Body)
		}
	}()

	allMatchMessage := []byte("This message has format=json AND type=log")
	allMatchHeaders := amqp.Table{
		"format": "json",
		"type":   "log",
	}
	if err := producer.PublishWithHeaders(ctx, allMatchMessage, allMatchHeaders); err != nil {
		log.Fatalf("Failed to publish all match message: %v", err)
	}

	anyMatchMessage := []byte("This message has format=xml only")
	anyMatchHeaders := amqp.Table{
		"format": "xml",
	}
	if err := producer.PublishWithHeaders(ctx, anyMatchMessage, anyMatchHeaders); err != nil {
		log.Fatalf("Failed to publish any match message: %v", err)
	}

	fmt.Println("Headers Exchange - Published messages")
	time.Sleep(time.Second)
}

func workerExample(ctx context.Context, conn *Connection) {
	fmt.Println("\n--- Worker Example ---")

	producer, err := NewProducer(conn, "worker_exchange", DirectExchange)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	producer.SetRoutingKey("worker_key")

	subscriber, err := NewSubscriber(conn, "worker_exchange", DirectExchange, SubscriberOptions{
		QueueName:     "worker_queue",
		BindingKeys:   []string{"worker_key"},
		ConsumerTag:   "worker",
		AutoAck:       false,
		PrefetchCount: 1,
	})
	if err != nil {
		log.Fatalf("Failed to create subscriber: %v", err)
	}

	if err := subscriber.BindQueue(); err != nil {
		log.Fatalf("Failed to bind queue: %v", err)
	}

	handler := func(delivery amqp.Delivery) error {
		fmt.Printf("Worker - Processing message: %s\n", delivery.Body)
		time.Sleep(500 * time.Millisecond)
		return nil
	}

	worker := NewWorker(subscriber, handler, 3, false)

	if err := worker.Start(ctx); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	for i := 0; i < 10; i++ {
		message := []byte(fmt.Sprintf("Worker message %d", i))
		if err := producer.Publish(ctx, message, nil); err != nil {
			log.Fatalf("Failed to publish message: %v", err)
		}
		fmt.Printf("Worker - Published message %d\n", i)
	}

	fmt.Println("Worker - Published all messages")
	time.Sleep(3 * time.Second)

	if err := worker.Stop(); err != nil {
		log.Printf("Error stopping worker: %v", err)
	}
}

func batchPublisherExample(ctx context.Context, conn *Connection) {
	fmt.Println("\n--- Batch Publisher Example ---")

	producer, err := NewProducer(conn, "batch_exchange", DirectExchange)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	producer.SetRoutingKey("batch_key")

	consumer, err := NewConsumer(conn, "batch_exchange", DirectExchange, ConsumerOptions{
		QueueName: "batch_queue",
		AutoAck:   true,
	})
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	consumer.SetRoutingKey("batch_key")

	if err := consumer.BindQueue(); err != nil {
		log.Fatalf("Failed to bind queue: %v", err)
	}

	deliveries, err := consumer.Consume(ctx)
	if err != nil {
		log.Fatalf("Failed to start consuming: %v", err)
	}

	count := 0
	go func() {
		for range deliveries {
			count++
			fmt.Printf("Batch Publisher - Received message count: %d\n", count)
		}
	}()

	batchPublisher := NewBatchPublisher(producer, 5)

	for i := 0; i < 12; i++ {
		message := []byte(fmt.Sprintf("Batch message %d", i))
		batchPublisher.Add(message, nil)

		if batchPublisher.IsFull() {
			published, err := batchPublisher.Publish(ctx)
			if err != nil {
				log.Fatalf("Failed to publish batch: %v", err)
			}
			fmt.Printf("Batch Publisher - Published batch of %d messages\n", published)
		}
	}

	if batchPublisher.Size() > 0 {
		published, err := batchPublisher.Publish(ctx)
		if err != nil {
			log.Fatalf("Failed to publish remaining batch: %v", err)
		}
		fmt.Printf("Batch Publisher - Published remaining batch of %d messages\n", published)
	}

	fmt.Println("Batch Publisher - Published all messages")
	time.Sleep(time.Second)
}

```

```
logger.InitDefault(logger.Config{
		Level:      "info",
		JSONFormat: true,
		AppName:    "my-service",
	})

	ctx := context.Background()
	ctx = trace_id.EnsureTracerID(ctx)

	logger.WithContext(ctx).Info("Processing request",
		zap.String("user_id", "12345"),
		zap.Int("items", 42),
	)

	if err := processItem(ctx); err != nil {
		logger.WithContext(ctx).WithError(err).Error("Failed to process item")
	}

	book := Book{
		ID:   "1",
		Name: "San Pham",
	}

	logger.WithContext(ctx).Info("info book", zap.Any("book", book))
```

```
rollback gorm 
func ExampleUsage() {
	var db *gorm.DB

	err := ExecuteWithRollbackGorm(db, func(tx *gorm.DB) error {
		user := User{Name: "John"}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		product := Product{Title: "New Product"}
		if err := tx.Create(&product).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Printf("Transaction failed: %v", err)
	}
}

sql 
func InsertUser(db *sql.DB, user User) error {
    tx, err := db.Begin()
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %v", err)
    }
    
    _, err = tx.Exec("INSERT INTO users (name, email) VALUES (?, ?)", user.Name, user.Email)
    if err != nil {
        return rollbackTransaction(tx, fmt.Errorf("failed to insert user: %v", err))
    }
    
    if err = tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %v", err)
    }
    
    return nil
}
```