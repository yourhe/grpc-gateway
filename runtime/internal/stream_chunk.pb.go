// Code generated by protoc-gen-go. DO NOT EDIT.
// source: runtime/internal/stream_chunk.proto

package internal

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// StreamError is a response type which is returned when
// streaming rpc returns an error.
type StreamError struct {
	GrpcCode             int32    `protobuf:"varint,1,opt,name=grpc_code,json=grpcCode,proto3" json:"grpc_code,omitempty"`
	HttpCode             int32    `protobuf:"varint,2,opt,name=http_code,json=httpCode,proto3" json:"http_code,omitempty"`
	Message              string   `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
	HttpStatus           string   `protobuf:"bytes,4,opt,name=http_status,json=httpStatus,proto3" json:"http_status,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StreamError) Reset()         { *m = StreamError{} }
func (m *StreamError) String() string { return proto.CompactTextString(m) }
func (*StreamError) ProtoMessage()    {}
func (*StreamError) Descriptor() ([]byte, []int) {
	return fileDescriptor_stream_chunk_4a21d364092bb7fc, []int{0}
}
func (m *StreamError) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StreamError.Unmarshal(m, b)
}
func (m *StreamError) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StreamError.Marshal(b, m, deterministic)
}
func (dst *StreamError) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StreamError.Merge(dst, src)
}
func (m *StreamError) XXX_Size() int {
	return xxx_messageInfo_StreamError.Size(m)
}
func (m *StreamError) XXX_DiscardUnknown() {
	xxx_messageInfo_StreamError.DiscardUnknown(m)
}

var xxx_messageInfo_StreamError proto.InternalMessageInfo

func (m *StreamError) GetGrpcCode() int32 {
	if m != nil {
		return m.GrpcCode
	}
	return 0
}

func (m *StreamError) GetHttpCode() int32 {
	if m != nil {
		return m.HttpCode
	}
	return 0
}

func (m *StreamError) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func (m *StreamError) GetHttpStatus() string {
	if m != nil {
		return m.HttpStatus
	}
	return ""
}

func init() {
	proto.RegisterType((*StreamError)(nil), "grpc.gateway.runtime.StreamError")
}

func init() {
	proto.RegisterFile("runtime/internal/stream_chunk.proto", fileDescriptor_stream_chunk_4a21d364092bb7fc)
}

var fileDescriptor_stream_chunk_4a21d364092bb7fc = []byte{
	// 181 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x34, 0x8e, 0xbf, 0xee, 0x82, 0x30,
	0x14, 0x85, 0xd3, 0xdf, 0x1f, 0x85, 0xcb, 0x46, 0x1c, 0x9a, 0x38, 0x48, 0x74, 0x61, 0x82, 0xc1,
	0x37, 0xd0, 0xf8, 0x02, 0xb0, 0xb9, 0x90, 0x0a, 0x37, 0x40, 0x94, 0x96, 0xdc, 0x5e, 0x62, 0x5c,
	0x7d, 0x72, 0xd3, 0x22, 0xe3, 0xf9, 0xbe, 0x73, 0x92, 0x03, 0x07, 0x9a, 0x34, 0xf7, 0x03, 0xe6,
	0xbd, 0x66, 0x24, 0xad, 0x1e, 0xb9, 0x65, 0x42, 0x35, 0x54, 0x75, 0x37, 0xe9, 0x7b, 0x36, 0x92,
	0x61, 0x13, 0x6f, 0x5a, 0x1a, 0xeb, 0xac, 0x55, 0x8c, 0x4f, 0xf5, 0xca, 0xbe, 0x8b, 0xfd, 0x5b,
	0x40, 0x54, 0xfa, 0xf2, 0x85, 0xc8, 0x50, 0xbc, 0x85, 0xd0, 0xf5, 0xaa, 0xda, 0x34, 0x28, 0x45,
	0x22, 0xd2, 0xff, 0x22, 0x70, 0xe0, 0x6c, 0x1a, 0x74, 0xb2, 0x63, 0x1e, 0x67, 0xf9, 0x33, 0x4b,
	0x07, 0xbc, 0x94, 0xb0, 0x1e, 0xd0, 0x5a, 0xd5, 0xa2, 0xfc, 0x4d, 0x44, 0x1a, 0x16, 0x4b, 0x8c,
	0x77, 0x10, 0xf9, 0x99, 0x65, 0xc5, 0x93, 0x95, 0x7f, 0xde, 0x82, 0x43, 0xa5, 0x27, 0x27, 0xb8,
	0x06, 0xcb, 0xf3, 0xdb, 0xca, 0xbf, 0x3d, 0x7e, 0x02, 0x00, 0x00, 0xff, 0xff, 0xa9, 0x07, 0x92,
	0xb6, 0xd4, 0x00, 0x00, 0x00,
}
