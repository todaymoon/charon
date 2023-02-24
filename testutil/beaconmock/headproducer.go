// Copyright © 2022 Obol Labs Inc.
//
// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of  MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for
// more details.
//
// You should have received a copy of the GNU General Public License along with
// this program.  If not, see <http://www.gnu.org/licenses/>.

package beaconmock

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	eth2p0 "github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/gorilla/mux"
	"github.com/r3labs/sse/v2"

	"github.com/obolnetwork/charon/app/log"
	"github.com/obolnetwork/charon/app/z"
)

const (
	topicHead  = "head"
	topicBlock = "block"
)

func newHeadProducer() *headProducer {
	return &headProducer{
		server:         sse.New(),
		streamsByTopic: make(map[string][]string),
		quit:           make(chan struct{}),
	}
}

// headProducer is a stateful struct for providing deterministic block roots based on slot events.
type headProducer struct {
	// Immutable state
	server *sse.Server
	quit   chan struct{}

	// Mutable state
	mu             sync.Mutex
	currentHead    *eth2v1.HeadEvent
	streamsByTopic map[string][]string
}

// Start starts the internal slot ticker that updates head.
func (p *headProducer) Start(httpMock HTTPMock) error {
	ctx := context.Background()
	genesisTime, err := httpMock.GenesisTime(ctx)
	if err != nil {
		return err
	}
	slotDuration, err := httpMock.SlotDuration(ctx)
	if err != nil {
		return err
	}

	startSlotTicker(p.quit, p.updateHead, genesisTime, slotDuration)

	return nil
}

func (p *headProducer) Close() {
	close(p.quit)
}

func (p *headProducer) Handlers() map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"/eth/v1/events":                        p.handleEvents,
		"/eth/v1/beacon/blocks/{block_id}/root": p.handleGetBlockRoot,
	}
}

func (p *headProducer) getCurrentHead() *eth2v1.HeadEvent {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.currentHead
}

func (p *headProducer) setCurrentHead(currentHead *eth2v1.HeadEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.currentHead = currentHead
}

func (p *headProducer) getStreamIDs(topic string) []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.streamsByTopic[topic]
}

func (p *headProducer) setStreamIDs(topic string, streamID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.streamsByTopic[topic] = append(p.streamsByTopic[topic], streamID)
}

// updateHead updates current head based on provided slot.
func (p *headProducer) updateHead(slot eth2p0.Slot) {
	currentHead := pseudoRandomHeadEvent(slot)
	p.setCurrentHead(currentHead)

	currentBlock := &eth2v1.BlockEvent{
		Slot:  slot,
		Block: currentHead.Block,
	}

	headData, err := json.Marshal(currentHead)
	if err != nil {
		panic(err) // This should never happen and this is test code sorry ;)
	}

	blockData, err := json.Marshal(currentBlock)
	if err != nil {
		panic(err) // This should never happen and this is test code sorry ;)
	}

	// Publish head events.
	for _, streamID := range p.getStreamIDs(topicHead) {
		p.server.Publish(streamID, &sse.Event{
			Event: []byte(topicHead),
			Data:  headData,
		})
	}

	// Publish block events.
	for _, streamID := range p.getStreamIDs(topicBlock) {
		p.server.Publish(streamID, &sse.Event{
			Event: []byte(topicBlock),
			Data:  blockData,
		})
	}
}

type getBlockRootResponseJSON struct {
	ExecutionOptimistic bool                    `json:"execution_optimistic"`
	Data                beaconBlockRootDataJSON `json:"data"`
}

type beaconBlockRootDataJSON struct {
	Root string `json:"root"`
}

type errorMsgJSON struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// handleGetBlockRoot is an http handler to handle "/eth/v1/beacon/blocks/{block_id}/root" endpoint.
func (p *headProducer) handleGetBlockRoot(w http.ResponseWriter, r *http.Request) {
	head := p.getCurrentHead()

	if head == nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp, err := json.Marshal(errorMsgJSON{
			Code:    500,
			Message: "Head producer not ready",
		})
		if err != nil {
			panic(err) // This should never happen and this is test code sorry ;)
		}
		_, _ = w.Write(resp)

		return
	}

	blockID := mux.Vars(r)["block_id"]
	if blockID != "head" && blockID != fmt.Sprint(head.Slot) {
		w.WriteHeader(http.StatusBadRequest)
		resp, err := json.Marshal(errorMsgJSON{
			Code:    500,
			Message: fmt.Sprintf("Invalid block ID: %s", blockID),
		})
		if err != nil {
			panic(err) // This should never happen and this is test code sorry ;)
		}
		_, _ = w.Write(resp)

		return
	}

	resp, err := json.Marshal(getBlockRootResponseJSON{
		ExecutionOptimistic: false,
		Data: beaconBlockRootDataJSON{
			Root: fmt.Sprintf("%#x", head.Block),
		},
	})
	if err != nil {
		panic(err) // This should never happen and this is test code sorry ;)
	}

	_, _ = w.Write(resp)
}

// handleEvents is a http handler to handle "/eth/v1/events".
func (p *headProducer) handleEvents(w http.ResponseWriter, r *http.Request) {
	//nolint:gosec
	streamID := strconv.Itoa(rand.Int())
	p.server.CreateStream(streamID)

	query := r.URL.Query()
	query.Set("stream", streamID) // Add sseStreamID for sse server to serve events on.
	r.URL.RawQuery = query.Encode()

	for _, topic := range query["topics"] {
		if topic != topicHead && topic != topicBlock {
			log.Warn(context.Background(), "Unsupported topic requested", nil, z.Str("topic", topic))
			w.WriteHeader(http.StatusInternalServerError)
			resp, err := json.Marshal(errorMsgJSON{
				Code:    500,
				Message: "unknown topic",
			})
			if err != nil {
				panic(err) // This should never happen and this is test code sorry ;)
			}
			_, _ = w.Write(resp)

			return
		}

		p.setStreamIDs(topic, streamID)
	}

	p.server.ServeHTTP(w, r)
}

// startSlotTicker returns a blocking channel that will be populated with new slots in real time.
// It is also populated with the current slot immediately.
func startSlotTicker(quit chan struct{}, callback func(eth2p0.Slot), genesisTime time.Time, slotDuration time.Duration) {
	chainAge := time.Since(genesisTime)
	height := int64(chainAge / slotDuration)
	startTime := genesisTime.Add(time.Duration(height) * slotDuration)

	go func() {
		for {
			callback(eth2p0.Slot(height))

			height++
			startTime = startTime.Add(slotDuration)
			delay := time.Until(startTime)

			select {
			case <-quit:
				return
			case <-time.After(delay):
			}
		}
	}()
}

func pseudoRandomHeadEvent(slot eth2p0.Slot) *eth2v1.HeadEvent {
	r := rand.New(rand.NewSource(int64(slot))) //nolint:gosec

	root := func() eth2p0.Root {
		var root eth2p0.Root
		_, _ = r.Read(root[:])

		return root
	}

	return &eth2v1.HeadEvent{
		Slot:                      slot,
		Block:                     root(),
		State:                     root(),
		EpochTransition:           false,
		CurrentDutyDependentRoot:  root(),
		PreviousDutyDependentRoot: root(),
	}
}
