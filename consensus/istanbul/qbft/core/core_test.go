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
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/istanbul"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"testing"
)

func makeBlock(number int64) *types.Block {
	header := &types.Header{
		Difficulty: big.NewInt(0),
		Number:     big.NewInt(number),
		GasLimit:   0,
		GasUsed:    0,
		Time:       0,
	}
	block := &types.Block{}
	return block.WithSeal(header)
}

func newTestProposal() istanbul.Proposal {
	return makeBlock(1)
}

func TestQuorumSize(t *testing.T) {
	N := uint64(4)
	F := uint64(1)

	sys := NewTestSystemWithBackend(N, F)
	backend := sys.backends[0]
	c := backend.engine

	valSet := c.valSet
	for i := 1; i <= 1000; i++ {
		valSet.AddValidator(common.StringToAddress(fmt.Sprint(i)))
		if 2*c.QuorumSize() <= (valSet.Size()+valSet.F()) || 2*c.QuorumSize() > (valSet.Size()+valSet.F()+2) {
			t.Errorf("quorumSize constraint failed, expected value (2*QuorumSize > Size+F && 2*QuorumSize <= Size+F+2) to be:%v, got: %v, for size: %v", true, false, valSet.Size())
		}
	}
}
