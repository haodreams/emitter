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

package presence

import (
	"encoding/json"
	"github.com/emitter-io/emitter/internal/errors"
	"github.com/emitter-io/emitter/internal/event"
	"github.com/kelindar/binary/nocopy"
	"net/http"
	"strings"
	"time"

	"github.com/emitter-io/emitter/internal/message"
	"github.com/emitter-io/emitter/internal/security"
	"github.com/emitter-io/emitter/internal/service"
)

// Notify sends out an event to notify when a client is subscribed/unsubscribed.
func (s *Service) Notify(ev *Notification, filter func(message.Subscriber) bool) {
	channel := []byte("emitter/presence/") // TODO: avoid allocation
	if encoded, ok := ev.Encode(); ok {
		s.pubsub.Publish(message.New(ev.Ssid, channel, encoded), filter)
	}
}

// ------------------------------------------------------------------------------------

// OnPresence processes a presence request.
func (s *Service) OnPresence(c service.Conn, payload []byte) (service.Response, bool) {
	msg := Request{
		Status:  true, // Default: send status info
		Changes: nil,  // Default: send all changes
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return errors.ErrBadRequest, false
	}

	// Ensure we have trailing slash
	if !strings.HasSuffix(msg.Channel, "/") {
		msg.Channel = msg.Channel + "/"
	}

	// Parse the channel
	channel := security.ParseChannel([]byte(msg.Key + "/" + msg.Channel))
	if channel.ChannelType == security.ChannelInvalid {
		return errors.ErrBadRequest, false
	}

	// Check the authorization and permissions
	_, key, allowed := s.auth.Authorize(channel, security.AllowPresence)
	if !allowed {
		return errors.ErrUnauthorized, false
	}

	// Keys which are supposed to be extended should not be used for presence
	if key.HasPermission(security.AllowExtend) {
		return errors.ErrUnauthorizedExt, false
	}

	// Create the ssid for the presence
	ssid := message.NewSsid(key.Contract(), channel.Query)

	// Check if the client is interested in subscribing/unsubscribing from changes.
	if msg.Changes != nil {
		ev := &event.Subscription{
			Conn:    c.LocalID(),
			User:    nocopy.String(c.Username()),
			Ssid:    message.NewSsidForPresence(ssid),
			Channel: channel.Channel,
		}

		switch *msg.Changes {
		case true:
			s.pubsub.Subscribe(c, ev)
		case false:
			s.pubsub.Unsubscribe(c, ev)
		}
	}

	// If we requested a status, populate the slice via scatter/gather.
	now := time.Now().UTC().Unix()
	who := make([]Info, 0, 4)
	if msg.Status {

		// Gather local & cluster presence
		who = append(who, s.getAllPresence(ssid)...)
		return &Response{
			Time:    now,
			Event:   EventTypeStatus,
			Channel: msg.Channel,
			Who:     who,
		}, true
	}
	return nil, true
}

// OnHTTPPresence occurs when a new HTTP presence request is received.
func (s *Service) OnHTTPPresence(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Deserialize the body.
	msg := Request{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&msg)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Ensure we have trailing slash
	if !strings.HasSuffix(msg.Channel, "/") {
		msg.Channel = msg.Channel + "/"
	}

	// Parse the channel
	channel := security.ParseChannel([]byte("emitter/" + msg.Channel))
	if channel.ChannelType == security.ChannelInvalid {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check the authorization and permissions
	_, key, allowed := s.auth.Authorize(channel, security.AllowPresence)
	if !allowed {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Create the ssid for the presence
	ssid := message.NewSsid(key.Contract(), channel.Query)
	now := time.Now().UTC().Unix()
	who := s.getAllPresence(ssid)
	resp, err := json.Marshal(&Response{
		Time:    now,
		Event:   EventTypeStatus,
		Channel: msg.Channel,
		Who:     who,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(resp)
	return
}
