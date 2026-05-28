package checkpoint

import "fmt"

// readStreamStateHandle parses a stream state handle from the binary stream.
func readStreamStateHandle(br *binaryReader) (*StreamStateHandle, error) {
	var err error
	var kind byte

	if kind, err = br.ReadByte(); err != nil {
		return nil, fmt.Errorf("read stream state handle type: %w", err)
	}

	h := &StreamStateHandle{Type: StreamHandleType(kind)}
	switch h.Type {
	case StreamHandleNull:
		return nil, nil
	case StreamHandleByteStream:
		return readByteStreamHandle(br, h)
	case StreamHandleFile:
		return readFileStreamHandle(br, h)
	case StreamHandleRelative:
		return readRelativeStreamHandle(br, h)
	case StreamHandleSegmentFile:
		return readSegmentStreamHandle(br, h)
	case StreamHandleEmptySegment:
		return h, nil
	default:
		return nil, fmt.Errorf("unsupported stream state handle type %d", kind)
	}
}

func readByteStreamHandle(br *binaryReader, h *StreamStateHandle) (*StreamStateHandle, error) {
	var err error
	var name string
	var length int32
	var data []byte

	if name, err = br.ReadUTF(); err != nil {
		return nil, fmt.Errorf("read byte stream handle name: %w", err)
	}

	if length, err = br.ReadInt32(); err != nil {
		return nil, fmt.Errorf("read byte stream handle length: %w", err)
	}

	if data, err = br.ReadBytes(int(length)); err != nil {
		return nil, fmt.Errorf("read byte stream handle data: %w", err)
	}

	h.Name = name
	h.Size = int64(length)
	h.Data = data

	return h, nil
}

func readFileStreamHandle(br *binaryReader, h *StreamStateHandle) (*StreamStateHandle, error) {
	var err error
	var size int64
	var path string

	if size, err = br.ReadInt64(); err != nil {
		return nil, fmt.Errorf("read file handle size: %w", err)
	}

	if path, err = br.ReadUTF(); err != nil {
		return nil, fmt.Errorf("read file handle path: %w", err)
	}

	h.Size = size
	h.Path = path

	return h, nil
}

func readRelativeStreamHandle(br *binaryReader, h *StreamStateHandle) (*StreamStateHandle, error) {
	var err error
	var path string
	var size int64

	if path, err = br.ReadUTF(); err != nil {
		return nil, fmt.Errorf("read relative handle path: %w", err)
	}

	if size, err = br.ReadInt64(); err != nil {
		return nil, fmt.Errorf("read relative handle size: %w", err)
	}

	h.Path = path
	h.Size = size

	return h, nil
}

func readSegmentStreamHandle(br *binaryReader, h *StreamStateHandle) (*StreamStateHandle, error) {
	var err error
	var start int64
	var size int64
	var scope int32
	var path string
	var logicalID string

	if start, err = br.ReadInt64(); err != nil {
		return nil, fmt.Errorf("read segment handle start: %w", err)
	}

	if size, err = br.ReadInt64(); err != nil {
		return nil, fmt.Errorf("read segment handle size: %w", err)
	}

	if scope, err = br.ReadInt32(); err != nil {
		return nil, fmt.Errorf("read segment handle scope: %w", err)
	}

	if path, err = br.ReadUTF(); err != nil {
		return nil, fmt.Errorf("read segment handle path: %w", err)
	}

	if logicalID, err = br.ReadUTF(); err != nil {
		return nil, fmt.Errorf("read segment handle logical id: %w", err)
	}

	h.StartPos = start
	h.Size = size
	h.Scope = scope
	h.Path = path
	h.LogicalID = logicalID

	return h, nil
}
