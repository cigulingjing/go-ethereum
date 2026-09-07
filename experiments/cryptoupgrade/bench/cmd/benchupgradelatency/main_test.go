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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/network"
)

func TestSelectSenderAndTargets(t *testing.T) {
	nodes := []network.NodeConfig{
		{ID: "node1", Role: "signer", Account: "0x1111111111111111111111111111111111111111"},
		{ID: "node2", Role: "observer"},
		{ID: "node3", Role: "rpc"},
	}
	sender, err := selectSender(nodes, "")
	if err != nil {
		t.Fatalf("selectSender default failed: %v", err)
	}
	if sender.ID != "node1" {
		t.Fatalf("default sender = %s, want node1", sender.ID)
	}
	targets, err := selectTargetNodes(nodes, "node2,node3")
	if err != nil {
		t.Fatalf("selectTargetNodes failed: %v", err)
	}
	if len(targets) != 2 || targets[0].ID != "node2" || targets[1].ID != "node3" {
		t.Fatalf("unexpected target selection: %#v", targets)
	}
	if _, err := selectTargetNodes(nodes, "node2,node2"); err == nil {
		t.Fatal("duplicate target node was accepted")
	}
	if _, err := selectTargetNodes(nodes, "missing"); err == nil {
		t.Fatal("missing target node was accepted")
	}
}

