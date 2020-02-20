// Code generated by protoc-gen-go. DO NOT EDIT.
// source: contact.proto

package protobuf

import (
	fmt "fmt"
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

type ContactUpdate struct {
	Clock                uint64   `protobuf:"varint,1,opt,name=clock,proto3" json:"clock,omitempty"`
	EnsName              string   `protobuf:"bytes,2,opt,name=ens_name,json=ensName,proto3" json:"ens_name,omitempty"`
	ProfileImage         string   `protobuf:"bytes,3,opt,name=profile_image,json=profileImage,proto3" json:"profile_image,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ContactUpdate) Reset()         { *m = ContactUpdate{} }
func (m *ContactUpdate) String() string { return proto.CompactTextString(m) }
func (*ContactUpdate) ProtoMessage()    {}
func (*ContactUpdate) Descriptor() ([]byte, []int) {
	return fileDescriptor_a5036fff2565fb15, []int{0}
}

func (m *ContactUpdate) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ContactUpdate.Unmarshal(m, b)
}
func (m *ContactUpdate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ContactUpdate.Marshal(b, m, deterministic)
}
func (m *ContactUpdate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ContactUpdate.Merge(m, src)
}
func (m *ContactUpdate) XXX_Size() int {
	return xxx_messageInfo_ContactUpdate.Size(m)
}
func (m *ContactUpdate) XXX_DiscardUnknown() {
	xxx_messageInfo_ContactUpdate.DiscardUnknown(m)
}

var xxx_messageInfo_ContactUpdate proto.InternalMessageInfo

func (m *ContactUpdate) GetClock() uint64 {
	if m != nil {
		return m.Clock
	}
	return 0
}

func (m *ContactUpdate) GetEnsName() string {
	if m != nil {
		return m.EnsName
	}
	return ""
}

func (m *ContactUpdate) GetProfileImage() string {
	if m != nil {
		return m.ProfileImage
	}
	return ""
}

func init() {
	proto.RegisterType((*ContactUpdate)(nil), "protobuf.ContactUpdate")
}

func init() { proto.RegisterFile("contact.proto", fileDescriptor_a5036fff2565fb15) }

var fileDescriptor_a5036fff2565fb15 = []byte{
	// 135 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x4d, 0xce, 0xcf, 0x2b,
	0x49, 0x4c, 0x2e, 0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x00, 0x53, 0x49, 0xa5, 0x69,
	0x4a, 0xa9, 0x5c, 0xbc, 0xce, 0x10, 0xa9, 0xd0, 0x82, 0x94, 0xc4, 0x92, 0x54, 0x21, 0x11, 0x2e,
	0xd6, 0xe4, 0x9c, 0xfc, 0xe4, 0x6c, 0x09, 0x46, 0x05, 0x46, 0x0d, 0x96, 0x20, 0x08, 0x47, 0x48,
	0x92, 0x8b, 0x23, 0x35, 0xaf, 0x38, 0x3e, 0x2f, 0x31, 0x37, 0x55, 0x82, 0x49, 0x81, 0x51, 0x83,
	0x33, 0x88, 0x3d, 0x35, 0xaf, 0xd8, 0x2f, 0x31, 0x37, 0x55, 0x48, 0x99, 0x8b, 0xb7, 0xa0, 0x28,
	0x3f, 0x2d, 0x33, 0x27, 0x35, 0x3e, 0x33, 0x37, 0x31, 0x3d, 0x55, 0x82, 0x19, 0x2c, 0xcf, 0x03,
	0x15, 0xf4, 0x04, 0x89, 0x25, 0xb1, 0x81, 0x2d, 0x34, 0x06, 0x04, 0x00, 0x00, 0xff, 0xff, 0x74,
	0xf9, 0x5c, 0xff, 0x88, 0x00, 0x00, 0x00,
}
