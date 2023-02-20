// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"github.com/ethereum/go-ethereum/common"
	qbfttypes "github.com/ethereum/go-ethereum/consensus/istanbul/qbft/types"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/consensus/istanbul"
)

func newTestPreprepare(v *istanbul.View, proposal istanbul.Proposal, source common.Address) qbfttypes.QBFTMessage {
	preprepare := qbfttypes.NewPreprepare(v.Sequence, v.Round, proposal)
	preprepare.SetSource(source)

	return preprepare
}

func TestHandlePreprepare(t *testing.T) {
	N := uint64(4) // replica 0 is the proposer, it will send messages to others
	F := uint64(1) // F does not affect tests

	testCases := []struct {
		system          *testSystem
		expectedRequest istanbul.Proposal
		expectedErr     error
		existingBlock   bool
	}{
		{
			// normal case
			func() *testSystem {
				sys := NewTestSystemWithBackend(N, F)

				for i, backend := range sys.backends {
					c := backend.engine
					c.valSet = backend.peers
					if i != 0 {
						c.state = StateAcceptRequest
					}
				}
				return sys
			}(),
			newTestProposal(),
			nil,
			false,
		},
		{
			// future message
			func() *testSystem {
				sys := NewTestSystemWithBackend(N, F)

				for i, backend := range sys.backends {
					c := backend.engine
					c.valSet = backend.peers
					if i != 0 {
						c.state = StateAcceptRequest
						// hack: force set subject that future message can be simulated
						c.current = newTestRoundState(
							&istanbul.View{
								Round:    big.NewInt(0),
								Sequence: big.NewInt(0),
							},
							c.valSet,
							sys.backends[0].address,
						)

					} else {
						c.current.SetSequence(big.NewInt(10))
					}
				}
				return sys
			}(),
			makeBlock(1),
			errFutureMessage,
			false,
		},
		{
			// errOldMessage
			func() *testSystem {
				sys := NewTestSystemWithBackend(N, F)

				for i, backend := range sys.backends {
					c := backend.engine
					c.valSet = backend.peers
					if i != 0 {
						c.state = StatePreprepared
						c.current.SetSequence(big.NewInt(10))
						c.current.SetRound(big.NewInt(10))
					}
				}
				return sys
			}(),
			makeBlock(1),
			errOldMessage,
			false,
		},
	}

OUTER:
	for _, test := range testCases {
		test.system.Run(false)

		v0 := test.system.backends[0]
		r0 := v0.engine

		curView := r0.currentView()

		for i, v := range test.system.backends {
			// i == 0 is primary backend, it is responsible for send PRE-PREPARE messages to others.
			if i == 0 {
				continue
			}

			c := v.engine

			// run each backends and verify handlePreprepare function.
			targetMessage := newTestPreprepare(
				curView,
				test.expectedRequest,
				v0.address,
			)
			err := c.handleDecodedMessage(targetMessage)
			if err != nil {
				if err != test.expectedErr {
					t.Errorf("error mismatch: have %v, want %v", err, test.expectedErr)
				}
				if err == errFutureMessage {
					if c.backlogs[v0.address].Size() != 1 {
						t.Errorf("backlogs length mismatch: have %v, want %v", c.backlogs[v0.address].Size(), 1)
					}
					rawMsg, priority := c.backlogs[v0.address].Pop()
					msg, ok := rawMsg.(qbfttypes.QBFTMessage)
					if !ok {
						t.Errorf("backlog message is invalid type")
					}
					if priority != toPriority(msg.Code(), curView) {
						t.Errorf("backlogs length mismatch: have %v, want %v", c.backlogs[v0.address].Size(), 1)
					}
					if msg != targetMessage {
						t.Errorf("backlog message mismatch: have %v, want %v", msg, targetMessage)
					}
				}
				continue OUTER
			}

			if c.state != StatePreprepared {
				t.Errorf("state mismatch: have %v, want %v", c.state, StatePreprepared)
			}

			if !test.existingBlock && !reflect.DeepEqual(c.current.Subject().View, curView) {
				t.Errorf("view mismatch: have %v, want %v", c.current.Subject().View, curView)
			}

			// verify prepare messages
			expectedCode := qbfttypes.PrepareCode
			if test.existingBlock {
				expectedCode = qbfttypes.CommitCode
			}
			decodedMsg, err := qbfttypes.Decode(uint64(expectedCode), v.sentMsgs[0])
			if err != nil {
				t.Errorf("error mismatch: have %v, want nil", err)
			}

			if decodedMsg.Code() != uint64(expectedCode) {
				t.Errorf("message code mismatch: have %v, want %v", decodedMsg.Code(), expectedCode)
			}

		}
	}
}

