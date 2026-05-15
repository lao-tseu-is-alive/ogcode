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
type PlanID string
type TaskID string

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

func NewPlanID() PlanID {
	return PlanID("pln_" + newULID())
}

func NewTaskID() TaskID {
	return TaskID("tsk_" + newULID())
}

func NewNoteID() string {
	return "nte_" + newULID()
}

func NewNoteVersionID() string {
	return "ntv_" + newULID()
}

func newULID() string {
	mu.Lock()
	defer mu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}