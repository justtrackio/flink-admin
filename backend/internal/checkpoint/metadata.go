package checkpoint

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

const metadataMagicNumber uint32 = 0x4960672d

type ParseOptions struct {
	ParseFull            bool
	IncludeInlineStrings bool
}

// Parse reads a Flink checkpoint _metadata stream and returns the parsed result.
func Parse(reader io.Reader, options ParseOptions) (*CheckpointMetadata, error) {
	br := newBinaryReader(reader)
	magic, err := br.ReadUint32()
	if err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}
	if magic != metadataMagicNumber {
		return nil, fmt.Errorf("invalid magic number: %x", magic)
	}

	version, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}
	checkpointID, err := br.ReadInt64()
	if err != nil {
		return nil, fmt.Errorf("read checkpoint id: %w", err)
	}

	masterStates, err := readMasterStates(br)
	if err != nil {
		return nil, err
	}

	operatorStates, err := readOperatorStates(br, version, options.ParseFull)
	if err != nil {
		return nil, err
	}

	propertiesRaw, err := io.ReadAll(br.r)
	if err != nil {
		return nil, fmt.Errorf("read properties raw: %w", err)
	}

	metadata := &CheckpointMetadata{
		Magic:          magic,
		Version:        version,
		CheckpointID:   checkpointID,
		MasterStates:   masterStates,
		OperatorStates: operatorStates,
		PropertiesRaw:  propertiesRaw,
	}

	if version >= 4 && len(propertiesRaw) > 0 {
		metadata.Properties = parseCheckpointProperties(propertiesRaw)
	}

	return metadata, nil
}

// ParseSummary returns a lightweight summary of a _metadata stream.
func ParseSummary(reader io.Reader, options ParseOptions) (*CheckpointSummary, error) {
	buf := &bytes.Buffer{}
	tee := io.TeeReader(reader, buf)
	metadata, err := Parse(tee, ParseOptions{ParseFull: false})
	if err != nil {
		return nil, err
	}

	summary := &CheckpointSummary{
		Version:       metadata.Version,
		CheckpointID:  metadata.CheckpointID,
		NumOperators:  len(metadata.OperatorStates),
		Operators:     make([]OperatorSummary, 0, len(metadata.OperatorStates)),
		Properties:    metadata.Properties,
		PropertiesRaw: metadata.PropertiesRaw,
	}

	for _, operator := range metadata.OperatorStates {
		summary.Operators = append(summary.Operators, OperatorSummary{
			Name:           operator.Name,
			UID:            operator.UID,
			OperatorID:     operator.OperatorID,
			Parallelism:    operator.Parallelism,
			MaxParallelism: operator.MaxParallelism,
		})
	}

	if options.IncludeInlineStrings {
		strings := scanInlineStrings(buf.Bytes())
		summary.InlineStrings = strings
		summary.StateFilePaths = extractStateFilePaths(buf.Bytes())
	}

	return summary, nil
}

// ParseFile opens the given file path and parses it as _metadata.
func ParseFile(path string, options ParseOptions) (metadata *CheckpointMetadata, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open metadata file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close metadata file: %w", cerr)
		}
	}()

	metadata, err = Parse(file, options)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// ParseFileSummary opens the given file path and parses it as a summary.
func ParseFileSummary(path string, options ParseOptions) (summary *CheckpointSummary, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open metadata file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close metadata file: %w", cerr)
		}
	}()

	summary, err = ParseSummary(file, options)
	if err != nil {
		return nil, err
	}

	return summary, nil
}

// readMasterStates parses master state entries from the stream.
func readMasterStates(br *binaryReader) ([]MasterState, error) {
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read master state count: %w", err)
	}
	if count < 0 {
		return nil, fmt.Errorf("master state count negative: %d", count)
	}

	states := make([]MasterState, 0, count)
	for i := int32(0); i < count; i++ {
		magic, err := br.ReadUint32()
		if err != nil {
			return nil, fmt.Errorf("read master state magic: %w", err)
		}
		if magic != 0xC96B1696 {
			return nil, fmt.Errorf("invalid master state magic: %x", magic)
		}

		payloadSize, err := br.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("read master state payload size: %w", err)
		}
		payload, err := br.ReadBytes(int(payloadSize))
		if err != nil {
			return nil, fmt.Errorf("read master state payload: %w", err)
		}

		innerReader := newBinaryReader(bytes.NewReader(payload))
		version, err := innerReader.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("read master state version: %w", err)
		}
		name, err := innerReader.ReadUTF()
		if err != nil {
			return nil, fmt.Errorf("read master state name: %w", err)
		}
		payloadLength, err := innerReader.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("read master state data length: %w", err)
		}
		data, err := innerReader.ReadBytes(int(payloadLength))
		if err != nil {
			return nil, fmt.Errorf("read master state data: %w", err)
		}

		states = append(states, MasterState{
			Version: version,
			Name:    name,
			Payload: data,
		})
	}

	return states, nil
}

// readOperatorStates parses operator state entries from the stream.
func readOperatorStates(br *binaryReader, version int32, parseFull bool) ([]OperatorState, error) {
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read operator state count: %w", err)
	}
	if count < 0 {
		return nil, fmt.Errorf("operator state count negative: %d", count)
	}

	states := make([]OperatorState, 0, count)
	for i := int32(0); i < count; i++ {
		state, err := readOperatorState(br, version, parseFull)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}

	return states, nil
}

