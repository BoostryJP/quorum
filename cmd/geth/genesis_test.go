// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cespare/cp"
)

var customGenesisTests = []struct {
	genesis string
	query   string
	result  string
}{
	// Genesis file with an empty chain configuration (ensure missing fields work)
	{
		genesis: `{
			"alloc"      : {},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000001338",
			"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config"     : { "isQuorum":false }
		}`,
		query:  "eth.getBlock(0).nonce",
		result: "0x0000000000001338",
	},
	// Genesis file with specific chain configurations
	{
		genesis: `{
			"alloc"      : {},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000000042",
			"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config"     : {
				"homesteadBlock" : 42,
				"daoForkBlock"   : 141,
				"daoForkSupport" : true,
				"isQuorum" : false
			},
		}`,
		query:  "eth.getBlock(0).nonce",
		result: "0x0000000000000042",
	},
}

// Tests that initializing Geth with a custom genesis block and chain definitions
// work properly.
func TestCustomGenesis(t *testing.T) {
	defer SetResetPrivateConfig("ignore")()
	for i, tt := range customGenesisTests {
		// Create a temporary data directory to use and inspect later
		datadir := tmpdir(t)
		defer os.RemoveAll(datadir)

		// copy the node key and static-nodes.json so that geth can start with the raft consensus
		gethDir := filepath.Join(datadir, "geth")
		sourceNodeKey := filepath.Join("testdata", "geth")
		if err := cp.CopyAll(gethDir, sourceNodeKey); err != nil {
			t.Fatal(err)
		}

		// Initialize the data directory with the custom genesis block
		json := filepath.Join(datadir, "genesis.json")
		if err := os.WriteFile(json, []byte(tt.genesis), 0600); err != nil {
			t.Fatalf("test %d: failed to write genesis file: %v", i, err)
		}
		runGeth(t, "--datadir", datadir, "init", json).WaitExit()

		// Query the custom genesis block
		geth := runGeth(t, "--networkid", "1337", "--syncmode=full",
			"--datadir", datadir, "--maxpeers", "0", "--port", "0",
			"--nodiscover", "--nat", "none", "--ipcdisable",
			"--raft",
			"--exec", tt.query, "console")
		geth.ExpectRegexp(tt.result)
		geth.ExpectExit()
	}
}

func TestCustomGenesisUpgradeWithPrivacyEnhancementsBlock(t *testing.T) {
	defer SetResetPrivateConfig("ignore")()
	// Create a temporary data directory to use and inspect later
	datadir := tmpdir(t)
	defer os.RemoveAll(datadir)

	// copy the node key and static-nodes.json so that geth can start with the raft consensus
	gethDir := filepath.Join(datadir, "geth")
	sourceNodeKey := filepath.Join("testdata", "geth")
	if err := cp.CopyAll(gethDir, sourceNodeKey); err != nil {
		t.Fatal(err)
	}

	genesisContentWithoutPrivacyEnhancements :=
		`{
			"alloc"      : {},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000000042",
			"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config"     : {
				"homesteadBlock" : 42,
				"daoForkBlock"   : 141,
				"daoForkSupport" : true,
				"isQuorum" : false
			}
		}`

	// Initialize the data directory with the custom genesis block
	json := filepath.Join(datadir, "genesis.json")
	if err := os.WriteFile(json, []byte(genesisContentWithoutPrivacyEnhancements), 0600); err != nil {
		t.Fatalf("failed to write genesis file: %v", err)
	}
	geth := runGeth(t, "--datadir", datadir, "init", json)
	geth.WaitExit()

	genesisContentWithPrivacyEnhancements :=
		`{
			"alloc"      : {},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000000042",
			"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config"     : {
				"homesteadBlock" : 42,
				"daoForkBlock"   : 141,
				"privacyEnhancementsBlock"   : 1000,
				"daoForkSupport" : true,
				"isQuorum" : false
			}
		}`

	if err := os.WriteFile(json, []byte(genesisContentWithPrivacyEnhancements), 0600); err != nil {
		t.Fatalf("failed to write genesis file: %v", err)
	}
	geth = runGeth(t, "--datadir", datadir, "init", json)
	geth.WaitExit()

	expectedText := "Privacy enhancements have been enabled from block height 1000. Please ensure your privacy manager is upgraded and supports privacy enhancements"

	result := strings.TrimSpace(geth.StderrText())
	if !strings.Contains(result, expectedText) {
		geth.Fatalf("bad stderr text. want '%s', got '%s'", expectedText, result)
	}

	// start quorum - it should fail the transaction manager PrivacyEnhancements feature validation
	geth = runGeth(t,
		"--datadir", datadir, "--maxpeers", "0", "--port", "0",
		"--nodiscover", "--nat", "none", "--ipcdisable",
		"--raft", "console")
	geth.ExpectRegexp("Cannot start quorum with privacy enhancements enabled while the transaction manager does not support it")
	geth.ExpectExit()
}
