package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Packet struct {
	SchemaVersion    string         `yaml:"schema_version"`
	PacketID         string         `yaml:"packet_id"`
	TaskID           string         `yaml:"task_id"`
	Type             string         `yaml:"type"`
	CreatedAt        string         `yaml:"created_at"`
	PreviousPacketID *string        `yaml:"previous_packet_id"`
	Payload          map[string]any `yaml:"payload"`
	PayloadHash      string         `yaml:"payload_hash"`
}

type PacketChainResult struct {
	Valid               bool
	PacketsChecked      int
	FirstBrokenIndex    *int
	FirstBrokenPacketId *string
	ExpectedHash        *string
	ActualHash          *string
	Reason              *string
}

func normalizeForJson(value any) any {
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(v))
		for _, k := range keys {
			out[k] = normalizeForJson(v[k])
		}
		return out
	case map[any]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			if sk, ok := k.(string); ok {
				keys = append(keys, sk)
			} else {
				keys = append(keys, fmt.Sprintf("%v", k))
			}
		}
		sort.Strings(keys)
		out := make(map[string]any, len(v))
		for _, k := range keys {
			for mk, mv := range v {
				sk, sok := mk.(string)
				if sok && sk == k {
					out[k] = normalizeForJson(mv)
					break
				}
				if !sok && fmt.Sprintf("%v", mk) == k {
					out[k] = normalizeForJson(mv)
					break
				}
			}
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = normalizeForJson(item)
		}
		return out
	default:
		return v
	}
}

func computePayloadHash(payload map[string]any) string {
	normalized := normalizeForJson(payload)
	data, err := json.Marshal(normalized)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

func generatePacketId(taskId string) string {
	ts := time.Now().UTC().Format("2006-01-02T15:04:05.000000000Z")
	ts = strings.ReplaceAll(ts, ":", "-")
	ts = strings.ReplaceAll(ts, ".", "-")
	return fmt.Sprintf("packet-%s-%s", ts, taskId)
}

func createPacket(cardPath string, packetsDir string) (*Packet, string, error) {
	if err := os.MkdirAll(packetsDir, 0755); err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(cardPath)
	if err != nil {
		return nil, "", err
	}

	var card map[string]any
	if err := yaml.Unmarshal(data, &card); err != nil {
		return nil, "", err
	}

	taskId := "unknown"
	if v, ok := card["task_id"]; ok {
		if s, ok := v.(string); ok {
			taskId = s
		}
	}

	absPath, err := filepath.Abs(cardPath)
	if err != nil {
		return nil, "", err
	}

	payload := map[string]any{
		"card_path":    absPath,
		"card_content": card,
	}

	payloadHash := computePayloadHash(payload)

	existing, err := listPacketsForTask(taskId, packetsDir)
	if err != nil {
		return nil, "", err
	}

	var previousPacketId *string
	if len(existing) > 0 {
		last := existing[len(existing)-1].PacketID
		previousPacketId = &last
	}

	packet := &Packet{
		SchemaVersion:    "1",
		PacketID:         generatePacketId(taskId),
		TaskID:           taskId,
		Type:             "claim",
		CreatedAt:        time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		PreviousPacketID: previousPacketId,
		Payload:          payload,
		PayloadHash:      payloadHash,
	}

	fileName := packet.PacketID + ".yaml"
	filePath := filepath.Join(packetsDir, fileName)

	if _, err := os.Stat(filePath); err == nil {
		return nil, "", fmt.Errorf("Packet file already exists: %s", filePath)
	}

	out, err := yaml.Marshal(packet)
	if err != nil {
		return nil, "", err
	}

	if err := os.WriteFile(filePath, out, 0644); err != nil {
		return nil, "", err
	}

	return packet, filePath, nil
}

func isPacketLike(obj map[string]any) bool {
	_, hasId := obj["packet_id"]
	_, hasTaskId := obj["task_id"]
	_, hasHash := obj["payload_hash"]
	_, hasPayload := obj["payload"]
	return hasId && hasTaskId && hasHash && hasPayload
}

func listPacketsForTask(taskId string, packetsDir string) ([]*Packet, error) {
	entries, err := os.ReadDir(packetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Packet{}, nil
		}
		return nil, err
	}

	var packets []*Packet
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(packetsDir, name))
		if err != nil {
			continue
		}

		var parsed map[string]any
		if err := yaml.Unmarshal(data, &parsed); err != nil {
			continue
		}

		if !isPacketLike(parsed) {
			continue
		}

		var packet Packet
		if err := yaml.Unmarshal(data, &packet); err != nil {
			continue
		}

		if packet.TaskID == taskId {
			packets = append(packets, &packet)
		}
	}

	sort.Slice(packets, func(i, j int) bool {
		return packets[i].CreatedAt < packets[j].CreatedAt
	})

	return packets, nil
}

