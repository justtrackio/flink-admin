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
		for i := int32(0); i < mapSize; i++ {
			if _, err := br.ReadUTF(); err != nil {
				return nil, fmt.Errorf("read operator state name: %w", err)
			}
			if _, err := br.ReadByte(); err != nil {
				return nil, fmt.Errorf("read operator state mode: %w", err)
			}
			offsetCount, err := br.ReadInt32()
			if err != nil {
				return nil, fmt.Errorf("read operator state offset count: %w", err)
			}
			if offsetCount < 0 {
				return nil, fmt.Errorf("operator state offset count negative: %d", offsetCount)
			}
			for j := int32(0); j < offsetCount; j++ {
				if _, err := br.ReadInt64(); err != nil {
					return nil, fmt.Errorf("read operator state offset: %w", err)
				}
			}
		}
		if h.Type == OperatorStateHandleFileMerging {
			if _, err := br.ReadUTF(); err != nil {
				return nil, fmt.Errorf("read operator state task owned dir: %w", err)
			}
			if _, err := br.ReadUTF(); err != nil {
				return nil, fmt.Errorf("read operator state shared dir: %w", err)
			}
			if _, err := br.ReadBool(); err != nil {
				return nil, fmt.Errorf("read operator state empty flag: %w", err)
			}
		}
		if _, err := readStreamStateHandle(br); err != nil {
			return nil, fmt.Errorf("read operator state delegate: %w", err)
		}
		return h, nil
	}

	h.StateNameToOffsets = make(map[string]OperatorStatePartition, mapSize)
	for i := int32(0); i < mapSize; i++ {
		name, err := br.ReadUTF()
		if err != nil {
			return nil, fmt.Errorf("read operator state name: %w", err)
		}

		modeOrdinal, err := br.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("read operator state mode: %w", err)
		}

		mode := distributionModeFromOrdinal(modeOrdinal)
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

		h.StateNameToOffsets[name] = OperatorStatePartition{
			DistributionMode: mode,
			Offsets:          offsets,
		}
	}

	if h.Type == OperatorStateHandleFileMerging {
		ownDir, err := br.ReadUTF()
		if err != nil {
			return nil, fmt.Errorf("read operator state task owned dir: %w", err)
		}
		sharedDir, err := br.ReadUTF()
		if err != nil {
			return nil, fmt.Errorf("read operator state shared dir: %w", err)
		}
		isEmpty, err := br.ReadBool()
		if err != nil {
			return nil, fmt.Errorf("read operator state empty flag: %w", err)
		}
		h.TaskOwnedDirectory = ownDir
		h.SharedDirectory = sharedDir
		h.IsEmptyFileMergingHandle = isEmpty
	}

	delegate, err := readStreamStateHandle(br)
	if err != nil {
		return nil, fmt.Errorf("read operator state handle delegate: %w", err)
	}
	if delegate != nil {
		h.DelegateState = delegate
	}

	return h, nil
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
