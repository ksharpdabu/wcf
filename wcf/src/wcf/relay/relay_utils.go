package relay

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"hash/crc32"
	"wcf/relay/msg"
)

func CheckRelayPacketReadyWithLength(data []byte, maxBytes uint32) (int, error) {
	if len(data) <= 6 {
		return 0, nil
	}
	total := binary.BigEndian.Uint32(data)
	if total > maxBytes {
		return -1, errors.New(fmt.Sprintf("should less than:%d, get:%d", maxBytes, total))
	}
	if data[4] != 0x2 {
		return -3, errors.New(fmt.Sprintf("packet delims err, start:%d", int(data[4])))
	}
	if len(data) < int(total) {
		return 0, nil
	}
	if data[4] != 0x2 || data[total-1] != 0x3 {
		return -2, errors.New(fmt.Sprintf("packet delims err, start:%d, end:%d", int(data[4]), int(data[total-1])))
	}
	return int(total), nil
}

//total + 0x2 + FrameBody + 0x3
func CheckRelayPacketReady(data []byte) (int, error) {
	return CheckRelayPacketReadyWithLength(data, ONE_PER_BUFFER_SIZE)
}

//单次只能一个包
func GetPacketData(data []byte) ([]byte, error) {
	total, err := CheckRelayPacketReady(data)
	if total <= 0 || err != nil {
		return nil, errors.New(fmt.Sprintf("check buf fail, v:%d, err:%v", total, err))
	}
	buf := data[5 : total-1]
	pb := &msg.DataPacket{}
	err = proto.Unmarshal(buf, pb)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("pb unmarshal fail, err:%v, data len:%d", err, len(data)))
	}
	crc := crc32.Checksum(pb.GetData(), crc32.IEEETable)
	if crc != pb.GetCrc() {
		return nil, errors.New(fmt.Sprintf("invalid data, crc not match, calc:%d, carry:%d", crc, pb.GetCrc()))
	}
	return pb.GetData(), nil
}

func BuildDataPacket(data []byte) []byte {
	pb := msg.DataPacket{}
	pb.Data = data
	pb.Crc = proto.Uint32(crc32.Checksum(data, crc32.IEEETable))
	raw, _ := proto.Marshal(&pb)
	buffer := make([]byte, 4+1+1+len(raw))
	binary.BigEndian.PutUint32(buffer, uint32(len(buffer)))
	buffer[4] = 0x2
	buffer[len(buffer)-1] = 0x3
	copy(buffer[5:], raw)
	return buffer
}

func BuildAuthReqMsg(config *RelayConfig) []byte {
	req := msg.AuthMsgReq{}
	req.Pwd = proto.String(config.Pwd)
	req.User = proto.String(config.User)
	req.Address = &msg.RelayAddress{
		AddressType: proto.Int32(config.Address.AddrType),
		Name:        proto.String(config.Address.Name),
		Port:        proto.Uint32(uint32(config.Address.Port)),
	}
	req.OpType = proto.Int32(config.RelayType)

	data, _ := proto.Marshal(&req)
	return data
}

func BuildAuthRspMsg(result int32, token uint32) []byte {
	pb := msg.AuthMsgRsp{}
	pb.Token = proto.Uint32(token)
	pb.Result = proto.Int32(result)
	data, _ := proto.Marshal(&pb)
	return data
}
