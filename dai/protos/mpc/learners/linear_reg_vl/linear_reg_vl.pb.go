// Code generated by protoc-gen-go. DO NOT EDIT.
// source: mpc/learners/linear_reg_vl/linear_reg_vl.proto

package linear_reg_vl

import (
	fmt "fmt"
	mpc "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

//MessageType defines the type of message with which communicate with nodes in cluster,
// and in some way it indicates the phase of learning
//Some types are for local message which is not passed between nodes
type MessageType int32

const (
	MessageType_MsgPsiEnc                MessageType = 0
	MessageType_MsgPsiAskReEnc           MessageType = 1
	MessageType_MsgPsiReEnc              MessageType = 2
	MessageType_MsgPsiIntersect          MessageType = 3
	MessageType_MsgTrainHup              MessageType = 4
	MessageType_MsgHomoPubkey            MessageType = 5
	MessageType_MsgTrainLoop             MessageType = 6
	MessageType_MsgTrainCalLocalGradCost MessageType = 7
	MessageType_MsgTrainPartBytes        MessageType = 8
	MessageType_MsgTrainCalEncGradCost   MessageType = 9
	MessageType_MsgTrainEncGradCost      MessageType = 10
	MessageType_MsgTrainDecLocalGradCost MessageType = 11
	MessageType_MsgTrainGradAndCost      MessageType = 12
	MessageType_MsgTrainUpdCostGrad      MessageType = 13
	MessageType_MsgTrainStatus           MessageType = 14
	MessageType_MsgTrainCheckStatus      MessageType = 15
	MessageType_MsgTrainModels           MessageType = 16
	MessageType_MsgPredictHup            MessageType = 17
	MessageType_MsgPredictPart           MessageType = 18
	MessageType_MsgPredictSum            MessageType = 19
)

var MessageType_name = map[int32]string{
	0:  "MsgPsiEnc",
	1:  "MsgPsiAskReEnc",
	2:  "MsgPsiReEnc",
	3:  "MsgPsiIntersect",
	4:  "MsgTrainHup",
	5:  "MsgHomoPubkey",
	6:  "MsgTrainLoop",
	7:  "MsgTrainCalLocalGradCost",
	8:  "MsgTrainPartBytes",
	9:  "MsgTrainCalEncGradCost",
	10: "MsgTrainEncGradCost",
	11: "MsgTrainDecLocalGradCost",
	12: "MsgTrainGradAndCost",
	13: "MsgTrainUpdCostGrad",
	14: "MsgTrainStatus",
	15: "MsgTrainCheckStatus",
	16: "MsgTrainModels",
	17: "MsgPredictHup",
	18: "MsgPredictPart",
	19: "MsgPredictSum",
}

var MessageType_value = map[string]int32{
	"MsgPsiEnc":                0,
	"MsgPsiAskReEnc":           1,
	"MsgPsiReEnc":              2,
	"MsgPsiIntersect":          3,
	"MsgTrainHup":              4,
	"MsgHomoPubkey":            5,
	"MsgTrainLoop":             6,
	"MsgTrainCalLocalGradCost": 7,
	"MsgTrainPartBytes":        8,
	"MsgTrainCalEncGradCost":   9,
	"MsgTrainEncGradCost":      10,
	"MsgTrainDecLocalGradCost": 11,
	"MsgTrainGradAndCost":      12,
	"MsgTrainUpdCostGrad":      13,
	"MsgTrainStatus":           14,
	"MsgTrainCheckStatus":      15,
	"MsgTrainModels":           16,
	"MsgPredictHup":            17,
	"MsgPredictPart":           18,
	"MsgPredictSum":            19,
}

func (x MessageType) String() string {
	return proto.EnumName(MessageType_name, int32(x))
}

func (MessageType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_93418147b2b47a20, []int{0}
}

type Message struct {
	Type                 MessageType                `protobuf:"varint,1,opt,name=type,proto3,enum=linear_reg_vl.MessageType" json:"type,omitempty"`
	To                   string                     `protobuf:"bytes,2,opt,name=to,proto3" json:"to,omitempty"`
	From                 string                     `protobuf:"bytes,3,opt,name=from,proto3" json:"from,omitempty"`
	LoopRound            uint64                     `protobuf:"varint,4,opt,name=loopRound,proto3" json:"loopRound,omitempty"`
	VlLPsiReEncIDsReq    *mpc.VLPsiReEncIDsRequest  `protobuf:"bytes,5,opt,name=vlLPsiReEncIDsReq,proto3" json:"vlLPsiReEncIDsReq,omitempty"`
	VlLPsiReEncIDsResp   *mpc.VLPsiReEncIDsResponse `protobuf:"bytes,6,opt,name=vlLPsiReEncIDsResp,proto3" json:"vlLPsiReEncIDsResp,omitempty"`
	HomoPubkey           []byte                     `protobuf:"bytes,7,opt,name=homoPubkey,proto3" json:"homoPubkey,omitempty"`
	PartBytes            []byte                     `protobuf:"bytes,8,opt,name=PartBytes,proto3" json:"PartBytes,omitempty"`
	EncGradFromOther     []byte                     `protobuf:"bytes,9,opt,name=encGradFromOther,proto3" json:"encGradFromOther,omitempty"`
	EncCostFromOther     []byte                     `protobuf:"bytes,10,opt,name=encCostFromOther,proto3" json:"encCostFromOther,omitempty"`
	GradBytes            []byte                     `protobuf:"bytes,11,opt,name=gradBytes,proto3" json:"gradBytes,omitempty"`
	CostBytes            []byte                     `protobuf:"bytes,12,opt,name=costBytes,proto3" json:"costBytes,omitempty"`
	Stopped              bool                       `protobuf:"varint,13,opt,name=stopped,proto3" json:"stopped,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                   `json:"-"`
	XXX_unrecognized     []byte                     `json:"-"`
	XXX_sizecache        int32                      `json:"-"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto.CompactTextString(m) }
func (*Message) ProtoMessage()    {}
func (*Message) Descriptor() ([]byte, []int) {
	return fileDescriptor_93418147b2b47a20, []int{0}
}

func (m *Message) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Message.Unmarshal(m, b)
}
func (m *Message) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Message.Marshal(b, m, deterministic)
}
func (m *Message) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message.Merge(m, src)
}
func (m *Message) XXX_Size() int {
	return xxx_messageInfo_Message.Size(m)
}
func (m *Message) XXX_DiscardUnknown() {
	xxx_messageInfo_Message.DiscardUnknown(m)
}

