// Code generated by protoc-gen-go. DO NOT EDIT.
// source: mpc/learners/dnn_paddlefl_vl/dnn_paddlefl_vl.proto

package dnn_paddlefl_vl // import "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc/learners/dnn_paddlefl_vl"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import mpc "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// MessageType defines the type of message with which communicate with nodes in cluster,
// and in some way it indicates the phase of learning
// Some types are for local message which is not passed between nodes
type MessageType int32

const (
	MessageType_MsgPsiEnc                MessageType = 0
	MessageType_MsgPsiAskReEnc           MessageType = 1
	MessageType_MsgPsiReEnc              MessageType = 2
	MessageType_MsgPsiIntersect          MessageType = 3
	MessageType_MsgFLENVPrepare          MessageType = 4
	MessageType_MsgFLDataPrepare         MessageType = 5
	MessageType_MsgFLDataGenerate        MessageType = 6
	MessageType_MsgFLDataSend            MessageType = 7
	MessageType_MsgFLDataExchange        MessageType = 8
	MessageType_MsgFLDataStatus          MessageType = 9
	MessageType_MsgTrain                 MessageType = 10
	MessageType_MsgPredictHup            MessageType = 17
	MessageType_MsgPredictResultSend     MessageType = 18
	MessageType_MsgPredictResultExchange MessageType = 19
	MessageType_MsgPredictResultStatus   MessageType = 20
	MessageType_MsgPredictResultRecovery MessageType = 21
	MessageType_MsgPredictStop           MessageType = 22
)

var MessageType_name = map[int32]string{
	0:  "MsgPsiEnc",
	1:  "MsgPsiAskReEnc",
	2:  "MsgPsiReEnc",
	3:  "MsgPsiIntersect",
	4:  "MsgFLENVPrepare",
	5:  "MsgFLDataPrepare",
	6:  "MsgFLDataGenerate",
	7:  "MsgFLDataSend",
	8:  "MsgFLDataExchange",
	9:  "MsgFLDataStatus",
	10: "MsgTrain",
	17: "MsgPredictHup",
	18: "MsgPredictResultSend",
	19: "MsgPredictResultExchange",
	20: "MsgPredictResultStatus",
	21: "MsgPredictResultRecovery",
	22: "MsgPredictStop",
}
var MessageType_value = map[string]int32{
	"MsgPsiEnc":                0,
	"MsgPsiAskReEnc":           1,
	"MsgPsiReEnc":              2,
	"MsgPsiIntersect":          3,
	"MsgFLENVPrepare":          4,
	"MsgFLDataPrepare":         5,
	"MsgFLDataGenerate":        6,
	"MsgFLDataSend":            7,
	"MsgFLDataExchange":        8,
	"MsgFLDataStatus":          9,
	"MsgTrain":                 10,
	"MsgPredictHup":            17,
	"MsgPredictResultSend":     18,
	"MsgPredictResultExchange": 19,
	"MsgPredictResultStatus":   20,
	"MsgPredictResultRecovery": 21,
	"MsgPredictStop":           22,
}

func (x MessageType) String() string {
	return proto.EnumName(MessageType_name, int32(x))
}
func (MessageType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_dnn_paddlefl_vl_43444e651b05323f, []int{0}
}

