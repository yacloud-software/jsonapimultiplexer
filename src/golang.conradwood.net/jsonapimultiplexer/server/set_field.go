package main

import (
	"fmt"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"strconv"
)

// take an input string and try to convert it to the format we need it in
// nice thing here is: We can generate helpful, specific messages, such
// as "The value X you passed in as parameter NAME must be of type UINT32"
func setField(m *dynamic.Message, fd *desc.FieldDescriptor, value string) error {
	switch fd.GetType() {
	case dpb.FieldDescriptorProto_TYPE_INT32:
		x, err := strconv.Atoi(value)
		return setFieldWithError(err, m, fd, x, value)
	case dpb.FieldDescriptorProto_TYPE_INT64:
		x, err := strconv.ParseInt(value, 10, 64)
		return setFieldWithError(err, m, fd, x, value)
	case dpb.FieldDescriptorProto_TYPE_UINT64:
		x, err := strconv.ParseUint(value, 10, 64)
		return setFieldWithError(err, m, fd, x, value)
	case dpb.FieldDescriptorProto_TYPE_UINT32:
		x, err := strconv.ParseUint(value, 10, 32)
		return setFieldWithError(err, m, fd, uint32(x), value)
	case dpb.FieldDescriptorProto_TYPE_FLOAT:
		x, err := strconv.ParseFloat(value, 64)
		return setFieldWithError(err, m, fd, float64(x), value)
	case dpb.FieldDescriptorProto_TYPE_STRING:
		return m.TrySetField(fd, value)
	case dpb.FieldDescriptorProto_TYPE_BOOL:
		x, err := strconv.ParseBool(value)
		return setFieldWithError(err, m, fd, x, value)
	case dpb.FieldDescriptorProto_TYPE_ENUM:
		return m.TrySetField(fd, value)
	default:
		return fmt.Errorf("Unable to set field %s (type %s) in %s to %s (Unsupported conversion)", fd.GetName(), fd.GetType(), m.XXX_MessageName(), value)
	}
}

// generate the helpful and consistent error message
func setFieldWithError(err error, m *dynamic.Message, fd *desc.FieldDescriptor, value interface{}, vs string) error {
	if err != nil {
		return fmt.Errorf("Field %s is of type %s. Your value \"%s\" does not convert to that format (%s)", fd.GetName(), fd.GetType(), vs, err)
	}
	return m.TrySetField(fd, value)
}
