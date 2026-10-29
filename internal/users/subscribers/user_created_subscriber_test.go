package subscribers

import (
	"context"
	"testing"
	"uuid"

	"github.com/cronos/keno4min/internal/nodes/events"
	"github.com/cronos/keno4min/internal/users"
	"github.com/cronos/keno4min/internal/users/usecase"
)

type recordingRepository struct {
	user          *users.User
	readErr       error
	saveErr       error
	reads, writes int
	lookupID      uuid.UUID
}

func (r *recordingRepository) FindById(_ context.Context, id uuid.UUID) (*users.User, error) {
	r.reads++
	r.lookupID = id
	return r.user, r.readErr
}

func (r *recordingRepository) FindByUsername(context.Context, string) (*users.User, error) {
	panic("creation must look up users by ID")
}

func (r *recordingRepository) Save(_ context.Context, user *users.User) error {
	r.writes++
	if r.saveErr == nil {
		r.user = user
	}
	return r.saveErr
}

func TestUserCreatedRejectsOtherEvents(t *testing.T) {
	repo := &recordingRepository{}
	subscriber := NewUserCreatedSubscriber(usecase.NewCreate(repo))
	if err := subscriber.On(context.Background(), events.NodeCreatedDomainEvent{}); err == nil {
		t.Fatal("expected wrong event error")
	}
	if repo.reads != 0 || repo.writes != 0 {
		t.Fatal("wrong event accessed repository")
	}
}
