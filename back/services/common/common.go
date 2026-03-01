// package common to export common types in services
package common

import (
	"context"
	"sync"
)

type CustomHandler struct {
	Ctx        context.Context
	Mu         sync.Mutex
	CancelFunc context.CancelFunc
}
type RequestBody struct {
	Text string `json:"text"`
}
