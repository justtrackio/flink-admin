package checkpoint

import "fmt"

// readKeyedStateHandle parses a keyed state handle, selecting the correct variant.
func readKeyedStateHandle(br *binaryReader, parseFull bool) (KeyedStateHandle, error) {
	_ = parseFull
	kind, err := br.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("read keyed state handle type: %w", err)
	}

	if KeyedStateHandleType(kind) == KeyedStateHandleNull {
		return nil, nil
	}

	switch KeyedStateHandleType(kind) {
	case KeyedStateHandleLegacy, KeyedStateHandleSavepoint, KeyedStateHandleKeyGroupsV2:
		return readKeyGroupsHandle(br, KeyedStateHandleType(kind))
	case KeyedStateHandleIncrementalLegacy, KeyedStateHandleIncrementalV2:
		return readIncrementalKeyGroupsHandle(br, KeyedStateHandleType(kind))
	case KeyedStateHandleChangelogLegacy, KeyedStateHandleChangelogV2:
		return readChangelogStateHandle(br, KeyedStateHandleType(kind), parseFull)
	case KeyedStateHandleChangelogByte:
		return readChangelogByteIncrementHandle(br, KeyedStateHandleType(kind), parseFull)
	case KeyedStateHandleChangelogFileLegacy, KeyedStateHandleChangelogFileV2:
		return readChangelogFileIncrementHandle(br, KeyedStateHandleType(kind))
	default:
		return nil, fmt.Errorf("unsupported keyed state handle type %d", kind)
	}
}

// readKeyGroupsHandle parses key-group based handles.
func readKeyGroupsHandle(br *binaryReader, kind KeyedStateHandleType) (KeyedStateHandle, error) {
	startKeyGroup, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read key groups start: %w", err)
	}
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read key groups count: %w", err)
	}
	if count < 0 {
		return nil, fmt.Errorf("key groups count negative: %d", count)
	}

	offsets := make([]int64, count)
	for i := int32(0); i < count; i++ {
		offset, err := br.ReadInt64()
		if err != nil {
			return nil, fmt.Errorf("read key groups offset: %w", err)
		}
		offsets[i] = offset
	}

	delegate, err := readStreamStateHandle(br)
	if err != nil {
		return nil, fmt.Errorf("read key groups delegate: %w", err)
	}

	handleID := ""
	if kind == KeyedStateHandleKeyGroupsV2 {
		id, err := br.ReadUTF()
		if err != nil {
			return nil, fmt.Errorf("read key groups handle id: %w", err)
		}
		handleID = id
	}

	return KeyGroupsHandle{
		Type:          kind,
		StartKeyGroup: startKeyGroup,
		NumKeyGroups:  count,
		Offsets:       offsets,
		Delegate:      delegate,
		HandleID:      handleID,
	}, nil
}

// readIncrementalKeyGroupsHandle parses incremental state handles.
// Type 5 (legacy) does not have checkpointedSize or stateHandleId fields.
// Type 11 (V2) includes both fields.
func readIncrementalKeyGroupsHandle(br *binaryReader, kind KeyedStateHandleType) (KeyedStateHandle, error) {
	isV2 := kind == KeyedStateHandleIncrementalV2

	checkpointID, err := br.ReadInt64()
	if err != nil {
		return nil, fmt.Errorf("read incremental checkpoint id: %w", err)
	}
	backendID, err := br.ReadUTF()
	if err != nil {
		return nil, fmt.Errorf("read incremental backend id: %w", err)
	}
	startKeyGroup, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read incremental start key group: %w", err)
	}
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read incremental key group count: %w", err)
	}

	// checkpointedSize is only present in V2 (type 11)
	var checkpointedSize int64 = -1 // UNKNOWN_CHECKPOINTED_SIZE
	if isV2 {
		checkpointedSize, err = br.ReadInt64()
		if err != nil {
			return nil, fmt.Errorf("read incremental checkpointed size: %w", err)
		}
	}

	metaHandle, err := readStreamStateHandle(br)
	if err != nil {
		return nil, fmt.Errorf("read incremental meta handle: %w", err)
	}

	sharedFiles, err := readHandleAndLocalPathList(br)
	if err != nil {
		return nil, fmt.Errorf("read incremental shared files: %w", err)
	}

	privateFiles, err := readHandleAndLocalPathList(br)
	if err != nil {
		return nil, fmt.Errorf("read incremental private files: %w", err)
	}

	// stateHandleId is only present in V2 (type 11)
	stateID := ""
	if isV2 {
		stateID, err = br.ReadUTF()
		if err != nil {
			return nil, fmt.Errorf("read incremental handle id: %w", err)
		}
	}

	return IncrementalKeyGroupsHandle{
		Type:             kind,
		CheckpointID:     checkpointID,
		BackendID:        backendID,
		StartKeyGroup:    startKeyGroup,
		NumKeyGroups:     count,
		CheckpointedSize: checkpointedSize,
		MetaHandle:       metaHandle,
		SharedFiles:      sharedFiles,
		PrivateFiles:     privateFiles,
		HandleID:         stateID,
	}, nil
}