func verifyPacketChain(packets []*Packet) PacketChainResult {
	if len(packets) == 0 {
		return PacketChainResult{Valid: true, PacketsChecked: 0}
	}

	byId := make(map[string]*Packet, len(packets))
	for _, p := range packets {
		byId[p.PacketID] = p
	}

	childCount := make(map[string]int)
	for _, p := range packets {
		if p.PreviousPacketID != nil {
			childCount[*p.PreviousPacketID]++
		}
	}

	for parentId, count := range childCount {
		if count > 1 {
			reason := fmt.Sprintf("fork detected: packet %s has %d children", parentId, count)
			return PacketChainResult{
				Valid:               false,
				PacketsChecked:      len(packets),
				FirstBrokenPacketId: &parentId,
				Reason:              &reason,
			}
		}
	}

	for i, p := range packets {
		expectedHash := computePayloadHash(p.Payload)
		if p.PayloadHash != expectedHash {
			reason := fmt.Sprintf("payload hash mismatch for packet %s", p.PacketID)
			idx := i
			return PacketChainResult{
				Valid:               false,
				PacketsChecked:      i + 1,
				FirstBrokenIndex:    &idx,
				FirstBrokenPacketId: &p.PacketID,
				ExpectedHash:        &expectedHash,
				ActualHash:          &p.PayloadHash,
				Reason:              &reason,
			}
		}

		if p.PreviousPacketID != nil {
			if _, ok := byId[*p.PreviousPacketID]; !ok {
				reason := fmt.Sprintf("missing parent packet %s for packet %s", *p.PreviousPacketID, p.PacketID)
				idx := i
				return PacketChainResult{
					Valid:               false,
					PacketsChecked:      i + 1,
					FirstBrokenIndex:    &idx,
					FirstBrokenPacketId: &p.PacketID,
					Reason:              &reason,
				}
			}
		}
	}

	visited := make(map[string]bool)
	traversalIndex := 0

	for _, start := range packets {
		if visited[start.PacketID] {
			continue
		}

		pathVisited := make(map[string]bool)
		current := start

		for current != nil {
			if pathVisited[current.PacketID] {
				reason := fmt.Sprintf("cycle detected at packet %s", current.PacketID)
				return PacketChainResult{
					Valid:               false,
					PacketsChecked:      traversalIndex + 1,
					FirstBrokenPacketId: &current.PacketID,
					Reason:              &reason,
				}
			}

			if visited[current.PacketID] {
				break
			}

			pathVisited[current.PacketID] = true
			visited[current.PacketID] = true
			traversalIndex++

			var next *Packet
			for _, cand := range packets {
				if cand.PreviousPacketID != nil && *cand.PreviousPacketID == current.PacketID {
					next = cand
					break
				}
			}
			current = next
		}
	}

	if len(visited) != len(packets) {
		for _, p := range packets {
			if !visited[p.PacketID] {
				reason := fmt.Sprintf("orphan packet %s not reachable from root", p.PacketID)
				return PacketChainResult{
					Valid:               false,
					PacketsChecked:      len(visited),
					FirstBrokenPacketId: &p.PacketID,
					Reason:              &reason,
				}
			}
		}
	}

	return PacketChainResult{Valid: true, PacketsChecked: len(packets)}
}

