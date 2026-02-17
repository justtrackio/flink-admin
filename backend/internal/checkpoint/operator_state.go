package checkpoint

import "fmt"

// readOperatorStateHandle parses a single operator state handle.
func readOperatorStateHandle(br *binaryReader, parseFull bool) (*OperatorStateHandle, error) {
	kind, err := br.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("read operator state handle type: %w", err)
	}

	if OperatorStateHandleType(kind) == OperatorStateHandleNull {
		return nil, nil
	}

	h := &OperatorStateHandle{Type: OperatorStateHandleType(kind)}
	if h.Type != OperatorStateHandlePartitionable && h.Type != OperatorStateHandleFileMerging {
		return nil, fmt.Errorf("unsupported operator state handle type %d", kind)
	}

	mapSize, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read operator state handle map size: %w", err)
	}

	if mapSize < 0 {
		return nil, fmt.Errorf("operator state handle map size negative: %d", mapSize)
	}

	if !parseFull {
		if err := skipOperatorStateHandle(br, h.Type, mapSize); err != nil {
			return nil, err
		}

		return h, nil
	}

	if err := populateOperatorStateHandle(br, h, mapSize); err != nil {
		return nil, err
	}

	return h, nil
}

func skipOperatorStateHandle(br *binaryReader, handleType OperatorStateHandleType, mapSize int32) error {
	if err := skipOperatorStateEntries(br, mapSize); err != nil {
		return err
	}
	if handleType == OperatorStateHandleFileMerging {
		if err := skipFileMergingData(br); err != nil {
			return err
		}
	}
	if err := skipStreamHandle(br); err != nil {
		return err
	}

	return nil
}

func populateOperatorStateHandle(br *binaryReader, h *OperatorStateHandle, mapSize int32) error {
	h.StateNameToOffsets = make(map[string]OperatorStatePartition, mapSize)
	if err := readOperatorStateEntries(br, h, mapSize); err != nil {
		return err
	}

	if h.Type == OperatorStateHandleFileMerging {
		if err := readFileMergingData(br, h); err != nil {
			return err
		}
	}

	if err := readDelegateStateHandle(br, h); err != nil {
		return err
	}

	return nil
}

func skipOperatorStateEntries(br *binaryReader, mapSize int32) error {
	for i := int32(0); i < mapSize; i++ {
		if err := skipOperatorStateEntry(br); err != nil {
			return err
		}
	}

	return nil
}

func readOperatorStateEntries(br *binaryReader, h *OperatorStateHandle, mapSize int32) error {
	for i := int32(0); i < mapSize; i++ {
		if err := readOperatorStateEntry(br, h); err != nil {
			return err
		}
	}

	return nil
}

func skipOperatorStateEntry(br *binaryReader) error {
	if _, err := br.ReadUTF(); err != nil {
		return fmt.Errorf("read operator state name: %w", err)
	}
	if _, err := br.ReadByte(); err != nil {
		return fmt.Errorf("read operator state mode: %w", err)
	}

	return skipOffsets(br)
}

func skipOffsets(br *binaryReader) error {
	offsetCount, err := br.ReadInt32()
	if err != nil {
		return fmt.Errorf("read operator state offset count: %w", err)
	}
	if offsetCount < 0 {
		return fmt.Errorf("operator state offset count negative: %d", offsetCount)
	}
	for j := int32(0); j < offsetCount; j++ {
		if _, err := br.ReadInt64(); err != nil {
			return fmt.Errorf("read operator state offset: %w", err)
		}
	}

	return nil
}

func readOperatorStateEntry(br *binaryReader, h *OperatorStateHandle) error {
	name, err := br.ReadUTF()
	if err != nil {
		return fmt.Errorf("read operator state name: %w", err)
	}

	modeOrdinal, err := br.ReadByte()
	if err != nil {
		return fmt.Errorf("read operator state mode: %w", err)
	}

	mode := distributionModeFromOrdinal(modeOrdinal)
	offsets, err := readOffsets(br)
	if err != nil {
		return err
	}

	h.StateNameToOffsets[name] = OperatorStatePartition{
		DistributionMode: mode,
		Offsets:          offsets,
	}

	return nil
}

func readOffsets(br *binaryReader) ([]int64, error) {
	offsetCount, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read operator state offset count: %w", err)
	}
	if offsetCount < 0 {
		return nil, fmt.Errorf("operator state offset count negative: %d", offsetCount)
	}

	offsets := make([]int64, offsetCount)
	for j := int32(0); j < offsetCount; j++ {
		offset, err := br.ReadInt64()
		if err != nil {
			return nil, fmt.Errorf("read operator state offset: %w", err)
		}
		offsets[j] = offset
	}

	return offsets, nil
}

func readFileMergingData(br *binaryReader, h *OperatorStateHandle) error {
	ownDir, err := br.ReadUTF()
	if err != nil {
		return fmt.Errorf("read operator state task owned dir: %w", err)
	}
	sharedDir, err := br.ReadUTF()
	if err != nil {
		return fmt.Errorf("read operator state shared dir: %w", err)
	}
	isEmpty, err := br.ReadBool()
	if err != nil {
		return fmt.Errorf("read operator state empty flag: %w", err)
	}
	h.TaskOwnedDirectory = ownDir
	h.SharedDirectory = sharedDir
	h.IsEmptyFileMergingHandle = isEmpty

	return nil
}

func readDelegateStateHandle(br *binaryReader, h *OperatorStateHandle) error {
	delegate, err := readStreamStateHandle(br)
	if err != nil {
		return fmt.Errorf("read operator state handle delegate: %w", err)
	}
	if delegate != nil {
		h.DelegateState = delegate
	}

	return nil
}

func skipFileMergingData(br *binaryReader) error {
	if _, err := br.ReadUTF(); err != nil {
		return fmt.Errorf("read operator state task owned dir: %w", err)
	}
	if _, err := br.ReadUTF(); err != nil {
		return fmt.Errorf("read operator state shared dir: %w", err)
	}
	if _, err := br.ReadBool(); err != nil {
		return fmt.Errorf("read operator state empty flag: %w", err)
	}

	return nil
}

func skipStreamHandle(br *binaryReader) error {
	if _, err := readStreamStateHandle(br); err != nil {
		return fmt.Errorf("read operator state delegate: %w", err)
	}

	return nil
}

// distributionModeFromOrdinal converts Flink's ordinal to a readable label.
func distributionModeFromOrdinal(value byte) string {
	switch value {
	case 0:
		return "SPLIT_DISTRIBUTE"
	case 1:
		return "UNION"
	case 2:
		return "BROADCAST"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", value)
	}
}
