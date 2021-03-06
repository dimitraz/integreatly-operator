// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/ads/googleads/v1/resources/ad_schedule_view.proto

package resources // import "google.golang.org/genproto/googleapis/ads/googleads/v1/resources"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "google.golang.org/genproto/googleapis/api/annotations"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// An ad schedule view summarizes the performance of campaigns by
// AdSchedule criteria.
type AdScheduleView struct {
	// The resource name of the ad schedule view.
	// AdSchedule view resource names have the form:
	//
	// `customers/{customer_id}/adScheduleViews/{campaign_id}~{criterion_id}`
	ResourceName         string   `protobuf:"bytes,1,opt,name=resource_name,json=resourceName,proto3" json:"resource_name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AdScheduleView) Reset()         { *m = AdScheduleView{} }
func (m *AdScheduleView) String() string { return proto.CompactTextString(m) }
func (*AdScheduleView) ProtoMessage()    {}
func (*AdScheduleView) Descriptor() ([]byte, []int) {
	return fileDescriptor_ad_schedule_view_ead0513e8452d09a, []int{0}
}
func (m *AdScheduleView) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AdScheduleView.Unmarshal(m, b)
}
func (m *AdScheduleView) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AdScheduleView.Marshal(b, m, deterministic)
}
func (dst *AdScheduleView) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AdScheduleView.Merge(dst, src)
}
func (m *AdScheduleView) XXX_Size() int {
	return xxx_messageInfo_AdScheduleView.Size(m)
}
func (m *AdScheduleView) XXX_DiscardUnknown() {
	xxx_messageInfo_AdScheduleView.DiscardUnknown(m)
}

var xxx_messageInfo_AdScheduleView proto.InternalMessageInfo

func (m *AdScheduleView) GetResourceName() string {
	if m != nil {
		return m.ResourceName
	}
	return ""
}

func init() {
	proto.RegisterType((*AdScheduleView)(nil), "google.ads.googleads.v1.resources.AdScheduleView")
}

func init() {
	proto.RegisterFile("google/ads/googleads/v1/resources/ad_schedule_view.proto", fileDescriptor_ad_schedule_view_ead0513e8452d09a)
}

var fileDescriptor_ad_schedule_view_ead0513e8452d09a = []byte{
	// 271 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x90, 0xc1, 0x4a, 0xc4, 0x30,
	0x10, 0x86, 0x69, 0x05, 0xc1, 0xa2, 0x1e, 0xd6, 0x8b, 0x88, 0x07, 0x57, 0x59, 0xf0, 0x94, 0x50,
	0x44, 0x90, 0x78, 0xca, 0x5e, 0x16, 0x3c, 0xc8, 0xb2, 0x42, 0x0f, 0x52, 0x28, 0xb1, 0x19, 0x62,
	0xa1, 0xcd, 0x94, 0x4e, 0xb7, 0x7b, 0xf5, 0x59, 0x3c, 0xfa, 0x28, 0x3e, 0x8a, 0x4f, 0x21, 0xdd,
	0x6c, 0x02, 0x5e, 0xdc, 0xdb, 0x4f, 0xf2, 0xfd, 0xdf, 0x0c, 0x93, 0x3c, 0x18, 0x44, 0x53, 0x03,
	0x57, 0x9a, 0xb8, 0x8b, 0x63, 0x1a, 0x52, 0xde, 0x01, 0xe1, 0xba, 0x2b, 0x81, 0xb8, 0xd2, 0x05,
	0x95, 0xef, 0xa0, 0xd7, 0x35, 0x14, 0x43, 0x05, 0x1b, 0xd6, 0x76, 0xd8, 0xe3, 0x64, 0xea, 0x70,
	0xa6, 0x34, 0xb1, 0xd0, 0x64, 0x43, 0xca, 0x42, 0xf3, 0xe2, 0xd2, 0xcb, 0xdb, 0x8a, 0x2b, 0x6b,
	0xb1, 0x57, 0x7d, 0x85, 0x96, 0x9c, 0xe0, 0xfa, 0x3e, 0x39, 0x95, 0xfa, 0x65, 0x67, 0xce, 0x2a,
	0xd8, 0x4c, 0x6e, 0x92, 0x13, 0x5f, 0x2e, 0xac, 0x6a, 0xe0, 0x3c, 0xba, 0x8a, 0x6e, 0x8f, 0x56,
	0xc7, 0xfe, 0xf1, 0x59, 0x35, 0x30, 0xff, 0x88, 0x93, 0x59, 0x89, 0x0d, 0xdb, 0x3b, 0x7e, 0x7e,
	0xf6, 0x57, 0xbf, 0x1c, 0xa7, 0x2e, 0xa3, 0xd7, 0xa7, 0x5d, 0xd3, 0x60, 0xad, 0xac, 0x61, 0xd8,
	0x19, 0x6e, 0xc0, 0x6e, 0x77, 0xf2, 0x27, 0x68, 0x2b, 0xfa, 0xe7, 0x22, 0x8f, 0x21, 0x7d, 0xc6,
	0x07, 0x0b, 0x29, 0xbf, 0xe2, 0xe9, 0xc2, 0x29, 0xa5, 0x26, 0xe6, 0xe2, 0x98, 0xb2, 0x94, 0xad,
	0x3c, 0xf9, 0xed, 0x99, 0x5c, 0x6a, 0xca, 0x03, 0x93, 0x67, 0x69, 0x1e, 0x98, 0x9f, 0x78, 0xe6,
	0x3e, 0x84, 0x90, 0x9a, 0x84, 0x08, 0x94, 0x10, 0x59, 0x2a, 0x44, 0xe0, 0xde, 0x0e, 0xb7, 0xcb,
	0xde, 0xfd, 0x06, 0x00, 0x00, 0xff, 0xff, 0xd4, 0xff, 0xfa, 0xec, 0xbd, 0x01, 0x00, 0x00,
}
