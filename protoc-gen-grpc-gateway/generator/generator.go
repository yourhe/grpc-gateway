// Package generator provides an abstract interface to code generators.
package generator

import (
	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

// Generator is an abstraction of code generators.
type Generator interface {
	// Generate generates output files from input .proto files.
	Generate(targets []*descriptor.FileDescriptorProto) ([]*plugin.CodeGeneratorResponse_File, error)
}
