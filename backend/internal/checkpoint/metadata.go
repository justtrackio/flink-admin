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
	var err error
	var magic uint32
	var version int32
	var checkpointID int64
	var masterStates []MasterState
	var operatorStates []OperatorState
	var propertiesRaw []byte

	br := newBinaryReader(reader)
	if magic, err = br.ReadUint32(); err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}

	if magic != metadataMagicNumber {
		return nil, fmt.Errorf("invalid magic number: %x", magic)
	}

	if version, err = br.ReadInt32(); err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}

	if checkpointID, err = br.ReadInt64(); err != nil {
		return nil, fmt.Errorf("read checkpoint id: %w", err)
	}

	if masterStates, err = readMasterStates(br); err != nil {
		return nil, err
	}

	if operatorStates, err = readOperatorStates(br, version, options.ParseFull); err != nil {
		return nil, err
	}

	if propertiesRaw, err = io.ReadAll(br.r); err != nil {
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
	var err error
	var metadata *CheckpointMetadata

	buf := &bytes.Buffer{}
	tee := io.TeeReader(reader, buf)
	if metadata, err = Parse(tee, ParseOptions{ParseFull: false}); err != nil {
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
	var file *os.File

	if file, err = os.Open(path); err != nil {
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
	var file *os.File

	if file, err = os.Open(path); err != nil {
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
	var err error
	var count int32
	var magic uint32
	var payloadSize int32
	var payload []byte
	var version int32
	var name string
	var payloadLength int32
	var data []byte

	if count, err = br.ReadInt32(); err != nil {
		return nil, fmt.Errorf("read master state count: %w", err)
	}

	if count < 0 {
		return nil, fmt.Errorf("master state count negative: %d", count)
	}

	states := make([]MasterState, 0, count)
	for i := int32(0); i < count; i++ {
		if magic, err = br.ReadUint32(); err != nil {
			return nil, fmt.Errorf("read master state magic: %w", err)
		}

		if magic != 0xC96B1696 {
			return nil, fmt.Errorf("invalid master state magic: %x", magic)
		}

		if payloadSize, err = br.ReadInt32(); err != nil {
			return nil, fmt.Errorf("read master state payload size: %w", err)
		}

		if payload, err = br.ReadBytes(int(payloadSize)); err != nil {
			return nil, fmt.Errorf("read master state payload: %w", err)
		}

		innerReader := newBinaryReader(bytes.NewReader(payload))
		if version, err = innerReader.ReadInt32(); err != nil {
			return nil, fmt.Errorf("read master state version: %w", err)
		}

		if name, err = innerReader.ReadUTF(); err != nil {
			return nil, fmt.Errorf("read master state name: %w", err)
		}

		if payloadLength, err = innerReader.ReadInt32(); err != nil {
			return nil, fmt.Errorf("read master state data length: %w", err)
		}

		if data, err = innerReader.ReadBytes(int(payloadLength)); err != nil {
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
	var err error
	var count int32
	var state OperatorState

	if count, err = br.ReadInt32(); err != nil {
		return nil, fmt.Errorf("read operator state count: %w", err)
	}

	if count < 0 {
		return nil, fmt.Errorf("operator state count negative: %d", count)
	}

	states := make([]OperatorState, 0, count)
	for i := int32(0); i < count; i++ {
		if state, err = readOperatorState(br, version, parseFull); err != nil {
			return nil, err
		}

		states = append(states, state)
	}

	return states, nil
}

// readOperatorState parses a single operator state entry.
func readOperatorState(br *binaryReader, version int32, parseFull bool) (OperatorState, error) {
	var err error
	var low int64
	var high int64
	var parallelism int32
	var maxParallelism int32
	var subtaskCount int32
	var value string
	var handle *StreamStateHandle
	var state SubtaskState

	name := ""
	uid := ""
	if version >= 5 {
		if value, err = br.ReadUTF(); err != nil {
			return OperatorState{}, fmt.Errorf("read operator name: %w", err)
		}

		name = value
		value, err = br.ReadUTF()
		if err != nil {
			return OperatorState{}, fmt.Errorf("read operator uid: %w", err)
		}
		uid = value
	}

	if low, err = br.ReadInt64(); err != nil {
		return OperatorState{}, fmt.Errorf("read operator id low: %w", err)
	}

	if high, err = br.ReadInt64(); err != nil {
		return OperatorState{}, fmt.Errorf("read operator id high: %w", err)
	}

	operatorID := buildOperatorID(low, high)
	if parallelism, err = br.ReadInt32(); err != nil {
		return OperatorState{}, fmt.Errorf("read operator parallelism: %w", err)
	}

	if maxParallelism, err = br.ReadInt32(); err != nil {
		return OperatorState{}, fmt.Errorf("read operator max parallelism: %w", err)
	}

	coordinatorState := (*StreamStateHandle)(nil)
	if version >= 3 {
		if handle, err = readStreamStateHandle(br); err != nil {
			return OperatorState{}, fmt.Errorf("read operator coordinator state: %w", err)
		}

		coordinatorState = handle
	}

	if subtaskCount, err = br.ReadInt32(); err != nil {
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
			if state, err = readSubtaskState(br, version, parseFull); err != nil {
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
	var err error
	var index int32
	var managedOp *OperatorStateHandle
	var rawOp *OperatorStateHandle
	var managedKeyed KeyedStateHandle
	var rawKeyed KeyedStateHandle
	var inputStates []ChannelStateHandle
	var outputStates []ChannelStateHandle

	if index, err = br.ReadInt32(); err != nil {
		return SubtaskState{}, fmt.Errorf("read subtask index: %w", err)
	}

	if index < 0 {
		return SubtaskState{
			Index:    -(index + 1),
			Finished: true,
		}, nil
	}

	if managedOp, err = readOptionalOperatorStateHandle(br, parseFull); err != nil {
		return SubtaskState{}, fmt.Errorf("read managed operator state: %w", err)
	}

	if rawOp, err = readOptionalOperatorStateHandle(br, parseFull); err != nil {
		return SubtaskState{}, fmt.Errorf("read raw operator state: %w", err)
	}

	if managedKeyed, err = readKeyedStateHandle(br, parseFull); err != nil {
		return SubtaskState{}, fmt.Errorf("read managed keyed state: %w", err)
	}

	if rawKeyed, err = readKeyedStateHandle(br, parseFull); err != nil {
		return SubtaskState{}, fmt.Errorf("read raw keyed state: %w", err)
	}

	if inputStates, err = readChannelStateHandles(br, version, ChannelStateInput, parseFull); err != nil {
		return SubtaskState{}, fmt.Errorf("read input channel states: %w", err)
	}

	if outputStates, err = readChannelStateHandles(br, version, ChannelStateOutput, parseFull); err != nil {
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
	var err error
	var marker int32
	var handle *OperatorStateHandle

	if marker, err = br.ReadInt32(); err != nil {
		return nil, fmt.Errorf("read operator state marker: %w", err)
	}

	if marker == 0 {
		return nil, nil
	}
	if marker != 1 {
		return nil, fmt.Errorf("unexpected operator state marker %d", marker)
	}

	if handle, err = readOperatorStateHandle(br, parseFull); err != nil {
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