type Message struct {
	Type                 MessageType                `protobuf:"varint,1,opt,name=type,enum=dnn_paddlefl_vl.MessageType" json:"type,omitempty"`
	To                   string                     `protobuf:"bytes,2,opt,name=to" json:"to,omitempty"`
	From                 string                     `protobuf:"bytes,3,opt,name=from" json:"from,omitempty"`
	LoopRound            uint64                     `protobuf:"varint,4,opt,name=loopRound" json:"loopRound,omitempty"`
	VlLPsiReEncIDsReq    *mpc.VLPsiReEncIDsRequest  `protobuf:"bytes,5,opt,name=vlLPsiReEncIDsReq" json:"vlLPsiReEncIDsReq,omitempty"`
	VlLPsiReEncIDsResp   *mpc.VLPsiReEncIDsResponse `protobuf:"bytes,6,opt,name=vlLPsiReEncIDsResp" json:"vlLPsiReEncIDsResp,omitempty"`
	HomoPubkey           []byte                     `protobuf:"bytes,7,opt,name=homoPubkey,proto3" json:"homoPubkey,omitempty"`
	PartBytes            []byte                     `protobuf:"bytes,8,opt,name=PartBytes,proto3" json:"PartBytes,omitempty"`
	EncGradFromOther     []byte                     `protobuf:"bytes,9,opt,name=encGradFromOther,proto3" json:"encGradFromOther,omitempty"`
	EncCostFromOther     []byte                     `protobuf:"bytes,10,opt,name=encCostFromOther,proto3" json:"encCostFromOther,omitempty"`
	GradBytes            []byte                     `protobuf:"bytes,11,opt,name=gradBytes,proto3" json:"gradBytes,omitempty"`
	CostBytes            []byte                     `protobuf:"bytes,12,opt,name=costBytes,proto3" json:"costBytes,omitempty"`
	Stopped              bool                       `protobuf:"varint,13,opt,name=stopped" json:"stopped,omitempty"`
	Aby3ShareData        []byte                     `protobuf:"bytes,14,opt,name=aby3ShareData,proto3" json:"aby3ShareData,omitempty"`
	Aby3ShareFile        []byte                     `protobuf:"bytes,15,opt,name=aby3ShareFile,proto3" json:"aby3ShareFile,omitempty"`
	VecSize              uint64                     `protobuf:"varint,16,opt,name=vecSize" json:"vecSize,omitempty"`
	Role                 uint64                     `protobuf:"varint,17,opt,name=role" json:"role,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                   `json:"-"`
	XXX_unrecognized     []byte                     `json:"-"`
	XXX_sizecache        int32                      `json:"-"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto.CompactTextString(m) }
func (*Message) ProtoMessage()    {}
func (*Message) Descriptor() ([]byte, []int) {
	return fileDescriptor_dnn_paddlefl_vl_43444e651b05323f, []int{0}
}
func (m *Message) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Message.Unmarshal(m, b)
}
func (m *Message) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Message.Marshal(b, m, deterministic)
}
func (dst *Message) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message.Merge(dst, src)
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

func (m *Message) GetAby3ShareData() []byte {
	if m != nil {
		return m.Aby3ShareData
	}
	return nil
}

func (m *Message) GetAby3ShareFile() []byte {
	if m != nil {
		return m.Aby3ShareFile
	}
	return nil
}

func (m *Message) GetVecSize() uint64 {
	if m != nil {
		return m.VecSize
	}
	return 0
}

func (m *Message) GetRole() uint64 {
	if m != nil {
		return m.Role
	}
	return 0
}

