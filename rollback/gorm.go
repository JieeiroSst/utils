package rollback

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

func ExecuteWithRollbackGorm(db *gorm.DB, txFunc func(tx *gorm.DB) error) error {
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("transaction rolled back due to panic: %v", r)
		}
	}()

	if err := txFunc(tx); err != nil {
		if rbErr := tx.Rollback().Error; rbErr != nil {
			log.Printf("error during rollback: %v", rbErr)
		}
		return err
	}

	if err := tx.Commit().Error; err != nil {
		if rbErr := tx.Rollback().Error; rbErr != nil {
			log.Printf("error during rollback after failed commit: %v", rbErr)
		}
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