var xxx_messageInfo_Message proto.InternalMessageInfo

func (m *Message) GetType() MessageType {
	if m != nil {
		return m.Type
	}
	return MessageType_MsgPsiEnc
}

func (m *Message) GetTo() string {
	if m != nil {
		return m.To
	}
	return ""
}

func (m *Message) GetFrom() string {
	if m != nil {
		return m.From
	}
	return ""
}

func (m *Message) GetLoopRound() uint64 {
	if m != nil {
		return m.LoopRound
	}
	return 0
}

func (m *Message) GetVlLPsiReEncIDsReq() *mpc.VLPsiReEncIDsRequest {
	if m != nil {
		return m.VlLPsiReEncIDsReq
	}
	return nil
}

func (m *Message) GetVlLPsiReEncIDsResp() *mpc.VLPsiReEncIDsResponse {
	if m != nil {
		return m.VlLPsiReEncIDsResp
	}
	return nil
}

func (m *Message) GetHomoPubkey() []byte {
	if m != nil {
		return m.HomoPubkey
	}
	return nil
}

func (m *Message) GetPartBytes() []byte {
	if m != nil {
		return m.PartBytes
	}
	return nil
}

func (m *Message) GetEncGradFromOther() []byte {
	if m != nil {
		return m.EncGradFromOther
	}
	return nil
}

func (m *Message) GetEncCostFromOther() []byte {
	if m != nil {
		return m.EncCostFromOther
	}
	return nil
}

func (m *Message) GetGradBytes() []byte {
	if m != nil {
		return m.GradBytes
	}
	return nil
}

