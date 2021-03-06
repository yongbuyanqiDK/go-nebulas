// Copyright (C) 2017 go-nebulas authors
//
// This file is part of the go-nebulas library.
//
// the go-nebulas library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-nebulas library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-nebulas library.  If not, see <http://www.gnu.org/licenses/>.
//

package net

import (
	"sync"

	"github.com/nebulasio/go-nebulas/util/logging"
	"github.com/sirupsen/logrus"
)

// Dispatcher a message dispatcher service.
type Dispatcher struct {
	subscribersMap    *sync.Map
	quitCh            chan bool
	receivedMessageCh chan Message
}

// NewDispatcher create Dispatcher instance.
func NewDispatcher() *Dispatcher {
	dp := &Dispatcher{
		subscribersMap:    new(sync.Map),
		quitCh:            make(chan bool, 10),
		receivedMessageCh: make(chan Message, 65536),
	}

	return dp
}

// Register register subscribers.
func (dp *Dispatcher) Register(subscribers ...*Subscriber) {
	for _, v := range subscribers {
		for _, mt := range v.msgTypes {
			m, _ := dp.subscribersMap.LoadOrStore(mt, new(sync.Map))
			m.(*sync.Map).Store(v, true)
		}
	}
}

// Deregister deregister subscribers.
func (dp *Dispatcher) Deregister(subscribers ...*Subscriber) {

	for _, v := range subscribers {
		for _, mt := range v.msgTypes {
			m, _ := dp.subscribersMap.Load(mt)
			if m == nil {
				continue
			}
			m.(*sync.Map).Delete(v)
			dp.subscribersMap.Delete(mt)
		}
	}
}

// Start start message dispatch goroutine.
func (dp *Dispatcher) Start() {
	logging.CLog().Info("Starting NetService Dispatcher...")

	go (func() {
		for {
			select {
			case <-dp.quitCh:
				logging.CLog().Info("Stoping NetService Dispatcher...")
				return

			case msg := <-dp.receivedMessageCh:
				msgType := msg.MessageType()
				logging.VLog().WithFields(logrus.Fields{
					"msgType": msgType,
				}).Info("dispatcher received message")

				v, _ := dp.subscribersMap.Load(msgType)
				m, _ := v.(*sync.Map)

				m.Range(func(key, value interface{}) bool {
					key.(*Subscriber).msgChan <- msg
					logging.VLog().WithFields(logrus.Fields{
						"msgType": msgType,
					}).Info("succeed dispatcher received message")
					return true
				})
			}
		}
	})()
}

// Stop stop goroutine.
func (dp *Dispatcher) Stop() {
	logging.CLog().Info("Stopping NetService Dispatcher...")

	dp.quitCh <- true
}

// PutMessage put new message to chan, then subscribers will be notified to process.
func (dp *Dispatcher) PutMessage(msg Message) {
	dp.receivedMessageCh <- msg
}
