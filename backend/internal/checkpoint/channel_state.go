package checkpoint

import "fmt"

// ChannelStateType indicates whether a channel state is input or output.
type ChannelStateType byte

const (
	// ChannelStateInput represents input channel state (InputChannelStateHandle).
	ChannelStateInput ChannelStateType = 1
	// ChannelStateOutput represents output channel state (ResultSubpartitionStateHandle).
	ChannelStateOutput ChannelStateType = 2
)

// readChannelStateHandles parses input/output channel state handles.
// For v3-v5, the channelType parameter determines whether to read input or output handles
// (since V1 format doesn't include a type discriminator byte).
// For v6+, the type is read from the stream.
func readChannelStateHandles(br *binaryReader, version int32, channelType ChannelStateType, parseFull bool) ([]ChannelStateHandle, error) {
	if version < 3 {
		return nil, nil
	}

	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read channel state count: %w", err)
	}
	if count < 0 {
		return nil, fmt.Errorf("channel state count negative: %d", count)
	}

	if count == 0 {
		return nil, nil
	}

	states := make([]ChannelStateHandle, 0, count)
	for i := int32(0); i < count; i++ {
		var handle ChannelStateHandle
		var err error
		if version >= 6 {
			handle, err = readChannelStateHandleV2(br, parseFull)
		} else {
			// V1 format: no type byte in stream, type is implicit from context
			handle, err = readChannelStateHandleV1(br, channelType, parseFull)
		}
		if err != nil {
			return nil, err
		}
		states = append(states, handle)
	}

	return states, nil
}

// readChannelStateHandleV1 parses channel state handles for metadata v3-v5.
// In V1 format, no type discriminator is written; the type is known from context
// (whether we're reading input channel states or result subpartition states).
func readChannelStateHandleV1(br *binaryReader, channelType ChannelStateType, parseFull bool) (ChannelStateHandle, error) {
	switch channelType {
	case ChannelStateInput:
		return readInputChannelStateHandle(br, byte(channelType), parseFull)
	case ChannelStateOutput:
		return readResultSubpartitionStateHandle(br, byte(channelType), parseFull)
	default:
		return ChannelStateHandle{}, fmt.Errorf("unsupported channel state type %d", channelType)
	}
}

// readChannelStateHandleV2 parses channel state handles for metadata v6+.
func readChannelStateHandleV2(br *binaryReader, parseFull bool) (ChannelStateHandle, error) {
	stateType, err := br.ReadByte()
	if err != nil {
		return ChannelStateHandle{}, fmt.Errorf("read channel state type: %w", err)
	}

	switch stateType {
	case 1:
		return readInputChannelStateHandle(br, stateType, parseFull)
	case 2:
		return readResultSubpartitionStateHandle(br, stateType, parseFull)
	case 3:
		return readMergedInputChannelStateHandle(br, stateType, parseFull)
	case 4:
		return readMergedResultSubpartitionStateHandle(br, stateType, parseFull)
	default:
		return ChannelStateHandle{}, fmt.Errorf("unsupported channel state type %d", stateType)
	}
}

// readInputChannelStateHandle parses input channel state handle entries.
func readInputChannelStateHandle(br *binaryReader, stateType byte, parseFull bool) (ChannelStateHandle, error) {
	return readUnmergedChannelStateHandle(br, stateType, parseFull, "input channel")
}

// readResultSubpartitionStateHandle parses result subpartition state handle entries.
func readResultSubpartitionStateHandle(br *binaryReader, stateType byte, parseFull bool) (ChannelStateHandle, error) {
	return readUnmergedChannelStateHandle(br, stateType, parseFull, "result subpartition")
}

