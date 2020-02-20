// Code generated by protoc-gen-go. DO NOT EDIT.
// source: chat_message.proto

package protobuf

import (
	fmt "fmt"
	math "math"

	proto "github.com/golang/protobuf/proto"
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

type ChatMessage_MessageType int32

const (
	ChatMessage_UNKNOWN_MESSAGE_TYPE ChatMessage_MessageType = 0
	ChatMessage_ONE_TO_ONE           ChatMessage_MessageType = 1
	ChatMessage_PUBLIC_GROUP         ChatMessage_MessageType = 2
	ChatMessage_PRIVATE_GROUP        ChatMessage_MessageType = 3
	// Only local
	ChatMessage_SYSTEM_MESSAGE_PRIVATE_GROUP ChatMessage_MessageType = 4
)

var ChatMessage_MessageType_name = map[int32]string{
	0: "UNKNOWN_MESSAGE_TYPE",
	1: "ONE_TO_ONE",
	2: "PUBLIC_GROUP",
	3: "PRIVATE_GROUP",
	4: "SYSTEM_MESSAGE_PRIVATE_GROUP",
}

var ChatMessage_MessageType_value = map[string]int32{
	"UNKNOWN_MESSAGE_TYPE":         0,
	"ONE_TO_ONE":                   1,
	"PUBLIC_GROUP":                 2,
	"PRIVATE_GROUP":                3,
	"SYSTEM_MESSAGE_PRIVATE_GROUP": 4,
}

func (x ChatMessage_MessageType) String() string {
	return proto.EnumName(ChatMessage_MessageType_name, int32(x))
}

func (ChatMessage_MessageType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_263952f55fd35689, []int{1, 0}
}

type ChatMessage_ContentType int32

const (
	ChatMessage_UNKNOWN_CONTENT_TYPE ChatMessage_ContentType = 0
	ChatMessage_TEXT_PLAIN           ChatMessage_ContentType = 1
	ChatMessage_STICKER              ChatMessage_ContentType = 2
	ChatMessage_STATUS               ChatMessage_ContentType = 3
	ChatMessage_EMOJI                ChatMessage_ContentType = 4
	ChatMessage_TRANSACTION_COMMAND  ChatMessage_ContentType = 5
)

var ChatMessage_ContentType_name = map[int32]string{
	0: "UNKNOWN_CONTENT_TYPE",
	1: "TEXT_PLAIN",
	2: "STICKER",
	3: "STATUS",
	4: "EMOJI",
	5: "TRANSACTION_COMMAND",
}

var ChatMessage_ContentType_value = map[string]int32{
	"UNKNOWN_CONTENT_TYPE": 0,
	"TEXT_PLAIN":           1,
	"STICKER":              2,
	"STATUS":               3,
	"EMOJI":                4,
	"TRANSACTION_COMMAND":  5,
}

func (x ChatMessage_ContentType) String() string {
	return proto.EnumName(ChatMessage_ContentType_name, int32(x))
}

func (ChatMessage_ContentType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_263952f55fd35689, []int{1, 1}
}

type StickerMessage struct {
	Hash                 string   `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	Pack                 int32    `protobuf:"varint,2,opt,name=pack,proto3" json:"pack,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StickerMessage) Reset()         { *m = StickerMessage{} }
func (m *StickerMessage) String() string { return proto.CompactTextString(m) }
func (*StickerMessage) ProtoMessage()    {}
func (*StickerMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_263952f55fd35689, []int{0}
}

func (m *StickerMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StickerMessage.Unmarshal(m, b)
}
func (m *StickerMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StickerMessage.Marshal(b, m, deterministic)
}
func (m *StickerMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StickerMessage.Merge(m, src)
}
func (m *StickerMessage) XXX_Size() int {
	return xxx_messageInfo_StickerMessage.Size(m)
}
func (m *StickerMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_StickerMessage.DiscardUnknown(m)
}

var xxx_messageInfo_StickerMessage proto.InternalMessageInfo

func (m *StickerMessage) GetHash() string {
	if m != nil {
		return m.Hash
	}
	return ""
}

func (m *StickerMessage) GetPack() int32 {
	if m != nil {
		return m.Pack
	}
	return 0
}

type ChatMessage struct {
	// Lamport timestamp of the chat message
	Clock uint64 `protobuf:"varint,1,opt,name=clock,proto3" json:"clock,omitempty"`
	// Unix timestamps in milliseconds, currently not used as we use whisper as more reliable, but here
	// so that we don't rely on it
	Timestamp uint64 `protobuf:"varint,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	// Text of the message
	Text string `protobuf:"bytes,3,opt,name=text,proto3" json:"text,omitempty"`
	// ID of the message that we are replying to
	ResponseTo string `protobuf:"bytes,4,opt,name=response_to,json=responseTo,proto3" json:"response_to,omitempty"`
	// Ens name of the sender
	EnsName string `protobuf:"bytes,5,opt,name=ens_name,json=ensName,proto3" json:"ens_name,omitempty"`
	// Chat id, this field is symmetric for public-chats and private group chats,
	// but asymmetric in case of one-to-ones, as the sender will use the chat-id
	// of the received, while the receiver will use the chat-id of the sender.
	// Probably should be the concatenation of sender-pk & receiver-pk in alphabetical order
	ChatId string `protobuf:"bytes,6,opt,name=chat_id,json=chatId,proto3" json:"chat_id,omitempty"`
	// The type of message (public/one-to-one/private-group-chat)
	MessageType ChatMessage_MessageType `protobuf:"varint,7,opt,name=message_type,json=messageType,proto3,enum=protobuf.ChatMessage_MessageType" json:"message_type,omitempty"`
	// The type of the content of the message
	ContentType ChatMessage_ContentType `protobuf:"varint,8,opt,name=content_type,json=contentType,proto3,enum=protobuf.ChatMessage_ContentType" json:"content_type,omitempty"`
	// Types that are valid to be assigned to Payload:
	//	*ChatMessage_Sticker
	Payload              isChatMessage_Payload `protobuf_oneof:"payload"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *ChatMessage) Reset()         { *m = ChatMessage{} }
func (m *ChatMessage) String() string { return proto.CompactTextString(m) }
func (*ChatMessage) ProtoMessage()    {}
func (*ChatMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_263952f55fd35689, []int{1}
}

func (m *ChatMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ChatMessage.Unmarshal(m, b)
}
func (m *ChatMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ChatMessage.Marshal(b, m, deterministic)
}
func (m *ChatMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ChatMessage.Merge(m, src)
}
func (m *ChatMessage) XXX_Size() int {
	return xxx_messageInfo_ChatMessage.Size(m)
}
func (m *ChatMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_ChatMessage.DiscardUnknown(m)
}

var xxx_messageInfo_ChatMessage proto.InternalMessageInfo

func (m *ChatMessage) GetClock() uint64 {
	if m != nil {
		return m.Clock
	}
	return 0
}

func (m *ChatMessage) GetTimestamp() uint64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *ChatMessage) GetText() string {
	if m != nil {
		return m.Text
	}
	return ""
}

func (m *ChatMessage) GetResponseTo() string {
	if m != nil {
		return m.ResponseTo
	}
	return ""
}

func (m *ChatMessage) GetEnsName() string {
	if m != nil {
		return m.EnsName
	}
	return ""
}

func (m *ChatMessage) GetChatId() string {
	if m != nil {
		return m.ChatId
	}
	return ""
}

func (m *ChatMessage) GetMessageType() ChatMessage_MessageType {
	if m != nil {
		return m.MessageType
	}
	return ChatMessage_UNKNOWN_MESSAGE_TYPE
}

func (m *ChatMessage) GetContentType() ChatMessage_ContentType {
	if m != nil {
		return m.ContentType
	}
	return ChatMessage_UNKNOWN_CONTENT_TYPE
}

type isChatMessage_Payload interface {
	isChatMessage_Payload()
}

type ChatMessage_Sticker struct {
	Sticker *StickerMessage `protobuf:"bytes,9,opt,name=sticker,proto3,oneof"`
}

func (*ChatMessage_Sticker) isChatMessage_Payload() {}

func (m *ChatMessage) GetPayload() isChatMessage_Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *ChatMessage) GetSticker() *StickerMessage {
	if x, ok := m.GetPayload().(*ChatMessage_Sticker); ok {
		return x.Sticker
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*ChatMessage) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*ChatMessage_Sticker)(nil),
	}
}

