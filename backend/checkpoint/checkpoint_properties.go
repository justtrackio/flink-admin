package checkpoint

import "bytes"

const (
	javaStreamMagic   = 0xACED
	javaStreamVersion = 0x0005
)

type checkpointPropertiesParser struct{}

// newCheckpointPropertiesParser creates a parser for trailing checkpoint properties bytes.
func newCheckpointPropertiesParser() *checkpointPropertiesParser {
	return &checkpointPropertiesParser{}
}

// parse inspects raw bytes for Java serialization markers and known tokens.
func (p *checkpointPropertiesParser) parse(raw []byte) *CheckpointProperties {
	if len(raw) < 4 {
		return nil
	}

	if binaryBigEndianUint16(raw[0:2]) != javaStreamMagic || binaryBigEndianUint16(raw[2:4]) != javaStreamVersion {
		return nil
	}

	properties := &CheckpointProperties{}

	if name := scanForToken(raw, []byte("CheckpointType")); name != "" {
		properties.CheckpointType = name
	}
	if name := scanForToken(raw, []byte("SharingFilesStrategy")); name != "" {
		properties.SharingStrategy = name
	}
	if name := scanForToken(raw, []byte("CheckpointProperties")); name != "" {
		properties.Source = name
	}

	if properties.CheckpointType == "" && properties.SharingStrategy == "" && properties.Source == "" {
		return nil
	}

	return properties
}

// scanForToken searches the raw payload for a byte token.
func scanForToken(raw []byte, token []byte) string {
	idx := bytes.Index(raw, token)
	if idx == -1 {
		return ""
	}

	return string(token)
}

// binaryBigEndianUint16 reads a big-endian uint16 from a byte slice.
func binaryBigEndianUint16(data []byte) uint16 {
	if len(data) < 2 {
		return 0
	}
	return uint16(data[0])<<8 | uint16(data[1])
}
