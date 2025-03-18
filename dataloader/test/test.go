package dataloader

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/JIeeiroSst/utils/dataloader" 
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID    int    `db:"id" gorm:"primaryKey"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

type Post struct {
	ID     int    `db:"id" gorm:"primaryKey"`
	UserID int    `db:"user_id" gorm:"index"`
	Title  string `db:"title"`
	Body   string `db:"body"`
}

type Comment struct {
	ID     int    `db:"id" gorm:"primaryKey"`
	PostID int    `db:"post_id" gorm:"index"`
	UserID int    `db:"user_id" gorm:"index"`
	Body   string `db:"body"`
}

func Example() {
	sqlExample()

	gormExample()

	batchLoaderExample()
}

func sqlExample() {
	db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/mydatabase")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create a SQLLoader with MySQL
	loader := dataloader.NewSQLLoader(
		db,
		dataloader.WithDBType(dataloader.MySQL),
		dataloader.WithMaxBatchSize(100),
		dataloader.WithWait(time.Millisecond*5),
		dataloader.WithCacheTTL(time.Minute*5),
	)

	// Load users by IDs
	userIDs := []interface{}{1, 2, 3, 4, 5}
	var users []User
	ctx := context.Background()

	err = loader.LoadMany(ctx, "users", "id", userIDs, &users)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d users\n", len(users))

	var posts []Post
	err = loader.LoadMany(ctx, "posts", "user_id", userIDs, &posts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d posts\n", len(posts))

	postIDs := make([]interface{}, len(posts))
	for i, post := range posts {
		postIDs[i] = post.ID
	}

	var comments []Comment
	err = loader.LoadMany(ctx, "comments", "post_id", postIDs, &comments)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d comments\n", len(comments))
}

func gormExample() {
	dialector := postgres.Open("host=localhost user=postgres password=postgres dbname=mydatabase port=5432 sslmode=disable")
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	loader := dataloader.NewGORMLoader(
		db,
		dataloader.WithDBType(dataloader.PostgreSQL),
		dataloader.WithMaxBatchSize(100),
		dataloader.WithWait(time.Millisecond*5),
		dataloader.WithCacheTTL(time.Minute*5),
	)

	userIDs := []interface{}{1, 2, 3, 4, 5}
	var users []User
	ctx := context.Background()

	err = loader.LoadMany(ctx, &User{}, "id", userIDs, &users)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d users with GORM\n", len(users))

	var posts []Post
	err = loader.LoadMany(ctx, &Post{}, "user_id", userIDs, &posts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d posts with GORM\n", len(posts))

	postIDs := make([]interface{}, len(posts))
	for i, post := range posts {
		postIDs[i] = post.ID
	}

	var comments []Comment
	err = loader.LoadMany(ctx, &Comment{}, "post_id", postIDs, &comments)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d comments with GORM\n", len(comments))
}

func batchLoaderExample() {
	dialector := mysql.Open("user:password@tcp(localhost:3306)/mydatabase?parseTime=true")
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	batchFn := func(ctx context.Context, keys []interface{}) (map[interface{}]interface{}, error) {
		var users []User
		if err := db.Where("id IN ?", keys).Find(&users).Error; err != nil {
			return nil, err
		}

		result := make(map[interface{}]interface{}, len(users))
		for _, user := range users {
			result[user.ID] = user
		}

		return result, nil
	}

	loader := dataloader.NewBatchLoader(
		batchFn,
		dataloader.WithMaxBatchSize(100),
		dataloader.WithWait(time.Millisecond*5),
		dataloader.WithCacheTTL(time.Minute*5),
	)
	defer loader.Close()

	ctx := context.Background()
	result, err := loader.Load(ctx, 1)
	if err != nil {
		log.Fatal(err)
	}

	user := result.(User)
	fmt.Printf("Loaded user: %s\n", user.Name)

	var wg sync.WaitGroup
	for i := 2; i <= 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			result, err := loader.Load(ctx, id)
			if err != nil {
				log.Printf("Error loading user %d: %v\n", id, err)
				return
			}
			user := result.(User)
			fmt.Printf("Loaded user: %s\n", user.Name)
		}(i)
	}

	wg.Wait()
}
