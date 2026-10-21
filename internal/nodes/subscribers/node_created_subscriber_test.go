package subscribers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	"github.com/cronos/keno4min/internal/nodes"
	"github.com/cronos/keno4min/internal/nodes/events"
	"github.com/cronos/keno4min/internal/nodes/usecase"
)

type recordingRepository struct {
	node *nodes.Node
	err  error
}

func (r *recordingRepository) FindById(context.Context, uuid.UUID) (*nodes.Node, error) {
	return nil, nil
}

func (r *recordingRepository) Save(_ context.Context, node *nodes.Node) error {
	r.node = node
	return r.err
}

func TestNodeCreatedMapsNodeIdentityAndLocation(t *testing.T) {
	var event events.NodeCreatedDomainEvent
	err := json.Unmarshal([]byte(`{
		"eventId":"afed5b77-1e8a-471f-9398-489784a1b897",
		"nodeId":"f2a0dc45-9ab8-4553-bd7c-2d53b345a5f6",
		"name":"admin_test", "address":"test_address", "timeZone":"America/Bogota"
	}`), &event)
	if err != nil {
		t.Fatal(err)
	}
	repository := &recordingRepository{}
	subscriber := NewNodeCretedSubscriber(usecase.NewCreate(repository))
	if err := subscriber.On(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if repository.node.ID != event.NodeId || repository.node.TimeZone != event.TimeZone || repository.node.Address != event.Address {
		t.Fatalf("wrong node identity or location: %+v", repository.node)
	}
	failure := errors.New("save failed")
	repository.err = failure
	if err := subscriber.On(context.Background(), event); !errors.Is(err, failure) {
		t.Fatalf("save failure must reach consumer: %v", err)
	}
}
