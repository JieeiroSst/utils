package rollback

import (
	"database/sql"
	"fmt"
)

func RollbackTransactionSql(tx *sql.Tx, err error) error {
	if rbErr := tx.Rollback(); rbErr != nil {
		if err != nil {
			return fmt.Errorf("original error: %v, rollback error: %v", err, rbErr)
		}
		return fmt.Errorf("rollback error: %v", rbErr)
	}

	if err != nil {
		return err
	}

	return nil
}
