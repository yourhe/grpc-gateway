package gengateway

import (
	"errors"
	"fmt"
	"go/format"
	"path"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/yourhe/grpc-gateway/protoc-gen-plugin/descriptor"
	gen "github.com/yourhe/grpc-gateway/protoc-gen-plugin/generator"
	options "google.golang.org/genproto/googleapis/api/annotations"
)

var (
	errNoTargetService = errors.New("no target service defined in the file")
)

type generator struct {
	reg               *descriptor.Registry
	baseImports       []descriptor.GoPackage
	useRequestContext bool
}

// New returns a new generator which generates grpc gateway files.
func New(reg *descriptor.Registry, useRequestContext bool) gen.Generator {
	var imports []descriptor.GoPackage
	for _, pkgpath := range []string{
		"context",
		"google.golang.org/grpc",
		"github.com/hashicorp/go-plugin",
	} {
		pkg := descriptor.GoPackage{
			Path: pkgpath,
			Name: path.Base(pkgpath),
		}

		if err := reg.ReserveGoPackageAlias(pkg.Name, pkg.Path); err != nil {
			for i := 0; ; i++ {
				alias := fmt.Sprintf("%s_%d", pkg.Name, i)
				if err := reg.ReserveGoPackageAlias(alias, pkg.Path); err != nil {
					continue
				}
				pkg.Alias = alias
				break
			}
		}
		imports = append(imports, pkg)
	}
	return &generator{reg: reg, baseImports: imports, useRequestContext: useRequestContext}
}

func (g *generator) Generate(targets []*descriptor.File) ([]*plugin.CodeGeneratorResponse_File, error) {
	var files []*plugin.CodeGeneratorResponse_File
	for _, file := range targets {
		glog.V(1).Infof("Processing %s", file.GetName())

		code, err := g.generate2(file)

		if err == errNoTargetService {
			glog.V(1).Infof("%s: %v", file.GetName(), err)
			continue
		}
		// fmt.Println("asdf", err)

		if err != nil {
			return nil, err
		}
		formatted, err := format.Source([]byte(code))
		// formatted := []byte(code)
		// if err != nil {
		// 	glog.Errorf("%v: %s", err, code)
		// 	return nil, err
		// }
		name := file.GetName()
		if file.GoPkg.Path != "" {
			name = fmt.Sprintf("%s/%s", file.GoPkg.Path, filepath.Base(name))
		}
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		output := fmt.Sprintf("%s.plugin.go", base)
		files = append(files, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(output),
			Content: proto.String(string(formatted)),
		})
		glog.V(1).Infof("Will emit %s", output)
		// glog.V(1).Infof("Will emit %s", output)
	}
	// glog.Error(files)

	return files, nil
}

func (g *generator) generate(file *descriptor.File) (string, error) {
	pkgSeen := make(map[string]bool)
	var imports []descriptor.GoPackage
	for _, pkg := range g.baseImports {
		pkgSeen[pkg.Path] = true
		imports = append(imports, pkg)
	}
	for _, svc := range file.Services {

		for _, m := range svc.Methods {
			pkg := m.RequestType.File.GoPkg
			if m.Options == nil || !proto.HasExtension(m.Options, options.E_Http) ||
				pkg == file.GoPkg || pkgSeen[pkg.Path] {
				continue
			}
			pkgSeen[pkg.Path] = true
			imports = append(imports, pkg)
		}
	}
	return applyTemplate(param{File: file, Imports: imports, UseRequestContext: g.useRequestContext})
}

func (g *generator) generate2(file *descriptor.File) (string, error) {
	pkgSeen := make(map[string]bool)
	var imports []descriptor.GoPackage
	for _, pkg := range g.baseImports {
		pkgSeen[pkg.Path] = true
		imports = append(imports, pkg)
	}
	for _, svc := range file.Services {

		for _, m := range svc.Methods {

			pkg := m.RequestType.File.GoPkg
			if m.Options == nil || !proto.HasExtension(m.Options, options.E_Http) ||
				pkg == file.GoPkg || pkgSeen[pkg.Path] {
				continue
			}
			pkgSeen[pkg.Path] = true
			imports = append(imports, pkg)
		}
	}

	for _, dep := range file.Dependency {
		d, err := g.reg.LookupFile(dep)
		if err != nil {
			glog.Error(err)
			continue
		}
		pkg := d.GoPkg
		if pkgSeen[pkg.Path] {
			continue
		}
		pkgSeen[pkg.Path] = true
		imports = append(imports, pkg)
		// glog.Fatal(d.GoPkg)
	}

	return applyTemplate2(param{File: file, Registry: g.reg, Imports: imports, UseRequestContext: g.useRequestContext})
}