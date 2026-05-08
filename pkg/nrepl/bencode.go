package nrepl

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
)

// BencodeEncode encodes a value to bencode format.
// Supported types: string, int/int64, []interface{}, map[string]interface{}.
func BencodeEncode(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := bencodeWrite(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func bencodeWrite(buf *bytes.Buffer, v interface{}) error {
	switch val := v.(type) {
	case string:
		buf.WriteString(strconv.Itoa(len(val)))
		buf.WriteByte(':')
		buf.WriteString(val)
	case int:
		buf.WriteByte('i')
		buf.WriteString(strconv.Itoa(val))
		buf.WriteByte('e')
	case int64:
		buf.WriteByte('i')
		buf.WriteString(strconv.FormatInt(val, 10))
		buf.WriteByte('e')
	case []interface{}:
		buf.WriteByte('l')
		for _, item := range val {
			if err := bencodeWrite(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte('e')
	case map[string]interface{}:
		buf.WriteByte('d')
		// bencode requires sorted keys
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if err := bencodeWrite(buf, k); err != nil {
				return err
			}
			if err := bencodeWrite(buf, val[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('e')
	default:
		return fmt.Errorf("bencode: unsupported type %T", v)
	}
	return nil
}

// BencodeDecode reads one bencoded value from the reader.
// Returns string, int64, []interface{}, or map[string]interface{}.
func BencodeDecode(r io.Reader) (interface{}, error) {
	return bencodeRead(newByteReader(r))
}

func bencodeRead(r byteReaderInterface) (interface{}, error) {
	b, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch {
	case b == 'i':
		return bencodeReadInt(r)
	case b == 'l':
		return bencodeReadList(r)
	case b == 'd':
		return bencodeReadDict(r)
	case b >= '0' && b <= '9':
		return bencodeReadString(r, b)
	default:
		return nil, fmt.Errorf("bencode: unexpected byte %q", b)
	}
}

func bencodeReadInt(r byteReaderInterface) (int64, error) {
	var buf bytes.Buffer
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		if b == 'e' {
			break
		}
		buf.WriteByte(b)
	}
	return strconv.ParseInt(buf.String(), 10, 64)
}

func bencodeReadString(r byteReaderInterface, first byte) (string, error) {
	var lenBuf bytes.Buffer
	lenBuf.WriteByte(first)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		if b == ':' {
			break
		}
		lenBuf.WriteByte(b)
	}
	length, err := strconv.Atoi(lenBuf.String())
	if err != nil {
		return "", fmt.Errorf("bencode: invalid string length: %w", err)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return "", err
	}
	return string(data), nil
}

func bencodeReadList(r byteReaderInterface) ([]interface{}, error) {
	var list []interface{}
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == 'e' {
			return list, nil
		}
		if err := r.UnreadByte(); err != nil {
			return nil, err
		}
		val, err := bencodeRead(r)
		if err != nil {
			return nil, err
		}
		list = append(list, val)
	}
}

func bencodeReadDict(r byteReaderInterface) (map[string]interface{}, error) {
	dict := make(map[string]interface{})
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == 'e' {
			return dict, nil
		}
		if err := r.UnreadByte(); err != nil {
			return nil, err
		}
		keyVal, err := bencodeRead(r)
		if err != nil {
			return nil, err
		}
		key, ok := keyVal.(string)
		if !ok {
			return nil, fmt.Errorf("bencode: dict key must be string, got %T", keyVal)
		}
		val, err := bencodeRead(r)
		if err != nil {
			return nil, err
		}
		dict[key] = val
	}
}

// byteReaderInterface combines io.Reader with byte-level reads.
type byteReaderInterface interface {
	io.Reader
	ReadByte() (byte, error)
	UnreadByte() error
}

// newByteReader wraps an io.Reader to provide ReadByte/UnreadByte if needed.
func newByteReader(r io.Reader) byteReaderInterface {
	if br, ok := r.(byteReaderInterface); ok {
		return br
	}
	return &singleByteReader{r: r}
}

type singleByteReader struct {
	r      io.Reader
	buf    [1]byte
	hasRev bool
}

func (s *singleByteReader) Read(p []byte) (int, error) {
	if s.hasRev && len(p) > 0 {
		p[0] = s.buf[0]
		s.hasRev = false
		if len(p) == 1 {
			return 1, nil
		}
		n, err := s.r.Read(p[1:])
		return n + 1, err
	}
	return s.r.Read(p)
}

func (s *singleByteReader) ReadByte() (byte, error) {
	if s.hasRev {
		s.hasRev = false
		return s.buf[0], nil
	}
	_, err := io.ReadFull(s.r, s.buf[:])
	return s.buf[0], err
}

func (s *singleByteReader) UnreadByte() error {
	s.hasRev = true
	return nil
}