type MockBlock struct {
	*types.Block
}

func (m *MockBlock) Number() *big.Int {
	time.Sleep(time.Second * 2)
	return new(big.Int).Set(m.Header().Number)
}

func TestSendPreprepareMsg(t *testing.T) {
	N := uint64(4) // replica 0 is the proposer, it will send messages to others
	F := uint64(1) // F does not affect tests

	testCases := []struct {
		system                 *testSystem
		request                *Request
		expectedState          State
		expectedPreprepareSent *big.Int
		expectedIsMessageSent  bool
	}{
		{
			// Normal Case
			func() *testSystem {
				sys := NewTestSystemWithBackend(N, F)

				for _, backend := range sys.backends {
					c := backend.engine
					c.valSet = backend.peers
					c.state = StateAcceptRequest
				}
				return sys
			}(),
			&Request{
				Proposal:        makeBlock(1),
				RCMessages:      nil,
				PrepareMessages: nil,
			},
			StateAcceptRequest,
			big.NewInt(0),
			true,
		},
		{
			// Current sequence does not match proposal sequence
			func() *testSystem {
				sys := NewTestSystemWithBackend(N, F)

				for _, backend := range sys.backends {
					c := backend.engine
					c.valSet = backend.peers
					c.state = StateAcceptRequest
				}
				return sys
			}(),
			&Request{
				Proposal:        makeBlock(0),
				RCMessages:      nil,
				PrepareMessages: nil,
			},
			StateAcceptRequest,
			big.NewInt(0),
			false,
		},
		{
			// Current sequence does not match proposal sequence
			// Current sequence is changed concurrently
			func() *testSystem {
				sys := NewTestSystemWithBackend(N, F)

				for _, backend := range sys.backends {
					c := backend.engine
					c.valSet = backend.peers
					c.state = StateAcceptRequest
				}
				return sys
			}(),
			&Request{
				// MockBlock consumes 2secs to return Number() func
				Proposal:        &MockBlock{makeBlock(0)},
				RCMessages:      nil,
				PrepareMessages: nil,
			},
			StateAcceptRequest,
			big.NewInt(0),
			false,
		},
	}

	for _, test := range testCases {
		test.system.Run(false)

		v0 := test.system.backends[0]
		r0 := v0.engine

		// Change current sequence concurrently
		go func() {
			time.Sleep(1)
			r0.current.SetSequence(big.NewInt(1))
		}()

		r0.sendPreprepareMsg(test.request)

		if r0.state != test.expectedState {
			t.Errorf("state mismatch: have %v, want %v", r0.state, test.expectedState)
		}

		if test.expectedIsMessageSent {
			_, err := qbfttypes.Decode(qbfttypes.PreprepareCode, v0.sentMsgs[0])
			if err != nil {
				t.Errorf("error mismatch: have %v, want nil", err)
			}
		}

		if r0.current.preprepareSent.Uint64() != test.expectedPreprepareSent.Uint64() {
			t.Errorf("current preprepareSent mismatch: have %v, want %v", r0.current.preprepareSent, test.expectedPreprepareSent)
		}
	}
}