func (m *Message) GetCostBytes() []byte {
	if m != nil {
		return m.CostBytes
	}
	return nil
}

func (m *Message) GetStopped() bool {
	if m != nil {
		return m.Stopped
	}
	return false
}

type PredictMessage struct {
	Type                 MessageType                `protobuf:"varint,1,opt,name=type,proto3,enum=linear_reg_vl.MessageType" json:"type,omitempty"`
	To                   string                     `protobuf:"bytes,2,opt,name=to,proto3" json:"to,omitempty"`
	From                 string                     `protobuf:"bytes,3,opt,name=from,proto3" json:"from,omitempty"`
	VlLPsiReEncIDsReq    *mpc.VLPsiReEncIDsRequest  `protobuf:"bytes,4,opt,name=vlLPsiReEncIDsReq,proto3" json:"vlLPsiReEncIDsReq,omitempty"`
	VlLPsiReEncIDsResp   *mpc.VLPsiReEncIDsResponse `protobuf:"bytes,5,opt,name=vlLPsiReEncIDsResp,proto3" json:"vlLPsiReEncIDsResp,omitempty"`
	PredictPart          []float64                  `protobuf:"fixed64,6,rep,packed,name=predictPart,proto3" json:"predictPart,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                   `json:"-"`
	XXX_unrecognized     []byte                     `json:"-"`
	XXX_sizecache        int32                      `json:"-"`
}

func (m *PredictMessage) Reset()         { *m = PredictMessage{} }
func (m *PredictMessage) String() string { return proto.CompactTextString(m) }
func (*PredictMessage) ProtoMessage()    {}
func (*PredictMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_93418147b2b47a20, []int{1}
}

func (m *PredictMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PredictMessage.Unmarshal(m, b)
}
func (m *PredictMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PredictMessage.Marshal(b, m, deterministic)
}
func (m *PredictMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PredictMessage.Merge(m, src)
}
func (m *PredictMessage) XXX_Size() int {
	return xxx_messageInfo_PredictMessage.Size(m)
}
func (m *PredictMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_PredictMessage.DiscardUnknown(m)
}

var xxx_messageInfo_PredictMessage proto.InternalMessageInfo

func (m *PredictMessage) GetType() MessageType {
	if m != nil {
		return m.Type
	}
	return MessageType_MsgPsiEnc
}

func (m *PredictMessage) GetTo() string {
	if m != nil {
		return m.To
	}
	return ""
}

func (m *PredictMessage) GetFrom() string {
	if m != nil {
		return m.From
	}
	return ""
}

func (m *PredictMessage) GetVlLPsiReEncIDsReq() *mpc.VLPsiReEncIDsRequest {
	if m != nil {
		return m.VlLPsiReEncIDsReq
	}
	return nil
}

func (m *PredictMessage) GetVlLPsiReEncIDsResp() *mpc.VLPsiReEncIDsResponse {
	if m != nil {
		return m.VlLPsiReEncIDsResp
	}
	return nil
}

func (m *PredictMessage) GetPredictPart() []float64 {
	if m != nil {
		return m.PredictPart
	}
	return nil
}

func init() {
	proto.RegisterEnum("linear_reg_vl.MessageType", MessageType_name, MessageType_value)
	proto.RegisterType((*Message)(nil), "linear_reg_vl.Message")
	proto.RegisterType((*PredictMessage)(nil), "linear_reg_vl.PredictMessage")
}

func init() {
	proto.RegisterFile("mpc/learners/linear_reg_vl/linear_reg_vl.proto", fileDescriptor_93418147b2b47a20)
}

var fileDescriptor_93418147b2b47a20 = []byte{
	// 607 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xb4, 0x94, 0xcb, 0x6e, 0xd3, 0x40,
	0x14, 0x86, 0x71, 0x92, 0x26, 0xcd, 0xc9, 0xa5, 0x93, 0x53, 0x01, 0x43, 0x54, 0x21, 0xab, 0x2b,
	0xab, 0x8b, 0x44, 0x2a, 0x4f, 0xd0, 0x1b, 0x6d, 0x51, 0x23, 0x22, 0xb7, 0x20, 0xc4, 0xa6, 0x72,
	0xed, 0x43, 0x62, 0xd5, 0xf1, 0x0c, 0x33, 0xe3, 0x4a, 0x79, 0x16, 0x9e, 0x86, 0xf7, 0x62, 0x81,
	0x7c, 0x49, 0x62, 0xb7, 0x65, 0x83, 0x60, 0x13, 0x65, 0xbe, 0xff, 0x3f, 0x73, 0x7c, 0x2e, 0x1a,
	0x18, 0x2d, 0xa4, 0x3f, 0x8e, 0xc8, 0x53, 0x31, 0x29, 0x3d, 0x8e, 0xc2, 0x98, 0x3c, 0x75, 0xab,
	0x68, 0x76, 0xfb, 0x10, 0x55, 0x4f, 0x23, 0xa9, 0x84, 0x11, 0xd8, 0xab, 0xc0, 0x61, 0x2f, 0x0d,
	0x97, 0x3a, 0xcc, 0xd5, 0xfd, 0x5f, 0x75, 0x68, 0x4d, 0x48, 0x6b, 0x6f, 0x46, 0x38, 0x82, 0x86,
	0x59, 0x4a, 0xe2, 0x96, 0x6d, 0x39, 0xfd, 0xc3, 0xe1, 0xa8, 0x7a, 0x5b, 0xe1, 0xba, 0x59, 0x4a,
	0x72, 0x33, 0x1f, 0xf6, 0xa1, 0x66, 0x04, 0xaf, 0xd9, 0x96, 0xd3, 0x76, 0x6b, 0x46, 0x20, 0x42,
	0xe3, 0x9b, 0x12, 0x0b, 0x5e, 0xcf, 0x48, 0xf6, 0x1f, 0xf7, 0xa0, 0x1d, 0x09, 0x21, 0x5d, 0x91,
	0xc4, 0x01, 0x6f, 0xd8, 0x96, 0xd3, 0x70, 0x37, 0x00, 0xcf, 0x61, 0xf0, 0x10, 0x5d, 0x4d, 0x75,
	0xe8, 0xd2, 0x59, 0xec, 0x5f, 0x9e, 0x6a, 0x97, 0xbe, 0xf3, 0x2d, 0xdb, 0x72, 0x3a, 0x87, 0x6f,
	0xd2, 0x3a, 0x47, 0x9f, 0x1f, 0x89, 0x09, 0x69, 0xe3, 0x3e, 0x8d, 0xc1, 0x0f, 0x80, 0x8f, 0xa1,
	0x96, 0xbc, 0x99, 0xdd, 0x34, 0x7c, 0xee, 0x26, 0x2d, 0x45, 0xac, 0xc9, 0x7d, 0x26, 0x0a, 0xdf,
	0x02, 0xcc, 0xc5, 0x42, 0x4c, 0x93, 0xbb, 0x7b, 0x5a, 0xf2, 0x96, 0x6d, 0x39, 0x5d, 0xb7, 0x44,
	0xd2, 0x92, 0xa6, 0x9e, 0x32, 0xc7, 0x4b, 0x43, 0x9a, 0x6f, 0x67, 0xf2, 0x06, 0xe0, 0x01, 0x30,
	0x8a, 0xfd, 0x73, 0xe5, 0x05, 0xef, 0x95, 0x58, 0x7c, 0x34, 0x73, 0x52, 0xbc, 0x9d, 0x99, 0x9e,
	0xf0, 0xc2, 0x7b, 0x22, 0xb4, 0xd9, 0x78, 0x61, 0xed, 0xad, 0xf0, 0x34, 0xeb, 0x4c, 0x79, 0x41,
	0x9e, 0xb5, 0x93, 0x67, 0x5d, 0x83, 0x54, 0xf5, 0x85, 0x2e, 0xbe, 0xa9, 0x9b, 0xab, 0x6b, 0x80,
	0x1c, 0x5a, 0xda, 0x08, 0x29, 0x29, 0xe0, 0x3d, 0xdb, 0x72, 0xb6, 0xdd, 0xd5, 0x71, 0xff, 0x47,
	0x0d, 0xfa, 0x53, 0x45, 0x41, 0xe8, 0x9b, 0xff, 0xb9, 0x05, 0xcf, 0xce, 0xb9, 0xf1, 0xcf, 0xe6,
	0xbc, 0xf5, 0x57, 0x73, 0xb6, 0xa1, 0x23, 0xf3, 0xd2, 0xd3, 0xe9, 0xf1, 0xa6, 0x5d, 0x77, 0x2c,
	0xb7, 0x8c, 0x0e, 0x7e, 0xd6, 0xa1, 0x53, 0x2a, 0x18, 0x7b, 0xd0, 0x9e, 0xe8, 0xd9, 0x54, 0x87,
	0x67, 0xb1, 0xcf, 0x5e, 0x20, 0x42, 0x3f, 0x3f, 0x1e, 0xe9, 0xfb, 0xec, 0x66, 0x66, 0xe1, 0x0e,
	0x74, 0x72, 0x96, 0x83, 0x1a, 0xee, 0xc2, 0x4e, 0x0e, 0x2e, 0x63, 0x43, 0x4a, 0x93, 0x6f, 0x58,
	0xbd, 0x70, 0xdd, 0x28, 0x2f, 0x8c, 0x2f, 0x12, 0xc9, 0x1a, 0x38, 0x80, 0xde, 0x44, 0xcf, 0x2e,
	0xd6, 0x4b, 0xc6, 0xb6, 0x90, 0x41, 0x77, 0xe5, 0xb9, 0x12, 0x42, 0xb2, 0x26, 0xee, 0x01, 0x5f,
	0x91, 0x13, 0x2f, 0xba, 0x12, 0xbe, 0x17, 0xa5, 0xfb, 0x94, 0xee, 0x09, 0x6b, 0xe1, 0x4b, 0x18,
	0xac, 0xd4, 0xf5, 0x36, 0xb2, 0x6d, 0x1c, 0xc2, 0xab, 0x52, 0xd0, 0x59, 0xbe, 0x82, 0x59, 0x48,
	0x1b, 0x5f, 0xc3, 0xee, 0x4a, 0x2b, 0x0b, 0x50, 0xce, 0x74, 0x4a, 0x7e, 0x35, 0x53, 0xa7, 0x1c,
	0x96, 0xd2, 0xa3, 0x38, 0x17, 0xba, 0x65, 0xe1, 0x93, 0xcc, 0x60, 0xaa, 0xb3, 0x5e, 0xd1, 0xa9,
	0x4c, 0xb8, 0x36, 0x9e, 0x49, 0x34, 0xeb, 0x97, 0xcd, 0x27, 0x73, 0xf2, 0xef, 0x0b, 0x61, 0xa7,
	0x6c, 0x9e, 0x88, 0x80, 0x22, 0xcd, 0x58, 0xd1, 0x9f, 0x62, 0x53, 0xd3, 0x96, 0x0d, 0x56, 0xdd,
	0xdf, 0x8c, 0x8b, 0x61, 0xd5, 0x76, 0x9d, 0x2c, 0xd8, 0xee, 0xf1, 0xe5, 0xd7, 0xf3, 0x59, 0x68,
	0xe6, 0xc9, 0xdd, 0xc8, 0x17, 0x8b, 0xf1, 0xd4, 0x0b, 0x82, 0x88, 0xf2, 0xdf, 0xe2, 0x70, 0x7a,
	0xf3, 0x65, 0x1c, 0x78, 0xe1, 0x38, 0x7b, 0x0f, 0xf5, 0xf8, 0xcf, 0xaf, 0xeb, 0x5d, 0x33, 0xb3,
	0xbc, 0xfb, 0x1d, 0x00, 0x00, 0xff, 0xff, 0xba, 0xaa, 0x31, 0x69, 0x82, 0x05, 0x00, 0x00,
}