func handlePacket(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: x-harness packet <create|verify-chain> [options]")
		return ExitUsage
	}

	subcommand := args[0]
	remaining := args[1:]

	switch subcommand {
	case "create":
		return handlePacketCreate(remaining, stdout, stderr)
	case "verify-chain":
		return handlePacketVerifyChain(remaining, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown packet subcommand: %s\n", subcommand)
		return ExitUsage
	}
}

func handlePacketCreate(args []string, stdout io.Writer, stderr io.Writer) int {
	cardPath := "completion-card.yaml"
	packetsDir := ".x-harness/packets"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--card":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: --card requires a value")
				return ExitUsage
			}
			cardPath = args[i+1]
			i++
		case "--packets-dir":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: --packets-dir requires a value")
				return ExitUsage
			}
			packetsDir = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
			return ExitUsage
		}
	}

	absCardPath, err := filepath.Abs(cardPath)
	if err != nil {
		absCardPath = cardPath
	}

	if _, err := os.Stat(absCardPath); os.IsNotExist(err) {
		fmt.Fprintf(stderr, "Error: Completion card not found at %s\n", absCardPath)
		return ExitUsage
	}

	packet, filePath, err := createPacket(absCardPath, packetsDir)
	if err != nil {
		fmt.Fprintf(stderr, "Error creating packet: %v\n", err)
		return ExitError
	}

	fmt.Fprintf(stdout, "packet created: %s\n", filePath)
	fmt.Fprintf(stdout, "packet_id: %s\n", packet.PacketID)
	fmt.Fprintf(stdout, "task_id: %s\n", packet.TaskID)
	if packet.PreviousPacketID != nil {
		fmt.Fprintf(stdout, "previous_packet_id: %s\n", *packet.PreviousPacketID)
	} else {
		fmt.Fprintln(stdout, "previous_packet_id: null")
	}
	fmt.Fprintf(stdout, "payload_hash: %s\n", packet.PayloadHash)
	return ExitOK
}

func handlePacketVerifyChain(args []string, stdout io.Writer, stderr io.Writer) int {
	taskId := ""
	packetsDir := ".x-harness/packets"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--task-id":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: --task-id requires a value")
				return ExitUsage
			}
			taskId = args[i+1]
			i++
		case "--packets-dir":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: --packets-dir requires a value")
				return ExitUsage
			}
			packetsDir = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "unknown flag: %s\n", args[i])
				return ExitUsage
			}
			fmt.Fprintf(stderr, "unexpected argument: %s\n", args[i])
			return ExitUsage
		}
	}

	if taskId == "" {
		fmt.Fprintln(stderr, "Error: --task-id is required")
		return ExitUsage
	}

	absPacketsDir, err := filepath.Abs(packetsDir)
	if err != nil {
		absPacketsDir = packetsDir
	}

	if _, err := os.Stat(absPacketsDir); os.IsNotExist(err) {
		fmt.Fprintf(stderr, "Error: Packets directory not found: %s\n", absPacketsDir)
		return ExitUsage
	}

	packets, err := listPacketsForTask(taskId, absPacketsDir)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return ExitError
	}

	if len(packets) == 0 {
		fmt.Fprintf(stderr, "Error: No packets found for task %s\n", taskId)
		return ExitUsage
	}

	result := verifyPacketChain(packets)

	if result.Valid {
		fmt.Fprintf(stdout, "chain valid: %d packet(s) checked\n", result.PacketsChecked)
		return ExitOK
	}

	fmt.Fprintf(stderr, "chain broken: %s\n", *result.Reason)
	if result.FirstBrokenPacketId != nil {
		fmt.Fprintf(stderr, "packet: %s\n", *result.FirstBrokenPacketId)
	}
	if result.ExpectedHash != nil && result.ActualHash != nil {
		fmt.Fprintf(stderr, "expected hash: %s\n", *result.ExpectedHash)
		fmt.Fprintf(stderr, "actual hash:   %s\n", *result.ActualHash)
	}
	return ExitError
}
