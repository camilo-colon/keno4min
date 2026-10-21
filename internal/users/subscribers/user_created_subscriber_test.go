package subscribers

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/cronos/keno4min/internal/nodes/events"
	"github.com/cronos/keno4min/internal/platform/workerfx"
	"github.com/cronos/keno4min/internal/users"
	userevents "github.com/cronos/keno4min/internal/users/events"
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

func TestUserCreatedDelivery(t *testing.T) {
	failure := errors.New("database unavailable")
	userID := uuid.MustParse("afed5b77-1e8a-471f-9398-489784a1b897")
	nodeID := uuid.MustParse("f2a0dc45-9ab8-4553-bd7c-2d53b345a5f6")
	for _, tc := range []struct {
		name, detail              string
		readErr, saveErr, wantErr error
		wantReads, wantWrites     int
	}{
		{name: "missing userId", detail: `{"eventId":"` + userID.String() + `", "username":"cajero", "nodeId":"` + nodeID.String() + `"}`, wantErr: usecase.ErrInvalidUser},
		{name: "zero userId", detail: `{"userId":"00000000-0000-0000-0000-000000000000", "username":"cajero", "nodeId":"` + nodeID.String() + `"}`, wantErr: usecase.ErrInvalidUser},
		{name: "created", detail: `{"userId":"` + userID.String() + `", "eventId":"d1e06436-1f35-4a45-ad66-601721030a46","username":"cajero", "roleId":2, "nodeId":"` + nodeID.String() + `"}`, wantReads: 1, wantWrites: 1},
		{name: "missing username", detail: `{"userId":"` + userID.String() + `", "eventId":"d1e06436-1f35-4a45-ad66-601721030a46","nodeId":"` + nodeID.String() + `"}`, wantErr: usecase.ErrInvalidUser},
		{name: "blank username", detail: `{"userId":"` + userID.String() + `", "eventId":"d1e06436-1f35-4a45-ad66-601721030a46","username":"  ","nodeId":"` + nodeID.String() + `"}`, wantErr: usecase.ErrInvalidUser},
		{name: "missing node", detail: `{"userId":"` + userID.String() + `", "eventId":"d1e06436-1f35-4a45-ad66-601721030a46","username":"cajero"}`, wantErr: usecase.ErrInvalidUser},
		{name: "null payload", detail: `null`, wantErr: usecase.ErrInvalidUser},
		{name: "read failure", detail: `{"userId":"` + userID.String() + `", "eventId":"d1e06436-1f35-4a45-ad66-601721030a46","username":"cajero", "nodeId":"` + nodeID.String() + `"}`, readErr: failure, wantErr: failure, wantReads: 1},
		{name: "save failure", detail: `{"userId":"` + userID.String() + `", "eventId":"d1e06436-1f35-4a45-ad66-601721030a46","username":"cajero", "nodeId":"` + nodeID.String() + `"}`, saveErr: failure, wantErr: failure, wantReads: 1, wantWrites: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &recordingRepository{readErr: tc.readErr, saveErr: tc.saveErr}
			subscriber := NewUserCreatedSubscriber(usecase.NewCreate(repo))
			deserializer, err := workerfx.NewDomainEventJSONDeserializer(subscriber.SubscribeTo())
			if err != nil {
				t.Fatal(err)
			}
			event, err := deserializer.Deserialize([]byte(`{"detail-type":"cronos.core.user.created","detail":` + tc.detail + `}`))
			if err != nil {
				t.Fatal(err)
			}
			if err := subscriber.On(context.Background(), event); !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if repo.reads != tc.wantReads || repo.writes != tc.wantWrites {
				t.Fatalf("unexpected operations: %+v", repo)
			}
			if tc.name == "created" {
				if repo.user.ID != userID || repo.lookupID != userID || repo.user.Username != "cajero" || repo.user.RoleId != 2 || repo.user.NodeId != nodeID {
					t.Fatalf("wrong user: %+v", repo.user)
				}
				// A redelivered creation must preserve subsequent changes.
				repo.user.RoleId = 7
				redelivery := event.(userevents.UserCreatedDomainEvent)
				redelivery.EventId = uuid.NewV7()
				redelivery.Username = "renamed"
				if err := subscriber.On(context.Background(), redelivery); err != nil {
					t.Fatal(err)
				}
				if repo.lookupID != userID || repo.writes != 1 || repo.user.RoleId != 7 || repo.user.Username != "cajero" {
					t.Fatal("redelivery overwrote existing user")
				}
			}
		})
	}
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
