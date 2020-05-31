/**********************************************************************************
* Copyright (c) 2009-2020 Misakai Ltd.
* This program is free software: you can redistribute it and/or modify it under the
* terms of the GNU Affero General Public License as published by the  Free Software
* Foundation, either version 3 of the License, or(at your option) any later version.
*
* This program is distributed  in the hope that it  will be useful, but WITHOUT ANY
* WARRANTY;  without even  the implied warranty of MERCHANTABILITY or FITNESS FOR A
* PARTICULAR PURPOSE.  See the GNU Affero General Public License  for  more details.
*
* You should have  received a copy  of the  GNU Affero General Public License along
* with this program. If not, see<http://www.gnu.org/licenses/>.
************************************************************************************/

package service

import (
	"github.com/emitter-io/emitter/internal/event"
	"github.com/emitter-io/emitter/internal/message"
	"github.com/emitter-io/emitter/internal/provider/contract"
	"github.com/emitter-io/emitter/internal/security"
	"io"
	"time"
)

// Authorizer service performs authorization checks.
type Authorizer interface {
	Authorize(*security.Channel, uint8) (contract.Contract, security.Key, bool)
}

// PubSub represents a pubsub service.
type PubSub interface {
	Publish(*message.Message, func(message.Subscriber) bool) int64
	Subscribe(message.Subscriber, *event.Subscription) bool
	Unsubscribe(message.Subscriber, *event.Subscription) bool
}

// Handler represents a generic emitter request handler
type Handler interface {
	Type() uint32
	Handle([]byte) (Response, bool)
}

// Response represents an emitter response.
type Response interface {
	ForRequest(uint16)
}

// Surveyee handles the surveys.
type Surveyee interface {
	OnSurvey(string, []byte) ([]byte, bool)
}

//Surveyor issues the surveys.
type Surveyor interface {
	Query(string, []byte) (message.Awaiter, error)
}

// Conn represents a connection interface.
type Conn interface {
	io.Closer
	message.Subscriber
	CanSubscribe(message.Ssid, []byte) bool
	CanUnsubscribe(message.Ssid, []byte) bool
	GetLink([]byte) []byte
	LocalID() security.ID
	Username() string
	MeasureElapsed(string, time.Time)
	Track(contract.Contract)
}
