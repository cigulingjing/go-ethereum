// Copyright 2026 The go-ethereum Authors
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

package main

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
)

func TestParseVersionEventUsesReceiptBlockForImmediateUpgrade(t *testing.T) {
	event := cryptoupgrade.CodeStorageABI.Events["codeVersionUploaded"]
	data, err := event.Inputs.NonIndexed().Pack("add", uint64(2), uint64(7))
	if err != nil {
		t.Fatalf("pack event: %v", err)
	}
	receipt := &types.Receipt{
		BlockNumber: big.NewInt(7),
		Logs: []*types.Log{{
			Address: common.CodeStorageAddress,
			Topics:  []common.Hash{event.ID},
			Data:    data,
		}},
	}
	got := parseVersionEvent(receipt, "Add", 2, 0, true)
	if !got.Consistent || got.Name != "add" || got.Version != 2 || got.ActivationBlock != 7 {
		t.Fatalf("unexpected immediate event result: %#v", got)
	}
}

func TestSummarizeSampleChecksReportsConsistencyFailures(t *testing.T) {
	sample := sampleResult{
		Stage:           "after",
		ExpectedVersion: 2,
		ExpectedOutput:  "205",
		Nodes: []nodeSampleResult{
			{NodeID: "node1", BlockHash: "0x1", ReceiptVisible: true, ReceiptBlockHash: "0xa", Version: 2, Output: "205", OK: true},
			{NodeID: "node2", BlockHash: "0x2", ReceiptVisible: true, ReceiptBlockHash: "0xa", Version: 1, Output: "200", OK: false, Error: "version mismatch"},
		},
	}
	got := summarizeSampleChecks(sample)
	if got.Passed || got.ChainViewConsistent || got.VersionConsistent || got.OutputConsistent {
		t.Fatalf("unexpected passing checks: %#v", got)
	}
	failures := strings.Join(got.Failures, ";")
	for _, want := range []string{"node2: version mismatch", "after: chain view mismatch", "after: version mismatch", "after: output mismatch"} {
		if !strings.Contains(failures, want) {
			t.Fatalf("missing failure %q in %#v", want, got.Failures)
		}
	}
}

func TestRoundVersionsIncreaseAboveLikelyExistingVersions(t *testing.T) {
	oldVersion, newVersion := roundVersions(37, 1)
	if oldVersion <= 1012 || newVersion != oldVersion+1 {
		t.Fatalf("unexpected version pair: old=%d new=%d", oldVersion, newVersion)
	}
}
