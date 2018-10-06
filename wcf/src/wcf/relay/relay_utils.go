package relay

import (
	"encoding/binary"
	"wcf/relay/msg"
	"github.com/golang/protobuf/proto"
	"hash/crc32"
	"errors"
	"fmt"
)

//total + 0x2 + FrameBody + 0x3
func CheckRelayPacketReay(data []byte) (int, error) {
	if len(data) <= 6 {
		return 0, nil
	}
	total := binary.BigEndian.Uint32(data)
	if total > ONE_PER_BUFFER_SIZE {
		return -1, errors.New(fmt.Sprintf("should less than:%d, get:%d", MAX_BYTE_PER_PACKET, total))
	}
	if len(data) < int(total) {
		return 0, nil
	}
	if data[4] != 0x2 || data[total - 1] != 0x3 {
		return -2, errors.New(fmt.Sprintf("packet delims err, start:%d, end:%d", int(data[4]), int(data[total - 1])))
	}
	return int(total), nil
}

//单次只能一个包
func GetPacketData(data []byte) ([]byte, error) {
	total, err := CheckRelayPacketReay(data)
	if total <= 0 || err != nil {
		return nil, errors.New(fmt.Sprintf("check buf fail, v:%d, err:%v", total, err))
	}
	buf := data[5:total - 1]
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

func BuildDataPacket(data []byte) ([]byte, error) {
	pb := msg.DataPacket{}
	pb.Data = data
	pb.Crc = proto.Uint32(crc32.Checksum(data, crc32.IEEETable))
	raw, err := proto.Marshal(&pb)
	if err != nil {
		return nil, err
	}
	buffer := make([]byte, 4 + 1 + 1 + len(raw))
	binary.BigEndian.PutUint32(buffer, uint32(len(buffer)))
	buffer[4] = 0x2
	buffer[len(buffer) - 1] = 0x3
	copy(buffer[5:], raw)
	return buffer, nil
}