type PredictMessage struct {
	Type                 MessageType                `protobuf:"varint,1,opt,name=type,enum=dnn_paddlefl_vl.MessageType" json:"type,omitempty"`
	To                   string                     `protobuf:"bytes,2,opt,name=to" json:"to,omitempty"`
	From                 string                     `protobuf:"bytes,3,opt,name=from" json:"from,omitempty"`
	VlLPsiReEncIDsReq    *mpc.VLPsiReEncIDsRequest  `protobuf:"bytes,4,opt,name=vlLPsiReEncIDsReq" json:"vlLPsiReEncIDsReq,omitempty"`
	VlLPsiReEncIDsResp   *mpc.VLPsiReEncIDsResponse `protobuf:"bytes,5,opt,name=vlLPsiReEncIDsResp" json:"vlLPsiReEncIDsResp,omitempty"`
	PredictPart          []float64                  `protobuf:"fixed64,6,rep,packed,name=predictPart" json:"predictPart,omitempty"`
	Aby3ShareData        []byte                     `protobuf:"bytes,7,opt,name=aby3ShareData,proto3" json:"aby3ShareData,omitempty"`
	Aby3ShareFile        []byte                     `protobuf:"bytes,8,opt,name=aby3ShareFile,proto3" json:"aby3ShareFile,omitempty"`
	VecSize              uint64                     `protobuf:"varint,9,opt,name=vecSize" json:"vecSize,omitempty"`
	Role                 uint64                     `protobuf:"varint,10,opt,name=role" json:"role,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                   `json:"-"`
	XXX_unrecognized     []byte                     `json:"-"`
	XXX_sizecache        int32                      `json:"-"`
}

func (m *PredictMessage) Reset()         { *m = PredictMessage{} }
func (m *PredictMessage) String() string { return proto.CompactTextString(m) }
func (*PredictMessage) ProtoMessage()    {}
func (*PredictMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_dnn_paddlefl_vl_43444e651b05323f, []int{1}
}
func (m *PredictMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PredictMessage.Unmarshal(m, b)
}
func (m *PredictMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PredictMessage.Marshal(b, m, deterministic)
}
func (dst *PredictMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PredictMessage.Merge(dst, src)
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

func (m *PredictMessage) GetAby3ShareData() []byte {
	if m != nil {
		return m.Aby3ShareData
	}
	return nil
}

func (m *PredictMessage) GetAby3ShareFile() []byte {
	if m != nil {
		return m.Aby3ShareFile
	}
	return nil
}

func (m *PredictMessage) GetVecSize() uint64 {
	if m != nil {
		return m.VecSize
	}
	return 0
}

func (m *PredictMessage) GetRole() uint64 {
	if m != nil {
		return m.Role
	}
	return 0
}

func init() {
	proto.RegisterType((*Message)(nil), "dnn_paddlefl_vl.Message")
	proto.RegisterType((*PredictMessage)(nil), "dnn_paddlefl_vl.PredictMessage")
	proto.RegisterEnum("dnn_paddlefl_vl.MessageType", MessageType_name, MessageType_value)
}

func init() {
	proto.RegisterFile("mpc/learners/dnn_paddlefl_vl/dnn_paddlefl_vl.proto", fileDescriptor_dnn_paddlefl_vl_43444e651b05323f)
}

var fileDescriptor_dnn_paddlefl_vl_43444e651b05323f = []byte{
	// 664 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xb4, 0x95, 0xdb, 0x6e, 0xda, 0x4c,
	0x10, 0xc7, 0x3f, 0x03, 0xe1, 0x30, 0x04, 0x58, 0x36, 0x07, 0xed, 0x17, 0x45, 0x95, 0x15, 0xf5,
	0x02, 0xe5, 0x02, 0xaa, 0xe4, 0x09, 0x9a, 0x86, 0xa4, 0x69, 0x93, 0x16, 0x99, 0x28, 0xaa, 0x7a,
	0x13, 0x2d, 0xf6, 0x04, 0xac, 0x18, 0xef, 0x76, 0x77, 0x41, 0xa5, 0x2f, 0xd2, 0x57, 0xed, 0x55,
	0x55, 0x79, 0xcd, 0x21, 0x1c, 0xd4, 0x56, 0x55, 0x7b, 0x83, 0x3c, 0xbf, 0xf9, 0xcf, 0xfc, 0xd9,
	0x9d, 0x01, 0xc3, 0xc9, 0x50, 0xfa, 0xad, 0x08, 0xb9, 0x8a, 0x51, 0xe9, 0x56, 0x10, 0xc7, 0xf7,
	0x92, 0x07, 0x41, 0x84, 0x0f, 0xd1, 0xfd, 0x38, 0x5a, 0x8d, 0x9b, 0x52, 0x09, 0x23, 0x68, 0x6d,
	0x05, 0x1f, 0x54, 0x92, 0x26, 0x52, 0x87, 0x69, 0xfe, 0xe8, 0x5b, 0x0e, 0x0a, 0x37, 0xa8, 0x35,
	0xef, 0x23, 0x7d, 0x01, 0x39, 0x33, 0x91, 0xc8, 0x1c, 0xd7, 0x69, 0x54, 0x4f, 0x0e, 0x9b, 0xab,
	0x1d, 0xa7, 0xba, 0xdb, 0x89, 0x44, 0xcf, 0x2a, 0x69, 0x15, 0x32, 0x46, 0xb0, 0x8c, 0xeb, 0x34,
	0x4a, 0x5e, 0xc6, 0x08, 0x4a, 0x21, 0xf7, 0xa0, 0xc4, 0x90, 0x65, 0x2d, 0xb1, 0xcf, 0xf4, 0x10,
	0x4a, 0x91, 0x10, 0xd2, 0x13, 0xa3, 0x38, 0x60, 0x39, 0xd7, 0x69, 0xe4, 0xbc, 0x05, 0xa0, 0x97,
	0x50, 0x1f, 0x47, 0xd7, 0x1d, 0x1d, 0x7a, 0xd8, 0x8e, 0xfd, 0xab, 0x73, 0xed, 0xe1, 0x27, 0xb6,
	0xe5, 0x3a, 0x8d, 0xf2, 0xc9, 0xff, 0xcd, 0xa1, 0xf4, 0x9b, 0x77, 0x2b, 0xc9, 0x11, 0x6a, 0xe3,
	0xad, 0xd7, 0xd0, 0x37, 0x40, 0x57, 0xa1, 0x96, 0x2c, 0x6f, 0x3b, 0x1d, 0x6c, 0xea, 0xa4, 0xa5,
	0x88, 0x35, 0x7a, 0x1b, 0xaa, 0xe8, 0x33, 0x80, 0x81, 0x18, 0x8a, 0xce, 0xa8, 0xf7, 0x88, 0x13,
	0x56, 0x70, 0x9d, 0xc6, 0xb6, 0xf7, 0x84, 0x24, 0x47, 0xea, 0x70, 0x65, 0xce, 0x26, 0x06, 0x35,
	0x2b, 0xda, 0xf4, 0x02, 0xd0, 0x63, 0x20, 0x18, 0xfb, 0x97, 0x8a, 0x07, 0x17, 0x4a, 0x0c, 0xdf,
	0x9b, 0x01, 0x2a, 0x56, 0xb2, 0xa2, 0x35, 0x3e, 0xd5, 0xbe, 0x12, 0xda, 0x2c, 0xb4, 0x30, 0xd7,
	0x2e, 0xf1, 0xc4, 0xb5, 0xaf, 0x78, 0x90, 0xba, 0x96, 0x53, 0xd7, 0x39, 0x48, 0xb2, 0xbe, 0xd0,
	0xd3, 0xef, 0xb4, 0x9d, 0x66, 0xe7, 0x80, 0x32, 0x28, 0x68, 0x23, 0xa4, 0xc4, 0x80, 0x55, 0x5c,
	0xa7, 0x51, 0xf4, 0x66, 0x21, 0x7d, 0x0e, 0x15, 0xde, 0x9b, 0x9c, 0x76, 0x07, 0x5c, 0xe1, 0x39,
	0x37, 0x9c, 0x55, 0x6d, 0xed, 0x32, 0x5c, 0x52, 0x5d, 0x84, 0x11, 0xb2, 0xda, 0x8a, 0x2a, 0x81,
	0x89, 0xcb, 0x18, 0xfd, 0x6e, 0xf8, 0x05, 0x19, 0xb1, 0x83, 0x9e, 0x85, 0xc9, 0x62, 0x28, 0x11,
	0x21, 0xab, 0x5b, 0x6c, 0x9f, 0x8f, 0xbe, 0x66, 0xa1, 0xda, 0x51, 0x18, 0x84, 0xbe, 0xf9, 0xb7,
	0x1b, 0xb8, 0x71, 0xc7, 0x72, 0x7f, 0x6d, 0xc7, 0xb6, 0xfe, 0x68, 0xc7, 0x5c, 0x28, 0xcb, 0xf4,
	0xf0, 0xc9, 0xe6, 0xb0, 0xbc, 0x9b, 0x6d, 0x38, 0xde, 0x53, 0xb4, 0x3e, 0x99, 0xc2, 0x6f, 0x4d,
	0xa6, 0xf8, 0x8b, 0xc9, 0x94, 0x36, 0x4f, 0x06, 0x16, 0x93, 0x39, 0xfe, 0x9e, 0x81, 0xf2, 0x93,
	0xab, 0xa6, 0x15, 0x28, 0xdd, 0xe8, 0x7e, 0x47, 0x87, 0xed, 0xd8, 0x27, 0xff, 0x51, 0x0a, 0xd5,
	0x34, 0x7c, 0xa9, 0x1f, 0xed, 0x99, 0x88, 0x43, 0x6b, 0x50, 0x4e, 0x59, 0x0a, 0x32, 0x74, 0x07,
	0x6a, 0x29, 0xb8, 0x8a, 0x0d, 0x2a, 0x8d, 0xbe, 0x21, 0xd9, 0x29, 0xbc, 0xb8, 0x6e, 0xbf, 0xbb,
	0xeb, 0x28, 0x94, 0x5c, 0x21, 0xc9, 0xd1, 0x5d, 0x20, 0x16, 0x26, 0xc7, 0x99, 0xd1, 0x2d, 0xba,
	0x07, 0xf5, 0x39, 0xbd, 0xc4, 0x18, 0x15, 0x37, 0x48, 0xf2, 0xb4, 0x0e, 0x95, 0x39, 0xee, 0x62,
	0x1c, 0x90, 0xc2, 0x92, 0xb2, 0xfd, 0xd9, 0x1f, 0xf0, 0xb8, 0x8f, 0xa4, 0x38, 0xf7, 0xb2, 0x4a,
	0xc3, 0xcd, 0x48, 0x93, 0x12, 0xdd, 0x86, 0xe2, 0x8d, 0xee, 0xdf, 0x2a, 0x1e, 0xc6, 0x04, 0xa6,
	0xcd, 0xa6, 0x3b, 0xf8, 0x7a, 0x24, 0x49, 0x9d, 0x32, 0xd8, 0x5d, 0x20, 0x0f, 0xf5, 0x28, 0x32,
	0xd6, 0x86, 0xd2, 0x43, 0x60, 0xab, 0x99, 0xb9, 0xdb, 0x0e, 0x3d, 0x80, 0xfd, 0xb5, 0xba, 0xd4,
	0x74, 0x77, 0x53, 0xa5, 0x87, 0xbe, 0x18, 0xa3, 0x9a, 0x90, 0xbd, 0xd9, 0x6d, 0xa6, 0xd9, 0xae,
	0x11, 0x92, 0xec, 0x9f, 0xbd, 0xfd, 0x78, 0xd5, 0x0f, 0xcd, 0x60, 0xd4, 0x6b, 0xfa, 0x62, 0xd8,
	0xea, 0xd8, 0x5f, 0x40, 0xfa, 0x39, 0x0d, 0xce, 0x6f, 0x3f, 0xb4, 0x02, 0x1e, 0xb6, 0xec, 0x9f,
	0xb8, 0x6e, 0xfd, 0xec, 0xc5, 0xd0, 0xcb, 0x5b, 0xd1, 0xe9, 0x8f, 0x00, 0x00, 0x00, 0xff, 0xff,
	0x3b, 0xb2, 0xd0, 0xc2, 0x3f, 0x06, 0x00, 0x00,
}
