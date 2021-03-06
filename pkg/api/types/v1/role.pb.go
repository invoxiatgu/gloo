// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: role.proto

package v1

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/gogo/protobuf/types"
import _ "github.com/golang/protobuf/ptypes/duration"
import _ "github.com/gogo/protobuf/gogoproto"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// *
// A Role is a container for a set of Virtual Services that will be used to generate a single proxy config
// to be applied to one or more Envoy nodes. The Role is best understood as an in-mesh application's localized view
// of the rest of the mesh.
// Each domain for each Virtual Service contained in a Role cannot appear more than once, or the Role
// will be invalid.
// In the current implementation, Roles are read-only objects created by Gloo for the puprose of reporting.
// In the future, Gloo will support fields in Roles that can be written to for the purpose of applying policy
// to groups of Virtual Services.
type Role struct {
	// Name of the role. Envoy nodes will be assigned a config matching the role they report to Gloo when registering
	// Envoy instances must specify their role in the prefix for their Node ID when they register to Gloo.
	//
	// Currently this is done in the format <Role>~<this portion is ignored>
	// which can be specified with the `--service-node` flag, or in the Envoy instance's bootstrap config.
	//
	// Role Names must be unique and follow the following syntax rules:
	// One or more lowercase rfc1035/rfc1123 labels separated by '.' with a maximum length of 253 characters.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// a list of virtual services that reference this role
	VirtualServices []string `protobuf:"bytes,2,rep,name=virtual_services,json=virtualServices" json:"virtual_services,omitempty"`
	// Status indicates the validation status of the role resource.
	// Status is read-only by clients, and set by gloo during validation
	Status *Status `protobuf:"bytes,6,opt,name=status" json:"status,omitempty" testdiff:"ignore"`
	// Metadata contains the resource metadata for the role
	Metadata *Metadata `protobuf:"bytes,7,opt,name=metadata" json:"metadata,omitempty"`
}

func (m *Role) Reset()                    { *m = Role{} }
func (m *Role) String() string            { return proto.CompactTextString(m) }
func (*Role) ProtoMessage()               {}
func (*Role) Descriptor() ([]byte, []int) { return fileDescriptorRole, []int{0} }

func (m *Role) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Role) GetVirtualServices() []string {
	if m != nil {
		return m.VirtualServices
	}
	return nil
}

func (m *Role) GetStatus() *Status {
	if m != nil {
		return m.Status
	}
	return nil
}

func (m *Role) GetMetadata() *Metadata {
	if m != nil {
		return m.Metadata
	}
	return nil
}

func init() {
	proto.RegisterType((*Role)(nil), "gloo.api.v1.Role")
}
func (this *Role) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Role)
	if !ok {
		that2, ok := that.(Role)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.Name != that1.Name {
		return false
	}
	if len(this.VirtualServices) != len(that1.VirtualServices) {
		return false
	}
	for i := range this.VirtualServices {
		if this.VirtualServices[i] != that1.VirtualServices[i] {
			return false
		}
	}
	if !this.Status.Equal(that1.Status) {
		return false
	}
	if !this.Metadata.Equal(that1.Metadata) {
		return false
	}
	return true
}

func init() { proto.RegisterFile("role.proto", fileDescriptorRole) }

var fileDescriptorRole = []byte{
	// 283 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x5c, 0x8f, 0xbf, 0x4e, 0xf3, 0x30,
	0x14, 0xc5, 0x95, 0xef, 0xab, 0x0a, 0x75, 0x11, 0x7f, 0x0c, 0x95, 0xa2, 0x0a, 0x95, 0xaa, 0x53,
	0x18, 0xb0, 0x55, 0xd8, 0x18, 0xbb, 0xb3, 0xa4, 0x1b, 0x0b, 0x72, 0x12, 0xc7, 0x58, 0x38, 0xb9,
	0x91, 0x7d, 0x1d, 0x89, 0x37, 0xe2, 0x21, 0x78, 0x16, 0x06, 0x1e, 0x81, 0x27, 0x40, 0x75, 0x0c,
	0x02, 0xb6, 0x73, 0xcf, 0xef, 0xde, 0x63, 0x1f, 0x42, 0x2c, 0x18, 0xc9, 0x3a, 0x0b, 0x08, 0x74,
	0xaa, 0x0c, 0x00, 0x13, 0x9d, 0x66, 0xfd, 0x7a, 0x7e, 0xae, 0x00, 0x94, 0x91, 0x3c, 0xa0, 0xc2,
	0xd7, 0xdc, 0xa1, 0xf5, 0x25, 0x0e, 0xab, 0xf3, 0xc5, 0x5f, 0x5a, 0x79, 0x2b, 0x50, 0x43, 0x1b,
	0xf9, 0x99, 0x02, 0x05, 0x41, 0xf2, 0x9d, 0x8a, 0xee, 0x81, 0x43, 0x81, 0xde, 0xc5, 0xe9, 0xb0,
	0x91, 0x28, 0x2a, 0x81, 0x62, 0x98, 0x57, 0xaf, 0x09, 0x19, 0xe5, 0x60, 0x24, 0xa5, 0x64, 0xd4,
	0x8a, 0x46, 0xa6, 0xc9, 0x32, 0xc9, 0x26, 0x79, 0xd0, 0xf4, 0x92, 0x1c, 0xf7, 0xda, 0xa2, 0x17,
	0xe6, 0xc1, 0x49, 0xdb, 0xeb, 0x52, 0xba, 0xf4, 0xdf, 0xf2, 0x7f, 0x36, 0xc9, 0x8f, 0xa2, 0xbf,
	0x8d, 0x36, 0xdd, 0x90, 0xf1, 0xf0, 0x4e, 0x3a, 0x5e, 0x26, 0xd9, 0xf4, 0xfa, 0x94, 0xfd, 0xe8,
	0xc5, 0xb6, 0x01, 0x6d, 0x66, 0x1f, 0x6f, 0x17, 0x27, 0x28, 0x1d, 0x56, 0xba, 0xae, 0x6f, 0x57,
	0x5a, 0xb5, 0x60, 0xe5, 0x2a, 0x8f, 0x97, 0x74, 0x4d, 0xf6, 0xbf, 0x7e, 0x97, 0xee, 0x85, 0x94,
	0xd9, 0xaf, 0x94, 0xbb, 0x08, 0xf3, 0xef, 0xb5, 0x0d, 0x7b, 0x79, 0x5f, 0x24, 0xf7, 0x99, 0xd2,
	0xf8, 0xe8, 0x0b, 0x56, 0x42, 0xc3, 0x1d, 0x18, 0xb8, 0xd2, 0xc0, 0x77, 0x87, 0xbc, 0x7b, 0x52,
	0x5c, 0x74, 0x9a, 0xe3, 0x73, 0x27, 0x1d, 0xef, 0xd7, 0xc5, 0x38, 0xb4, 0xbe, 0xf9, 0x0c, 0x00,
	0x00, 0xff, 0xff, 0x48, 0x38, 0x38, 0x46, 0x82, 0x01, 0x00, 0x00,
}
