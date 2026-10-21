package workerfx

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/cronos/keno4min/internal/common"
)

type DomainEventJSONDeserializer struct {
	eventTypes map[string]reflect.Type
}

func NewDomainEventJSONDeserializer(prototypes []common.DomainEvent) (*DomainEventJSONDeserializer, error) {
	eventTypes := make(map[string]reflect.Type, len(prototypes))
	for _, prototype := range prototypes {
		t := reflect.TypeOf(prototype)
		eventTypes[prototype.Type()] = t
	}
	return &DomainEventJSONDeserializer{eventTypes: eventTypes}, nil
}

func (d *DomainEventJSONDeserializer) resolve(name string) (reflect.Type, bool) {
	if t, exists := d.eventTypes[name]; exists {
		return t, true
	}
	var selected reflect.Type
	bestLength := -1
	for pattern, t := range d.eventTypes {
		if !strings.HasSuffix(pattern, "*") {
			continue
		}
		prefix := strings.TrimSuffix(pattern, "*")
		if strings.HasPrefix(name, prefix) && len(prefix) > bestLength {
			selected, bestLength = t, len(prefix)
		}
	}
	return selected, selected != nil
}

func (d *DomainEventJSONDeserializer) Deserialize(body []byte) (common.DomainEvent, error) {
	var envelope struct {
		DetailType string          `json:"detail-type"`
		Detail     json.RawMessage `json:"detail"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode event envelope: %w", err)
	}
	detailType := envelope.DetailType
	t, found := d.resolve(detailType)
	if !found {
		return nil, fmt.Errorf("no event prototype matches %q", detailType)
	}
	target := reflect.New(t)
	if err := json.Unmarshal(envelope.Detail, target.Interface()); err != nil {
		return nil, fmt.Errorf("decode event %q: %w", detailType, err)
	}
	event, ok := target.Elem().Interface().(common.DomainEvent)
	if !ok {
		return nil, fmt.Errorf("%v does not implement DomainEvent", t)
	}
	if event.Type() != detailType {
		return nil, fmt.Errorf("decoded event type %q does not match envelope %q", event.Type(), detailType)
	}
	return event, nil
}