// readHandleAndLocalPathList parses a list of state handles with local paths.
func readHandleAndLocalPathList(br *binaryReader) ([]HandleAndLocalPath, error) {
	count, err := br.ReadInt32()
	if err != nil {
		return nil, err
	}
	if count < 0 {
		return nil, fmt.Errorf("handle list count negative: %d", count)
	}

	entries := make([]HandleAndLocalPath, 0, count)
	for i := int32(0); i < count; i++ {
		path, err := br.ReadUTF()
		if err != nil {
			return nil, fmt.Errorf("read handle local path: %w", err)
		}
		handle, err := readStreamStateHandle(br)
		if err != nil {
			return nil, fmt.Errorf("read handle stream: %w", err)
		}
		entries = append(entries, HandleAndLocalPath{
			LocalPath: path,
			Handle:    handle,
		})
	}

	return entries, nil
}

// readChangelogStateHandle parses changelog handles with materialized state.
// Type 8 (legacy) does not have a separate checkpointId field; it uses materializationID.
// Type 14 (V2) includes a separate checkpointId field.
func readChangelogStateHandle(br *binaryReader, kind KeyedStateHandleType, parseFull bool) (KeyedStateHandle, error) {
	_ = parseFull
	startKeyGroup, numKeyGroups, checkpointedSize, err := readChangelogHeader(br)
	if err != nil {
		return nil, err
	}

	materialized, err := readChangelogKeyedStateHandles(br, "materialized")
	if err != nil {
		return nil, err
	}

	nonMaterialized, err := readChangelogKeyedStateHandles(br, "non materialized")
	if err != nil {
		return nil, err
	}

	materializationID, err := br.ReadInt64()
	if err != nil {
		return nil, fmt.Errorf("read changelog materialization id: %w", err)
	}

	checkpointID := materializationID
	if kind == KeyedStateHandleChangelogV2 {
		checkpointID, err = br.ReadInt64()
		if err != nil {
			return nil, fmt.Errorf("read changelog checkpoint id: %w", err)
		}
	}

	stateID, err := br.ReadUTF()
	if err != nil {
		return nil, fmt.Errorf("read changelog handle id: %w", err)
	}

	return ChangelogStateHandle{
		Type:              kind,
		StartKeyGroup:     startKeyGroup,
		NumKeyGroups:      numKeyGroups,
		CheckpointedSize:  checkpointedSize,
		Materialized:      materialized,
		NonMaterialized:   nonMaterialized,
		MaterializationID: materializationID,
		CheckpointID:      checkpointID,
		HandleID:          stateID,
	}, nil
}

func readChangelogHeader(br *binaryReader) (startKeyGroup int32, numKeyGroups int32, checkpointedSize int64, err error) {
	startKeyGroup, err = br.ReadInt32()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("read changelog start key group: %w", err)
	}
	count, err := br.ReadInt32()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("read changelog key group count: %w", err)
	}
	checkpointedSize, err = br.ReadInt64()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("read changelog checkpointed size: %w", err)
	}

	return startKeyGroup, count, checkpointedSize, nil
}

func readChangelogKeyedStateHandles(br *binaryReader, label string) ([]KeyedStateHandle, error) {
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read changelog %s count: %w", label, err)
	}
	if count < 0 {
		return nil, fmt.Errorf("changelog %s count negative: %d", label, count)
	}

	handles := make([]KeyedStateHandle, 0, count)
	for i := int32(0); i < count; i++ {
		handle, err := readKeyedStateHandle(br, true)
		if err != nil {
			return nil, fmt.Errorf("read changelog %s handle: %w", label, err)
		}
		if handle != nil {
			handles = append(handles, handle)
		}
	}

	return handles, nil
}

