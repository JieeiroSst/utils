package trace_id

import (
	"github.com/google/uuid"
)

func TracerID() string {
	return uuid.New().String()
}