// readOperatorState parses a single operator state entry.
func readOperatorState(br *binaryReader, version int32, parseFull bool) (OperatorState, error) {
	name := ""
	uid := ""
	if version >= 5 {
		value, err := br.ReadUTF()
		if err != nil {
			return OperatorState{}, fmt.Errorf("read operator name: %w", err)
		}
		name = value
		value, err = br.ReadUTF()
		if err != nil {
			return OperatorState{}, fmt.Errorf("read operator uid: %w", err)
		}
		uid = value
	}

	low, err := br.ReadInt64()
	if err != nil {
		return OperatorState{}, fmt.Errorf("read operator id low: %w", err)
	}
	high, err := br.ReadInt64()
	if err != nil {
		return OperatorState{}, fmt.Errorf("read operator id high: %w", err)
	}

	operatorID := buildOperatorID(low, high)
	parallelism, err := br.ReadInt32()
	if err != nil {
		return OperatorState{}, fmt.Errorf("read operator parallelism: %w", err)
	}
	maxParallelism, err := br.ReadInt32()
	if err != nil {
		return OperatorState{}, fmt.Errorf("read operator max parallelism: %w", err)
	}

	coordinatorState := (*StreamStateHandle)(nil)
	if version >= 3 {
		handle, err := readStreamStateHandle(br)
		if err != nil {
			return OperatorState{}, fmt.Errorf("read operator coordinator state: %w", err)
		}
		coordinatorState = handle
	}

	subtaskCount, err := br.ReadInt32()
	if err != nil {
		return OperatorState{}, fmt.Errorf("read operator subtask count: %w", err)
	}
	if subtaskCount < -1 {
		return OperatorState{}, fmt.Errorf("operator subtask count invalid: %d", subtaskCount)
	}

	finished := subtaskCount == -1
	var subtasks []SubtaskState
	if !finished {
		subtasks = make([]SubtaskState, 0, subtaskCount)
		for i := int32(0); i < subtaskCount; i++ {
			state, err := readSubtaskState(br, version, parseFull)
			if err != nil {
				return OperatorState{}, err
			}
			subtasks = append(subtasks, state)
		}
	}

	return OperatorState{
		Name:             name,
		UID:              uid,
		OperatorID:       operatorID,
		Parallelism:      parallelism,
		MaxParallelism:   maxParallelism,
		CoordinatorState: coordinatorState,
		SubtaskStates:    subtasks,
		Finished:         finished,
	}, nil
}

// readSubtaskState parses a subtask state entry.
func readSubtaskState(br *binaryReader, version int32, parseFull bool) (SubtaskState, error) {
	index, err := br.ReadInt32()
	if err != nil {
		return SubtaskState{}, fmt.Errorf("read subtask index: %w", err)
	}
	if index < 0 {
		return SubtaskState{
			Index:    -(index + 1),
			Finished: true,
		}, nil
	}

	managedOp, err := readOptionalOperatorStateHandle(br, parseFull)
	if err != nil {
		return SubtaskState{}, fmt.Errorf("read managed operator state: %w", err)
	}
	rawOp, err := readOptionalOperatorStateHandle(br, parseFull)
	if err != nil {
		return SubtaskState{}, fmt.Errorf("read raw operator state: %w", err)
	}
	managedKeyed, err := readKeyedStateHandle(br, parseFull)
	if err != nil {
		return SubtaskState{}, fmt.Errorf("read managed keyed state: %w", err)
	}
	rawKeyed, err := readKeyedStateHandle(br, parseFull)
	if err != nil {
		return SubtaskState{}, fmt.Errorf("read raw keyed state: %w", err)
	}

	inputStates, err := readChannelStateHandles(br, version, ChannelStateInput, parseFull)
	if err != nil {
		return SubtaskState{}, fmt.Errorf("read input channel states: %w", err)
	}
	outputStates, err := readChannelStateHandles(br, version, ChannelStateOutput, parseFull)
	if err != nil {
		return SubtaskState{}, fmt.Errorf("read output channel states: %w", err)
	}

	return SubtaskState{
		Index:                index,
		Finished:             false,
		ManagedOperatorState: managedOp,
		RawOperatorState:     rawOp,
		ManagedKeyedState:    managedKeyed,
		RawKeyedState:        rawKeyed,
		InputChannelStates:   inputStates,
		OutputChannelStates:  outputStates,
	}, nil
}

// readOptionalOperatorStateHandle reads a marker and optional operator state handle.
func readOptionalOperatorStateHandle(br *binaryReader, parseFull bool) (*OperatorStateHandle, error) {
	marker, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read operator state marker: %w", err)
	}
	if marker == 0 {
		return nil, nil
	}
	if marker != 1 {
		return nil, fmt.Errorf("unexpected operator state marker %d", marker)
	}

	handle, err := readOperatorStateHandle(br, parseFull)
	if err != nil {
		return nil, err
	}

	return handle, nil
}

// buildOperatorID builds a 16-byte operator ID from two 64-bit parts.
func buildOperatorID(low int64, high int64) [16]byte {
	var id [16]byte
	for i := 0; i < 8; i++ {
		id[i] = byte(high >> uint(56-8*i))
		id[i+8] = byte(low >> uint(56-8*i))
	}

	return id
}
