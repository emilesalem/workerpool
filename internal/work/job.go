package work

import (
	"time"

	"github.com/google/uuid"
)

type Job struct {
	Work    func() interface{}
	Elapsed time.Duration
	Result  interface{}
	ID      uuid.UUID
}
