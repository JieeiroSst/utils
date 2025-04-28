package hash

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	ZipCode string `json:"zip_code"`
}

type Person struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Age       int       `json:"age"`
	Active    bool      `json:"active"`
	Address   Address   `json:"address"`
	Contacts  []string  `json:"contacts"`
	Friends   []*Person `json:"friends"`
	CreatedAt time.Time `json:"created_at"`
	Score     *float64  `json:"score"`
}

func setupRedis(_ *testing.T) (*redis.Client, func()) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})

	ctx := context.Background()
	client.FlushDB(ctx)

	return client, func() {
		client.FlushDB(ctx)
		client.Close()
	}
}

func TestCacheHelper(t *testing.T) {
	ctx := context.Background()

	_, cleanup := setupRedis(t)
	defer cleanup()

	cache := NewCacheHelper("localhost:6379")

	t.Run("Basic types", func(t *testing.T) {
		err := cache.Set(ctx, "test:string", "hello world")
		assert.NoError(t, err)

		var result string
		err = cache.Get(ctx, "test:string", &result)
		assert.NoError(t, err)
		assert.Equal(t, "hello world", result)

		err = cache.Set(ctx, "test:int", 42)
		assert.NoError(t, err)

		var intResult int
		err = cache.Get(ctx, "test:int", &intResult)
		assert.NoError(t, err)
		assert.Equal(t, 42, intResult)
	})

	t.Run("Struct object", func(t *testing.T) {
		address := Address{
			Street:  "123 Test St",
			City:    "Test City",
			ZipCode: "12345",
		}

		err := cache.Set(ctx, "test:address", address)
		assert.NoError(t, err)

		var retrievedAddress Address
		err = cache.Get(ctx, "test:address", &retrievedAddress)
		assert.NoError(t, err)
		assert.Equal(t, address, retrievedAddress)
	})

	t.Run("Complex struct with nested objects and arrays", func(t *testing.T) {
		score := 95.5
		friend := &Person{
			ID:     2,
			Name:   "Friend",
			Age:    25,
			Active: true,
		}

		person := Person{
			ID:        1,
			Name:      "Test Person",
			Age:       30,
			Active:    true,
			Address:   Address{Street: "123 Main St", City: "Test City", ZipCode: "12345"},
			Contacts:  []string{"test@example.com", "+1234567890"},
			Friends:   []*Person{friend},
			CreatedAt: time.Now(),
			Score:     &score,
		}

		err := cache.Set(ctx, "test:person", person)
		assert.NoError(t, err)

		var retrievedPerson Person
		err = cache.Get(ctx, "test:person", &retrievedPerson)
		assert.NoError(t, err)

		assert.Equal(t, person.ID, retrievedPerson.ID)
		assert.Equal(t, person.Name, retrievedPerson.Name)
		assert.Equal(t, person.Age, retrievedPerson.Age)
		assert.Equal(t, person.Active, retrievedPerson.Active)

		assert.Equal(t, person.Address.Street, retrievedPerson.Address.Street)
		assert.Equal(t, person.Address.City, retrievedPerson.Address.City)

		assert.Equal(t, len(person.Contacts), len(retrievedPerson.Contacts))
		assert.Equal(t, person.Contacts[0], retrievedPerson.Contacts[0])

		assert.NotNil(t, retrievedPerson.Score)
		assert.Equal(t, *person.Score, *retrievedPerson.Score)

		assert.Equal(t, 1, len(retrievedPerson.Friends))
		assert.Equal(t, friend.ID, retrievedPerson.Friends[0].ID)
		assert.Equal(t, friend.Name, retrievedPerson.Friends[0].Name)
	})

	t.Run("Using GetInterface and SetInterface with hash data", func(t *testing.T) {
		data := map[string]interface{}{
			"id":        1,
			"name":      "Test User",
			"age":       30,
			"active":    true,
			"address":   Address{Street: "123 Main St", City: "Test City", ZipCode: "12345"},
			"contacts":  []string{"test@example.com", "+1234567890"},
			"timestamp": time.Now(),
		}

		err := cache.SetInterface(ctx, "test:hash", data)
		assert.NoError(t, err)

		type User struct {
			ID       int      `json:"id"`
			Name     string   `json:"name"`
			Age      int      `json:"age"`
			Active   bool     `json:"active"`
			Address  Address  `json:"address"`
			Contacts []string `json:"contacts"`
		}

		result, err := cache.GetInterface(ctx, "test:hash", User{})
		assert.NoError(t, err)

		assert.Equal(t, float64(1), result["id"])
		assert.Equal(t, "Test User", result["name"])

		addressMap, ok := result["address"].(map[string]interface{})
		assert.True(t, ok, "Address should be a map")
		assert.Equal(t, "123 Main St", addressMap["street"])
		assert.Equal(t, "Test City", addressMap["city"])

		contactsArr, ok := result["contacts"].([]interface{})
		assert.True(t, ok, "Contacts should be an array")
		assert.Equal(t, 2, len(contactsArr))
		assert.Equal(t, "test@example.com", contactsArr[0])
	})

	t.Run("Remove key", func(t *testing.T) {
		err := cache.Set(ctx, "test:remove", "to be removed")
		assert.NoError(t, err)

		err = cache.RemoveKey(ctx, "test:remove")
		assert.NoError(t, err)

		var result string
		err = cache.Get(ctx, "test:remove", &result)
		assert.Error(t, err)
	})
}