func TestRoundAlgorithmNameAndGeneratedSource(t *testing.T) {
	if got := roundAlgorithmName("Add", 1, 1); got != "Add" {
		t.Fatalf("single round algorithm = %s, want Add", got)
	}
	if got := roundAlgorithmName("Add", 2, 3); got != "AddLatency002" {
		t.Fatalf("multi-round algorithm = %s, want AddLatency002", got)
	}
	dir := t.TempDir()
	repoRoot := t.TempDir()
	sourceDir := filepath.Join(repoRoot, "cryptoupgrade", "algorithm", "go")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(sourceDir, "add.go")
	if err := os.WriteFile(sourcePath, []byte("package main\n\nimport \"math/big\"\n\nfunc Add(a *big.Int, b *big.Int) *big.Int { return new(big.Int).Add(a, b) }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := config{
		repoRoot:    repoRoot,
		resultDir:   dir,
		sourcePath:  sourcePath,
		algorithm:   "Add",
		uniqueNames: true,
		rounds:      2,
	}
	fixture := algorithmFixture{
		Algorithm:   "Add",
		UpgradeName: "Add",
		SourceFunc:  "Add",
		SourcePath:  sourcePath,
		InputTypes:  []string{"uint256", "uint256"},
		OutputTypes: []string{"uint256"},
		AlgoGas:     1,
	}
	source, err := prepareRoundSource(cfg, 1, 1, fixture)
	if err != nil {
		t.Fatalf("prepareRoundSource failed: %v", err)
	}
	if source.Algorithm != "Add" || source.UpgradeName != "AddLatency001" {
		t.Fatalf("source names = %s/%s, want Add/AddLatency001", source.Algorithm, source.UpgradeName)
	}
	if _, err := os.Stat(source.Path); err != nil {
		t.Fatalf("generated source missing: %v", err)
	}
	if !strings.Contains(string(source.Raw), "func AddLatency001") {
		t.Fatalf("generated source does not export AddLatency001:\n%s", string(source.Raw))
	}
}

func TestBuildPayloadStats(t *testing.T) {
	requireTinyGo(t)
	fixture := algorithmFixture{
		InputTypes:  []string{"uint256", "uint256"},
		OutputTypes: []string{"uint256"},
		AlgoGas:     1,
	}
	source := []byte("package main\n\nimport \"math/big\"\n\nfunc Add(a *big.Int, b *big.Int) *big.Int { return new(big.Int).Add(a, b) }\n")
	payload, err := buildPayload(source, "Add", fixture)
	if err != nil {
		t.Fatalf("buildPayload failed: %v", err)
	}
	if payload.Result.SourceBytes != len(source) {
		t.Fatalf("source bytes = %d, want %d", payload.Result.SourceBytes, len(source))
	}
	if payload.Result.CompressedGzipBytes == 0 || payload.Result.CompressedBase64Bytes == 0 {
		t.Fatalf("compressed sizes not populated: %#v", payload.Result)
	}
	if payload.Result.UploadCalldataBytes != len(payload.Upload) {
		t.Fatalf("calldata bytes = %d, want %d", payload.Result.UploadCalldataBytes, len(payload.Upload))
	}
	if payload.Result.UploadCalldataSHA256 == "" {
		t.Fatal("upload calldata hash is empty")
	}
	if payload.Result.WasmHash == "" || payload.Result.Version != 1 || payload.Result.ActivationBlock != 0 {
		t.Fatalf("wasm identity fields not populated: %#v", payload.Result)
	}
}

func requireTinyGo(t *testing.T) {
	t.Helper()
	if tinygo := os.Getenv("TINYGO"); tinygo != "" {
		return
	}
	if _, err := exec.LookPath("tinygo"); err != nil {
		t.Skip("tinygo not available")
	}
}

func TestBuiltinFixturesAndValidationProbes(t *testing.T) {
	cfg := config{a: "100", b: "100"}
	fixtures, err := builtinFixtures(cfg)
	if err != nil {
		t.Fatalf("builtinFixtures failed: %v", err)
	}
	if len(fixtures) != 7 {
		t.Fatalf("fixture count = %d, want 7", len(fixtures))
	}
	for _, fixture := range fixtures {
		probe, inputArgs, outputArgs, err := buildValidationProbe(fixture.UpgradeName, fixture.InputTypes, fixture.OutputTypes, fixture.Values, fixture.ExpectedValues)
		if err != nil {
			t.Fatalf("%s probe failed: %v", fixture.Algorithm, err)
		}
		if len(probe.EncodedInput) == 0 || len(probe.CallData) == 0 || len(probe.ExpectedRaw) == 0 {
			t.Fatalf("%s probe did not populate encoded data", fixture.Algorithm)
		}
		if len(inputArgs) != len(fixture.Values) {
			t.Fatalf("%s input args = %d, want %d", fixture.Algorithm, len(inputArgs), len(fixture.Values))
		}
		if len(outputArgs) != len(fixture.ExpectedValues) {
			t.Fatalf("%s output args = %d, want %d", fixture.Algorithm, len(outputArgs), len(fixture.ExpectedValues))
		}
		if probe.ExpectedText == "" {
			t.Fatalf("%s expected text is empty", fixture.Algorithm)
		}
	}
}

func TestGenericOutputFormatting(t *testing.T) {
	if got := formatABIValues([]interface{}{big.NewInt(200)}); got != "200" {
		t.Fatalf("number format = %s", got)
	}
	if got := formatABIValues([]interface{}{[]byte{0xaa, 0xbb}}); got != "0xaabb" {
		t.Fatalf("bytes format = %s", got)
	}
	if got := formatABIValues([]interface{}{true}); got != "true" {
		t.Fatalf("bool format = %s", got)
	}
}

func TestSummaries(t *testing.T) {
	start := time.Now()
	txHashReturned := start.Add(20 * time.Millisecond)
	senderReceipt := start.Add(120 * time.Millisecond)
	eventObserved := start.Add(125 * time.Millisecond)
	nodes := []nodeRoundResult{
		{
			ID:                     "node1",
			Completed:              true,
			receiptTime:            start.Add(100 * time.Millisecond),
			completeTime:           start.Add(200 * time.Millisecond),
			SubmitToReceiptMillis:  100,
			SubmitToCompleteMillis: 200,
		},
		{
			ID:                     "node2",
			Completed:              true,
			receiptTime:            start.Add(150 * time.Millisecond),
			completeTime:           start.Add(350 * time.Millisecond),
			SubmitToReceiptMillis:  150,
			SubmitToCompleteMillis: 350,
		},
	}
	annotateNodeStages(nodes, txHashReturned, senderReceipt, eventObserved)
	if nodes[1].TxHashToReceiptMillis != 130 {
		t.Fatalf("node tx-hash-to-receipt = %.3f, want 130", nodes[1].TxHashToReceiptMillis)
	}
	if nodes[1].SenderReceiptToCompleteMS != 230 {
		t.Fatalf("node sender-receipt-to-complete = %.3f, want 230", nodes[1].SenderReceiptToCompleteMS)
	}
	if nodes[1].EventToCompleteMillis != 225 {
		t.Fatalf("node event-to-complete = %.3f, want 225", nodes[1].EventToCompleteMillis)
	}
	roundSummary := summarizeRound(nodes, start, txHashReturned, senderReceipt, eventObserved)
	if !roundSummary.Completed {
		t.Fatal("round summary should be completed")
	}
	if roundSummary.SlowestNodeID != "node2" {
		t.Fatalf("slowest node = %s, want node2", roundSummary.SlowestNodeID)
	}
	if roundSummary.CompletedNodeCount != 2 {
		t.Fatalf("completed nodes = %d, want 2", roundSummary.CompletedNodeCount)
	}
	if roundSummary.SubmitToAllCompleteMillis <= roundSummary.SubmitToAllReceiptMillis {
		t.Fatalf("all-complete latency should exceed all-receipt latency: %#v", roundSummary)
	}
	if roundSummary.SubmitToTxHashMillis != 20 {
		t.Fatalf("submit-to-tx-hash = %.3f, want 20", roundSummary.SubmitToTxHashMillis)
	}
	if roundSummary.TxHashToSenderReceiptMillis != 100 {
		t.Fatalf("tx-hash-to-sender-receipt = %.3f, want 100", roundSummary.TxHashToSenderReceiptMillis)
	}
	if roundSummary.SenderReceiptToAllReceiptMillis != 30 {
		t.Fatalf("sender-receipt-to-all-receipt = %.3f, want 30", roundSummary.SenderReceiptToAllReceiptMillis)
	}
	if roundSummary.AllReceiptToAllCompleteMillis != 200 {
		t.Fatalf("all-receipt-to-all-complete = %.3f, want 200", roundSummary.AllReceiptToAllCompleteMillis)
	}
	if roundSummary.ReceiptToEventMillis != 5 {
		t.Fatalf("receipt-to-event = %.3f, want 5", roundSummary.ReceiptToEventMillis)
	}
	if roundSummary.EventToAllCompleteMillis != 225 {
		t.Fatalf("event-to-all-complete = %.3f, want 225", roundSummary.EventToAllCompleteMillis)
	}
	if roundSummary.MeanNodeSubmitToReceiptMillis != 125 {
		t.Fatalf("mean submit-to-receipt = %.3f, want 125", roundSummary.MeanNodeSubmitToReceiptMillis)
	}
	if roundSummary.MeanNodeTxHashToReceiptMillis != 105 {
		t.Fatalf("mean tx-hash-to-receipt = %.3f, want 105", roundSummary.MeanNodeTxHashToReceiptMillis)
	}
	if roundSummary.MeanNodeSenderReceiptToReceiptMS != 5 {
		t.Fatalf("mean sender-receipt-to-receipt = %.3f, want 5", roundSummary.MeanNodeSenderReceiptToReceiptMS)
	}
	if roundSummary.MeanNodeReceiptToCompleteMillis != 150 {
		t.Fatalf("mean receipt-to-complete = %.3f, want 150", roundSummary.MeanNodeReceiptToCompleteMillis)
	}
	if roundSummary.MeanNodeSubmitToCompleteMillis != 275 {
		t.Fatalf("mean submit-to-complete = %.3f, want 275", roundSummary.MeanNodeSubmitToCompleteMillis)
	}
	if roundSummary.MeanNodeTxHashToCompleteMillis != 255 {
		t.Fatalf("mean tx-hash-to-complete = %.3f, want 255", roundSummary.MeanNodeTxHashToCompleteMillis)
	}
	if roundSummary.MeanNodeSenderReceiptToCompleteMS != 155 {
		t.Fatalf("mean sender-receipt-to-complete = %.3f, want 155", roundSummary.MeanNodeSenderReceiptToCompleteMS)
	}
	timeline := buildUpgradeTimeline(start, txHashReturned, senderReceipt, eventObserved, nodes)
	if timeline.TransactionSubmittedAt == "" || timeline.TxHashObservedAt == "" || timeline.ReceiptObservedAt == "" || timeline.EventObservedAt == "" || timeline.ActivationObservedAt == "" {
		t.Fatalf("timeline missing phase timestamp: %#v", timeline)
	}
	experimentSummary := summarizeExperiment([]roundResult{{Completed: true, Summary: roundSummary}}, 2)
	if !experimentSummary.Completed || experimentSummary.CompletedRoundCount != 1 {
		t.Fatalf("unexpected experiment summary: %#v", experimentSummary)
	}
}

func TestSelectNetworkUsesExplicitFrom(t *testing.T) {
	cfg := &network.Config{Nodes: []network.NodeConfig{
		{ID: "node1", Role: "signer", Account: "0x1111111111111111111111111111111111111111"},
		{ID: "node2", Role: "observer"},
	}}
	from := "0x2222222222222222222222222222222222222222"
	selected, err := selectNetwork(cfg, "node2", "node2", from)
	if err != nil {
		t.Fatalf("selectNetwork failed: %v", err)
	}
	if selected.sender.ID != "node2" {
		t.Fatalf("sender = %s, want node2", selected.sender.ID)
	}
	if selected.from != common.HexToAddress(from) {
		t.Fatalf("from = %s, want %s", selected.from.Hex(), common.HexToAddress(from).Hex())
	}
}
