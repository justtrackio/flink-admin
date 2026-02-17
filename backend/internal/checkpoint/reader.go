package checkpoint

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

type binaryReader struct {
	r *bufio.Reader
}

// newBinaryReader wraps the reader with buffered, big-endian helpers.
func newBinaryReader(reader io.Reader) *binaryReader {
	return &binaryReader{r: bufio.NewReader(reader)}
}

// ReadByte reads a single byte from the stream.
func (br *binaryReader) ReadByte() (byte, error) {
	b, err := br.r.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("read byte: %w", err)
	}

	return b, nil
}

// ReadBool reads a single byte and treats non-zero as true.
func (br *binaryReader) ReadBool() (bool, error) {
	b, err := br.ReadByte()
	if err != nil {
		return false, err
	}

	return b != 0, nil
}

// ReadBytes reads an exact number of bytes from the stream.
func (br *binaryReader) ReadBytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("read bytes: negative length %d", n)
	}

	buf := make([]byte, n)
	if n == 0 {
		return buf, nil
	}

	if _, err := io.ReadFull(br.r, buf); err != nil {
		return nil, fmt.Errorf("read bytes: %w", err)
	}

	return buf, nil
}

// ReadInt32 reads a big-endian int32 from the stream.
func (br *binaryReader) ReadInt32() (int32, error) {
	buf, err := br.ReadBytes(4)
	if err != nil {
		return 0, err
	}

	return int32(binary.BigEndian.Uint32(buf)), nil
}

// ReadUint32 reads a big-endian uint32 from the stream.
func (br *binaryReader) ReadUint32() (uint32, error) {
	buf, err := br.ReadBytes(4)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(buf), nil
}

// ReadInt64 reads a big-endian int64 from the stream.
func (br *binaryReader) ReadInt64() (int64, error) {
	buf, err := br.ReadBytes(8)
	if err != nil {
		return 0, err
	}

	return int64(binary.BigEndian.Uint64(buf)), nil
}

// ReadUTF reads a Java modified UTF-8 string (DataOutputStream.writeUTF).
func (br *binaryReader) ReadUTF() (string, error) {
	length, err := br.ReadUint16()
	if err != nil {
		return "", fmt.Errorf("read utf length: %w", err)
	}

	if length == 0 {
		return "", nil
	}

	buf, err := br.ReadBytes(int(length))
	if err != nil {
		return "", fmt.Errorf("read utf bytes: %w", err)
	}

	return decodeModifiedUTF8(buf)
}

// ReadUint16 reads a big-endian uint16 from the stream.
func (br *binaryReader) ReadUint16() (uint16, error) {
	buf, err := br.ReadBytes(2)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(buf), nil
}

// decodeModifiedUTF8 decodes Java's modified UTF-8 encoding.
func decodeModifiedUTF8(buf []byte) (string, error) {
	out := make([]rune, 0, len(buf))
	for i := 0; i < len(buf); {
		b := buf[i]
		switch {
		case b>>7 == 0:
			out = append(out, rune(b))
			i++
		case b>>5 == 0x6:
			if i+1 >= len(buf) {
				return "", fmt.Errorf("decode utf: invalid 2-byte sequence")
			}
			b2 := buf[i+1]
			if b == 0xC0 && b2 == 0x80 {
				out = append(out, 0)
			} else {
				r := rune(b&0x1F)<<6 | rune(b2&0x3F)
				out = append(out, r)
			}
			i += 2
		case b>>4 == 0xE:
			if i+2 >= len(buf) {
				return "", fmt.Errorf("decode utf: invalid 3-byte sequence")
			}
			b2 := buf[i+1]
			b3 := buf[i+2]
			r := rune(b&0x0F)<<12 | rune(b2&0x3F)<<6 | rune(b3&0x3F)
			out = append(out, r)
			i += 3
		default:
			return "", fmt.Errorf("decode utf: unsupported sequence")
		}
	}

	return string(out), nil
}
