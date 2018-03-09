package mtp

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/cjongseok/slog"
	"math"
	"math/big"
	"reflect"
	"runtime"
	"strings"
	"time"
)

func GenerateNonce(size int) []byte {
	b := make([]byte, size)
	_, _ = rand.Read(b)
	return b
}

func GenerateMessageId() int64 {
	const nano = 1000 * 1000 * 1000
	//FIXME: Windows system clock has time resolution issue. https://github.com/golang/go/issues/17696
	//Remove the sleep when the issue is resolved.
	if strings.Contains(runtime.GOOS, "windows") {
		time.Sleep(2 * time.Millisecond)
	}
	unixnano := time.Now().UnixNano()

	return ((unixnano / nano) << 32) | ((unixnano % nano) & -4)
}

type EncodeBuf struct {
	buf []byte
}

func NewEncodeBuf(cap int) *EncodeBuf {
	if __debug&DEBUG_LEVEL_ENCODE_DETAILS != 0 {
		slog.Logln("Encode::NewBuf::", "cap=", cap)
	}
	return &EncodeBuf{make([]byte, 0, cap)}
}

func (e *EncodeBuf) Int(s int32) {
	e.buf = append(e.buf, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(e.buf[len(e.buf)-4:], uint32(s))
	if __debug&DEBUG_LEVEL_ENCODE_DETAILS != 0 {
		slog.Logln("Encode::Int::", s)
	}
}

func (e *EncodeBuf) UInt(s uint32) {
	e.buf = append(e.buf, 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(e.buf[len(e.buf)-4:], s)
	if __debug&DEBUG_LEVEL_ENCODE_DETAILS != 0 {
		slog.Logf("Encode::UInt::", "%d(0x%x)", s, s)
	}
}

func (e *EncodeBuf) Long(s int64) {
	e.buf = append(e.buf, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.LittleEndian.PutUint64(e.buf[len(e.buf)-8:], uint64(s))
	if __debug&DEBUG_LEVEL_ENCODE_DETAILS != 0 {
		slog.Logln("Encode::Long::", s)
	}
}

func (e *EncodeBuf) Double(s float64) {
	e.buf = append(e.buf, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.LittleEndian.PutUint64(e.buf[len(e.buf)-8:], math.Float64bits(s))
	if __debug&DEBUG_LEVEL_ENCODE_DETAILS != 0 {
		slog.Logln("Encode::Double::", s)
	}
}

func (e *EncodeBuf) String(s string) {
	e.StringBytes([]byte(s))
	if __debug&DEBUG_LEVEL_ENCODE_DETAILS != 0 {
		slog.Logln("Encode::String::", s)
	}
}

func (e *EncodeBuf) BigInt(s *big.Int) {
	e.StringBytes(s.Bytes())
	if __debug&DEBUG_LEVEL_ENCODE_DETAILS != 0 {
		slog.Logln("Encode::BigInt::", s)
	}
}

func (e *EncodeBuf) StringBytes(s []byte) {
	var res []byte
	size := len(s)
	if size < 254 {
		nl := 1 + size + (4-(size+1)%4)&3
		res = make([]byte, nl)
		res[0] = byte(size)
		copy(res[1:], s)

	} else {
		nl := 4 + size + (4-size%4)&3
		res = make([]byte, nl)
		binary.LittleEndian.PutUint32(res, uint32(size<<8|254))
		copy(res[4:], s)

	}
	e.buf = append(e.buf, res...)
	if __debug&DEBUG_LEVEL_ENCODE_DETAILS != 0 {
		slog.Logln("Encode::StringBytes::", s)
	}
}

func (e *EncodeBuf) Bytes(s []byte) {
	e.buf = append(e.buf, s...)
	if __debug&DEBUG_LEVEL_ENCODE_DETAILS != 0 {
		slog.Logln("Encode::Bytes::", s)
	}
}

func (e *EncodeBuf) VectorInt(v []int32) {
	x := make([]byte, 4+4+len(v)*4)
	binary.LittleEndian.PutUint32(x, crc_vector)
	binary.LittleEndian.PutUint32(x[4:], uint32(len(v)))
	i := 8
	for _, v := range v {
		binary.LittleEndian.PutUint32(x[i:], uint32(v))
		i += 4
	}
	e.buf = append(e.buf, x...)
	if __debug&DEBUG_LEVEL_ENCODE_DETAILS != 0 {
		slog.Logln("Encode::VectorInt::", v)
	}
}

func (e *EncodeBuf) VectorLong(v []int64) {
	x := make([]byte, 4+4+len(v)*8)
	binary.LittleEndian.PutUint32(x, crc_vector)
	binary.LittleEndian.PutUint32(x[4:], uint32(len(v)))
	i := 8
	for _, v := range v {
		binary.LittleEndian.PutUint64(x[i:], uint64(v))
		i += 8
	}
	e.buf = append(e.buf, x...)
	if __debug&DEBUG_LEVEL_ENCODE_DETAILS != 0 {
		slog.Logln("Encode::VectorLong::", v)
	}
}

func (e *EncodeBuf) VectorString(v []string) {
	x := make([]byte, 8)
	binary.LittleEndian.PutUint32(x, crc_vector)
	binary.LittleEndian.PutUint32(x[4:], uint32(len(v)))
	e.buf = append(e.buf, x...)
	for _, v := range v {
		e.String(v)
	}
	if __debug&DEBUG_LEVEL_ENCODE_DETAILS != 0 {
		slog.Logln("Encode::VectorString::", v)
	}
}

func (e *EncodeBuf) Vector(v []TL) {
	x := make([]byte, 8)
	binary.LittleEndian.PutUint32(x, crc_vector)
	binary.LittleEndian.PutUint32(x[4:], uint32(len(v)))
	e.buf = append(e.buf, x...)
	for _, v := range v {
		e.buf = append(e.buf, v.encode()...)
	}
	if __debug&DEBUG_LEVEL_ENCODE_DETAILS != 0 {
		slog.Logln("Encode::Vector::", v)
	}
}

func (e TL_msg_container) encode() []byte            { return nil }
func (e TL_resPQ) encode() []byte                    { return nil }
func (e TL_server_DH_params_ok) encode() []byte      { return nil }
func (e TL_server_DH_inner_data) encode() []byte     { return nil }
func (e TL_dh_gen_ok) encode() []byte                { return nil }
func (e TL_rpc_result) encode() []byte               { return nil }
func (e TL_rpc_error) encode() []byte                { return nil }
func (e TL_new_session_created) encode() []byte      { return nil }
func (e TL_bad_server_salt) encode() []byte          { return nil }
func (e TL_crc_bad_msg_notification) encode() []byte { return nil }

func (e TL_req_pq) encode() []byte {
	x := NewEncodeBuf(20)
	x.UInt(crc_req_pq)
	x.Bytes(e.nonce)
	return x.buf
}

func (e TL_p_q_inner_data) encode() []byte {
	x := NewEncodeBuf(256)
	x.UInt(crc_p_q_inner_data)
	x.BigInt(e.pq)
	x.BigInt(e.p)
	x.BigInt(e.q)
	x.Bytes(e.nonce)
	x.Bytes(e.server_nonce)
	x.Bytes(e.new_nonce)
	return x.buf
}

func (e TL_req_DH_params) encode() []byte {
	x := NewEncodeBuf(512)
	x.UInt(crc_req_DH_params)
	x.Bytes(e.nonce)
	x.Bytes(e.server_nonce)
	x.BigInt(e.p)
	x.BigInt(e.q)
	x.Long(int64(e.fp))
	x.StringBytes(e.encdata)
	return x.buf
}

func (e TL_client_DH_inner_data) encode() []byte {
	x := NewEncodeBuf(512)
	x.UInt(crc_client_DH_inner_data)
	x.Bytes(e.nonce)
	x.Bytes(e.server_nonce)
	x.Long(e.retry)
	x.BigInt(e.g_b)
	return x.buf
}

func (e TL_set_client_DH_params) encode() []byte {
	x := NewEncodeBuf(256)
	x.UInt(crc_set_client_DH_params)
	x.Bytes(e.nonce)
	x.Bytes(e.server_nonce)
	x.StringBytes(e.encdata)
	return x.buf
}

func (e TL_ping) encode() []byte {
	x := NewEncodeBuf(32)
	x.UInt(crc_ping)
	x.Long(e.ping_id)
	return x.buf
}

func (e TL_pong) encode() []byte {
	x := NewEncodeBuf(32)
	x.UInt(crc_pong)
	x.Long(e.msg_id)
	x.Long(e.ping_id)
	return x.buf
}

func (e TL_msgs_ack) encode() []byte {
	x := NewEncodeBuf(64)
	x.UInt(crc_msgs_ack)
	x.VectorLong(e.msgIds)
	return x.buf
}

func (e *EncodeBuf) FlaggedLong(flags, f int32, s int64) {
	bit := int32(1 << uint(f))
	if flags&bit == 0 {
		return
	}
	e.Long(s)
}
func (e *EncodeBuf) FlaggedDouble(flags, f int32, s float64) {
	bit := int32(1 << uint(f))
	if flags&bit == 0 {
		return
	}
	e.Double(s)
}
func (e *EncodeBuf) FlaggedInt(flags, f int32, s int32) {
	bit := int32(1 << uint(f))
	if flags&bit == 0 {
		return
	}
	e.Int(s)
}
func (e *EncodeBuf) FlaggedString(flags, f int32, s string) {
	bit := int32(1 << uint(f))
	if flags&bit == 0 {
		return
	}
	e.String(s)
}
func (e *EncodeBuf) FlaggedVector(flags, f int32, v []TL) {
	bit := int32(1 << uint(f))
	if flags&bit == 0 {
		return
	}
	e.Vector(v)
}
func (e *EncodeBuf) FlaggedObject(flags, f int32, o TL) {
	bit := int32(1 << uint(f))
	if flags&bit == 0 {
		return
	}
	e.Bytes(o.encode())
}
func (e *EncodeBuf) FlaggedStringBytes(flags, f int32, s []byte) {
	bit := int32(1 << uint(f))
	if flags&bit == 0 {
		return
	}
	e.StringBytes(s)
}
func (e *EncodeBuf) FlaggedVectorInt(flags, f int32, v []int32) {
	bit := int32(1 << uint(f))
	if flags&bit == 0 {
		return
	}
	e.VectorInt(v)
}
func (e *EncodeBuf) FlaggedVectorLong(flags, f int32, v []int64) {
	bit := int32(1 << uint(f))
	if flags&bit == 0 {
		return
	}
	e.VectorLong(v)
}
func (e *EncodeBuf) FlaggedVectorString(flags, f int32, v []string) {
	bit := int32(1 << uint(f))
	if flags&bit == 0 {
		return
	}
	e.VectorString(v)
}

func toTLslice(slice interface{}) []TL {
	if reflect.TypeOf(slice).Kind() != reflect.Slice {
		return nil
	}
	s := reflect.ValueOf(slice)
	if s.Len() < 1 {
		return nil
	}
	switch s.Index(0).Interface().(type) {
	case TL:
		tlslice := make([]TL, s.Len())
		for i := 0; i < s.Len(); i++ {
			tlslice[i] = s.Index(i).Interface().(TL)
		}
		return tlslice
	default:
		return nil
	}
}
