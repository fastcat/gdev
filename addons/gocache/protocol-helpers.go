package gocache

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
)

func DumpReq(req *Request) string {
	if req == nil {
		return "<nil>"
	}
	var buf bytes.Buffer
	buf.WriteString("Request{")
	fmt.Fprintf(&buf, "ID=%d Command=%s", req.ID, req.Command)
	if req.Command == CmdGet || req.Command == CmdPut {
		buf.WriteString(" ActionID=")
		buf.WriteString(hex.EncodeToString(req.ActionID))
	}
	if req.Command == CmdPut {
		buf.WriteString(" OutputID=")
		buf.WriteString(hex.EncodeToString(req.OutputID))
		buf.WriteString(" BodySize=")
		buf.WriteString(strconv.Itoa(int(req.BodySize)))
	}
	buf.WriteByte('}')
	return buf.String()
}