// readChangelogByteIncrementHandle parses in-memory changelog increments.
func readChangelogByteIncrementHandle(br *binaryReader, kind KeyedStateHandleType, parseFull bool) (KeyedStateHandle, error) {
	startKeyGroup, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read changelog byte start key group: %w", err)
	}
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read changelog byte key group count: %w", err)
	}
	fromSeq, err := br.ReadInt64()
	if err != nil {
		return nil, fmt.Errorf("read changelog byte from seq: %w", err)
	}
	toSeq, err := br.ReadInt64()
	if err != nil {
		return nil, fmt.Errorf("read changelog byte to seq: %w", err)
	}
	changesCount, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read changelog byte changes count: %w", err)
	}
	if changesCount < 0 {
		return nil, fmt.Errorf("changelog byte changes count negative: %d", changesCount)
	}

	changes := make([]ChangelogStateChange, 0, changesCount)
	for i := int32(0); i < changesCount; i++ {
		keyGroup, err := br.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("read changelog byte key group: %w", err)
		}
		length, err := br.ReadInt32()
		if err != nil {
			return nil, fmt.Errorf("read changelog byte length: %w", err)
		}
		if !parseFull {
			if _, err := br.ReadBytes(int(length)); err != nil {
				return nil, fmt.Errorf("read changelog byte data: %w", err)
			}

			continue
		}
		data, err := br.ReadBytes(int(length))
		if err != nil {
			return nil, fmt.Errorf("read changelog byte data: %w", err)
		}
		changes = append(changes, ChangelogStateChange{
			KeyGroup: keyGroup,
			Data:     data,
		})
	}

	stateID, err := br.ReadUTF()
	if err != nil {
		return nil, fmt.Errorf("read changelog byte handle id: %w", err)
	}

	return ChangelogByteIncrementHandle{
		Type:          kind,
		StartKeyGroup: startKeyGroup,
		NumKeyGroups:  count,
		FromSeq:       fromSeq,
		ToSeq:         toSeq,
		Changes:       changes,
		HandleID:      stateID,
	}, nil
}

// readChangelogFileIncrementHandle parses file-based changelog increments.
// Type 10 (legacy) does not have a storageIdentifier field; it defaults to "filesystem".
// Type 13 (V2) includes the storageIdentifier field.
func readChangelogFileIncrementHandle(br *binaryReader, kind KeyedStateHandleType) (KeyedStateHandle, error) {
	isV2 := kind == KeyedStateHandleChangelogFileV2

	startKeyGroup, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read changelog file start key group: %w", err)
	}
	count, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read changelog file key group count: %w", err)
	}
	streamCount, err := br.ReadInt32()
	if err != nil {
		return nil, fmt.Errorf("read changelog file stream count: %w", err)
	}
	if streamCount < 0 {
		return nil, fmt.Errorf("changelog file stream count negative: %d", streamCount)
	}

	offsets := make([]ChangelogStreamOffset, 0, streamCount)
	for i := int32(0); i < streamCount; i++ {
		offset, err := br.ReadInt64()
		if err != nil {
			return nil, fmt.Errorf("read changelog file offset: %w", err)
		}
		handle, err := readStreamStateHandle(br)
		if err != nil {
			return nil, fmt.Errorf("read changelog file handle: %w", err)
		}
		offsets = append(offsets, ChangelogStreamOffset{
			Offset: offset,
			Handle: handle,
		})
	}

	stateSize, err := br.ReadInt64()
	if err != nil {
		return nil, fmt.Errorf("read changelog file state size: %w", err)
	}
	checkpointedSize, err := br.ReadInt64()
	if err != nil {
		return nil, fmt.Errorf("read changelog file checkpointed size: %w", err)
	}
	stateID, err := br.ReadUTF()
	if err != nil {
		return nil, fmt.Errorf("read changelog file handle id: %w", err)
	}

	// storageIdentifier is only present in V2 (type 13); for V1, default to "filesystem"
	storageID := "filesystem"
	if isV2 {
		storageID, err = br.ReadUTF()
		if err != nil {
			return nil, fmt.Errorf("read changelog file storage id: %w", err)
		}
	}

	return ChangelogFileIncrementHandle{
		Type:             kind,
		StartKeyGroup:    startKeyGroup,
		NumKeyGroups:     count,
		Offsets:          offsets,
		StateSize:        stateSize,
		CheckpointedSize: checkpointedSize,
		HandleID:         stateID,
		StorageID:        storageID,
	}, nil
}
