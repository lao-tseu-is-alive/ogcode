package id

import (
	"math/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	entropy = rand.New(rand.NewSource(time.Now().UnixNano()))
	mu      sync.Mutex
)

type SessionID string
type MessageID string
type PartID string
type PermissionID string

func NewSessionID() SessionID {
	return SessionID("ses_" + newULID())
}

func NewMessageID() MessageID {
	return MessageID("msg_" + newULID())
}

func NewPartID() PartID {
	return PartID("prt_" + newULID())
}

func NewPermissionID() PermissionID {
	return PermissionID("prm_" + newULID())
}

func newULID() string {
	mu.Lock()
	defer mu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}