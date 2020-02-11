// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	federation "github.com/networkservicemesh/networkservicemesh/applications/federation-server/api"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"
	"github.com/pkg/errors"
	"github.com/spiffe/spire/proto/spire/common"
)

type event struct {
	monitor.BaseEvent
}

func (e *event) Message() (interface{}, error) {
	eventType, err := eventTypeToBundleEventType(e.EventType())
	if err != nil {
		return nil, err
	}

	bundles, err := bundlesFromEntities(e.Entities())
	if err != nil {
		return nil, err
	}

	return &federation.BundleEvent{
		Type:    eventType,
		Bundles: bundles,
	}, nil
}

type eventFactory struct {
	factoryName string
}

func (m *eventFactory) FactoryName() string {
	return m.factoryName
}

func (m *eventFactory) NewEvent(ctx context.Context, eventType monitor.EventType, entities map[string]monitor.Entity) monitor.Event {
	return &event{
		BaseEvent: monitor.NewBaseEvent(ctx, eventType, entities),
	}
}

func (m *eventFactory) EventFromMessage(ctx context.Context, message interface{}) (monitor.Event, error) {
	e, ok := message.(*federation.BundleEvent)
	if !ok {
		return nil, errors.Errorf("unable to cast %v to local.ConnectionEvent", message)
	}

	eventType, err := bundleEventTypeToEventType(e.GetType())
	if err != nil {
		return nil, err
	}

	entities := entitiesFromBundles(e.Bundles)

	return &event{
		BaseEvent: monitor.NewBaseEvent(ctx, eventType, entities),
	}, nil
}

func eventTypeToBundleEventType(eventType monitor.EventType) (federation.BundleEventType, error) {
	switch eventType {
	case monitor.EventTypeInitialStateTransfer:
		return federation.BundleEventType_INITIAL_STATE_TRANSFER, nil
	case monitor.EventTypeUpdate:
		return federation.BundleEventType_UPDATE, nil
	case monitor.EventTypeDelete:
		return federation.BundleEventType_DELETE, nil
	default:
		return 0, errors.Errorf("unable to cast %v to federation.BundleEventType", eventType)
	}
}

func bundleEventTypeToEventType(eventType federation.BundleEventType) (monitor.EventType, error) {
	switch eventType {
	case federation.BundleEventType_INITIAL_STATE_TRANSFER:
		return monitor.EventTypeInitialStateTransfer, nil
	case federation.BundleEventType_UPDATE:
		return monitor.EventTypeUpdate, nil
	case federation.BundleEventType_DELETE:
		return monitor.EventTypeDelete, nil
	default:
		return "", errors.Errorf("unable to cast %v to monitor.EventType", eventType)
	}
}

func bundlesFromEntities(entities map[string]monitor.Entity) (map[string]*common.Bundle, error) {
	bundles := map[string]*common.Bundle{}

	for k, v := range entities {
		if b, ok := v.(*bundleEntity); ok {
			bundles[k] = b.Bundle
		} else {
			return nil, errors.New("unable to cast Entity to Bundle")
		}
	}

	return bundles, nil
}

func entitiesFromBundles(bundles map[string]*common.Bundle) map[string]monitor.Entity {
	entities := map[string]monitor.Entity{}

	for k, v := range bundles {
		entities[k] = &bundleEntity{v}
	}

	return entities
}
