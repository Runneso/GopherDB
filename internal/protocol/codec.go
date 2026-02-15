package protocol

import (
	"bufio"
	"encoding/json"
	"io"
)

type Codec struct {
	reader *bufio.Reader
	writer *bufio.Writer
}

func NewCodec(r io.Reader, w io.Writer) *Codec {
	return &Codec{
		reader: bufio.NewReader(r),
		writer: bufio.NewWriter(w),
	}
}

func (codec *Codec) ReadRequest() (*Request, error) {
	line, err := codec.reader.ReadBytes('\n')

	if err != nil {
		return nil, err
	}

	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		return nil, err
	}

	return &req, nil
}

func (codec *Codec) WriteResponse(resp *Response) error {
	data, err := json.Marshal(resp)

	if err != nil {
		return err
	}

	if _, err := codec.writer.Write(data); err != nil {
		return err
	}
	if err := codec.writer.WriteByte('\n'); err != nil {
		return err
	}

	return codec.writer.Flush()
}