// readUnmergedChannelStateHandle is the shared implementation for readInputChannelStateHandle
// and readResultSubpartitionStateHandle. Both have identical binary layouts:
// subtask (int32), index1 (int32), index2 (int32), offsets (int32 count + int64s),
// stateSize (int64), delegate (StreamStateHandle).
func readUnmergedChannelStateHandle(br *binaryReader, stateType byte, parseFull bool, label string) (ChannelStateHandle, error) {
	subtask, err := br.ReadInt32()
	if err != nil {
		return ChannelStateHandle{}, fmt.Errorf("read %s subtask: %w", label, err)
	}
	index1, err := br.ReadInt32()
	if err != nil {
		return ChannelStateHandle{}, fmt.Errorf("read %s index1: %w", label, err)
	}
	index2, err := br.ReadInt32()
	if err != nil {
		return ChannelStateHandle{}, fmt.Errorf("read %s index2: %w", label, err)
	}
	offsetCount, err := br.ReadInt32()
	if err != nil {
		return ChannelStateHandle{}, fmt.Errorf("read %s offset count: %w", label, err)
	}
	if offsetCount < 0 {
		return ChannelStateHandle{}, fmt.Errorf("%s offset count negative: %d", label, offsetCount)
	}
	offsets := make([]int64, offsetCount)
	for i := int32(0); i < offsetCount; i++ {
		offset, err := br.ReadInt64()
		if err != nil {
			return ChannelStateHandle{}, fmt.Errorf("read %s offset: %w", label, err)
		}
		offsets[i] = offset
	}
	stateSize, err := br.ReadInt64()
	if err != nil {
		return ChannelStateHandle{}, fmt.Errorf("read %s state size: %w", label, err)
	}
	delegate, err := readStreamStateHandle(br)
	if err != nil {
		return ChannelStateHandle{}, fmt.Errorf("read %s delegate: %w", label, err)
	}
	if !parseFull {
		return ChannelStateHandle{
			Type:         stateType,
			SubtaskIndex: subtask,
			StateSize:    stateSize,
			Handle:       delegate,
		}, nil
	}

	return ChannelStateHandle{
		Type:                  stateType,
		SubtaskIndex:          subtask,
		GateOrPartition:       index1,
		ChannelOrSubpartition: index2,
		Offsets:               offsets,
		StateSize:             stateSize,
		Handle:                delegate,
	}, nil
}

// readMergedInputChannelStateHandle parses merged input channel state handles.
func readMergedInputChannelStateHandle(br *binaryReader, stateType byte, parseFull bool) (ChannelStateHandle, error) {
	return readMergedChannelStateHandle(br, stateType, parseFull, "merged input")
}

// readMergedResultSubpartitionStateHandle parses merged result subpartition handles.
func readMergedResultSubpartitionStateHandle(br *binaryReader, stateType byte, parseFull bool) (ChannelStateHandle, error) {
	return readMergedChannelStateHandle(br, stateType, parseFull, "merged result")
}

// readMergedChannelStateHandle is the shared implementation for readMergedInputChannelStateHandle
// and readMergedResultSubpartitionStateHandle. Both have identical binary layouts:
// subtask (int32), stateSize (int64), delegate (StreamStateHandle), rawOffsets (int32 length + bytes).
func readMergedChannelStateHandle(br *binaryReader, stateType byte, parseFull bool, label string) (ChannelStateHandle, error) {
	subtask, err := br.ReadInt32()
	if err != nil {
		return ChannelStateHandle{}, fmt.Errorf("read %s subtask: %w", label, err)
	}
	stateSize, err := br.ReadInt64()
	if err != nil {
		return ChannelStateHandle{}, fmt.Errorf("read %s state size: %w", label, err)
	}
	delegate, err := readStreamStateHandle(br)
	if err != nil {
		return ChannelStateHandle{}, fmt.Errorf("read %s delegate: %w", label, err)
	}
	length, err := br.ReadInt32()
	if err != nil {
		return ChannelStateHandle{}, fmt.Errorf("read %s offsets length: %w", label, err)
	}
	data, err := br.ReadBytes(int(length))
	if err != nil {
		return ChannelStateHandle{}, fmt.Errorf("read %s offsets: %w", label, err)
	}
	if !parseFull {
		return ChannelStateHandle{
			Type:         stateType,
			SubtaskIndex: subtask,
			StateSize:    stateSize,
			Handle:       delegate,
		}, nil
	}

	return ChannelStateHandle{
		Type:         stateType,
		SubtaskIndex: subtask,
		StateSize:    stateSize,
		Handle:       delegate,
		RawOffsets:   data,
	}, nil
}