func init() {
	proto.RegisterEnum("protobuf.ChatMessage_MessageType", ChatMessage_MessageType_name, ChatMessage_MessageType_value)
	proto.RegisterEnum("protobuf.ChatMessage_ContentType", ChatMessage_ContentType_name, ChatMessage_ContentType_value)
	proto.RegisterType((*StickerMessage)(nil), "protobuf.StickerMessage")
	proto.RegisterType((*ChatMessage)(nil), "protobuf.ChatMessage")
}

func init() { proto.RegisterFile("chat_message.proto", fileDescriptor_263952f55fd35689) }

var fileDescriptor_263952f55fd35689 = []byte{
	// 460 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x91, 0x4d, 0x6f, 0xd3, 0x4c,
	0x10, 0xc7, 0xeb, 0xc4, 0x89, 0x93, 0x71, 0x9f, 0xc8, 0xcf, 0x52, 0xa9, 0x8b, 0x54, 0x89, 0x90,
	0x53, 0x4e, 0x39, 0x14, 0x0e, 0x5c, 0x5d, 0x77, 0x55, 0x4c, 0xeb, 0xb5, 0xb5, 0xde, 0x00, 0x3d,
	0xad, 0xb6, 0xce, 0x42, 0xa2, 0xd4, 0x2f, 0x8a, 0x17, 0x89, 0x5c, 0xf8, 0xc6, 0x7c, 0x07, 0xe4,
	0x75, 0x42, 0xdc, 0x0b, 0xa7, 0x9d, 0x97, 0xff, 0xfc, 0xc6, 0xf3, 0x37, 0xa0, 0x6c, 0x2d, 0xb5,
	0xc8, 0x55, 0x5d, 0xcb, 0xef, 0x6a, 0x51, 0xed, 0x4a, 0x5d, 0xa2, 0x91, 0x79, 0x9e, 0x7e, 0x7c,
	0x9b, 0x7d, 0x80, 0x49, 0xaa, 0x37, 0xd9, 0x56, 0xed, 0xa2, 0x56, 0x81, 0x10, 0xd8, 0x6b, 0x59,
	0xaf, 0xb1, 0x35, 0xb5, 0xe6, 0x63, 0x66, 0xe2, 0xa6, 0x56, 0xc9, 0x6c, 0x8b, 0x7b, 0x53, 0x6b,
	0x3e, 0x60, 0x26, 0x9e, 0xfd, 0xb6, 0xc1, 0x0d, 0xd6, 0x52, 0x1f, 0xe7, 0x2e, 0x60, 0x90, 0x3d,
	0x97, 0xd9, 0xd6, 0x0c, 0xda, 0xac, 0x4d, 0xd0, 0x15, 0x8c, 0xf5, 0x26, 0x57, 0xb5, 0x96, 0x79,
	0x65, 0xc6, 0x6d, 0x76, 0x2a, 0x34, 0x5c, 0xad, 0x7e, 0x6a, 0xdc, 0x6f, 0x77, 0x35, 0x31, 0x7a,
	0x03, 0xee, 0x4e, 0xd5, 0x55, 0x59, 0xd4, 0x4a, 0xe8, 0x12, 0xdb, 0xa6, 0x05, 0xc7, 0x12, 0x2f,
	0xd1, 0x6b, 0x18, 0xa9, 0xa2, 0x16, 0x85, 0xcc, 0x15, 0x1e, 0x98, 0xae, 0xa3, 0x8a, 0x9a, 0xca,
	0x5c, 0xa1, 0x4b, 0x70, 0xcc, 0xb5, 0x9b, 0x15, 0x1e, 0x9a, 0xce, 0xb0, 0x49, 0xc3, 0x15, 0xba,
	0x85, 0xf3, 0x83, 0x03, 0x42, 0xef, 0x2b, 0x85, 0x9d, 0xa9, 0x35, 0x9f, 0x5c, 0xbf, 0x5d, 0x1c,
	0x7d, 0x58, 0x74, 0x2e, 0x59, 0x1c, 0x5e, 0xbe, 0xaf, 0x14, 0x73, 0xf3, 0x53, 0xd2, 0x50, 0xb2,
	0xb2, 0xd0, 0xaa, 0xd0, 0x2d, 0x65, 0xf4, 0x2f, 0x4a, 0xd0, 0x2a, 0x5b, 0x4a, 0x76, 0x4a, 0xd0,
	0x7b, 0x70, 0xea, 0xd6, 0x72, 0x3c, 0x9e, 0x5a, 0x73, 0xf7, 0x1a, 0x9f, 0x00, 0x2f, 0xff, 0xc5,
	0xc7, 0x33, 0x76, 0x94, 0xce, 0x7e, 0x81, 0xdb, 0xf9, 0x2e, 0x84, 0xe1, 0x62, 0x49, 0xef, 0x69,
	0xfc, 0x85, 0x8a, 0x88, 0xa4, 0xa9, 0x7f, 0x47, 0x04, 0x7f, 0x4c, 0x88, 0x77, 0x86, 0x26, 0x00,
	0x31, 0x25, 0x82, 0xc7, 0x22, 0xa6, 0xc4, 0xb3, 0x90, 0x07, 0xe7, 0xc9, 0xf2, 0xe6, 0x21, 0x0c,
	0xc4, 0x1d, 0x8b, 0x97, 0x89, 0xd7, 0x43, 0xff, 0xc3, 0x7f, 0x09, 0x0b, 0x3f, 0xfb, 0x9c, 0x1c,
	0x4a, 0x7d, 0x34, 0x85, 0xab, 0xf4, 0x31, 0xe5, 0x24, 0xfa, 0x4b, 0x7b, 0xa9, 0xb0, 0x67, 0x1a,
	0xdc, 0xce, 0x45, 0xdd, 0xfd, 0x41, 0x4c, 0x39, 0xa1, 0xbc, 0xb3, 0x9f, 0x93, 0xaf, 0x5c, 0x24,
	0x0f, 0x7e, 0x48, 0x3d, 0x0b, 0xb9, 0xe0, 0xa4, 0x3c, 0x0c, 0xee, 0x09, 0xf3, 0x7a, 0x08, 0x60,
	0x98, 0x72, 0x9f, 0x2f, 0x53, 0xaf, 0x8f, 0xc6, 0x30, 0x20, 0x51, 0xfc, 0x29, 0xf4, 0x6c, 0x74,
	0x09, 0xaf, 0x38, 0xf3, 0x69, 0xea, 0x07, 0x3c, 0x8c, 0x1b, 0x62, 0x14, 0xf9, 0xf4, 0xd6, 0x1b,
	0xdc, 0x8c, 0xc1, 0xa9, 0xe4, 0xfe, 0xb9, 0x94, 0xab, 0xa7, 0xa1, 0x31, 0xe9, 0xdd, 0x9f, 0x00,
	0x00, 0x00, 0xff, 0xff, 0x54, 0x6d, 0xd5, 0x10, 0xd0, 0x02, 0x00, 0x00,
}
