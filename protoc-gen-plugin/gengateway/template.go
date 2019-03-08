package gengateway

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	stdprotogen "github.com/golang/protobuf/protoc-gen-go/generator"

	"github.com/golang/glog"
	"github.com/yourhe/grpc-gateway/protoc-gen-plugin/descriptor"
	"github.com/yourhe/grpc-gateway/utilities"
)

type param struct {
	*descriptor.File
	Registry          *descriptor.Registry
	Imports           []descriptor.GoPackage
	UseRequestContext bool
}

type binding struct {
	*descriptor.Binding
}

// HasQueryParam determines if the binding needs parameters in query string.
//
// It sometimes returns true even though actually the binding does not need.
// But it is not serious because it just results in a small amount of extra codes generated.
func (b binding) HasQueryParam() bool {
	if b.Body != nil && len(b.Body.FieldPath) == 0 {
		return false
	}
	fields := make(map[string]bool)
	for _, f := range b.Method.RequestType.Fields {
		fields[f.GetName()] = true
	}
	if b.Body != nil {
		delete(fields, b.Body.FieldPath.String())
	}
	for _, p := range b.PathParams {
		delete(fields, p.FieldPath.String())
	}
	return len(fields) > 0
}

func (b binding) QueryParamFilter() queryParamFilter {
	var seqs [][]string
	if b.Body != nil {
		seqs = append(seqs, strings.Split(b.Body.FieldPath.String(), "."))
	}
	for _, p := range b.PathParams {
		seqs = append(seqs, strings.Split(p.FieldPath.String(), "."))
	}
	return queryParamFilter{utilities.NewDoubleArray(seqs)}
}

func (b binding) PathTmplToJSTmpl() string {
	regex := regexp.MustCompile("{(.*?)}")
	return regex.ReplaceAllString(b.PathTmpl.Template, `:$1`)
	// return regex.String()
}

// queryParamFilter is a wrapper of utilities.DoubleArray which provides String() to output DoubleArray.Encoding in a stable and predictable format.
type queryParamFilter struct {
	*utilities.DoubleArray
}

func GeneratorGoMethod(method *descriptor.Method) string {
	return fmt.Sprintf("%s(%s)(%s, err error)", method.GetName(), GeneratorGoType(method.RequestType.Fields), GeneratorGoType(method.ResponseType.Fields))
}

func GeneratorFiledsGoName(pkg string, fields []*descriptor.Field) string {
	if pkg != "" {
		pkg = pkg + "."
	}
	fieldsBuf := bytes.Buffer{}
	for _, field := range fields {
		if fieldsBuf.Len() > 1 {
			fieldsBuf.WriteString(", ")
		}

		fieldsBuf.WriteString(fmt.Sprintf("%s%s", pkg, field.GetName()))

	}
	return fieldsBuf.String()
}

func GeneratorGoType(fields []*descriptor.Field) string {
	// stdprotogen "github.com/golang/protobuf/protoc-gen-go/generator"

	g := stdprotogen.New()
	fieldsBuf := bytes.Buffer{}
	// fieldsBuf.WrizteString("(")
	for _, field := range fields {

		t := ""
		if field.GetType().String() == "TYPE_MESSAGE" {
			msg, _ := reg.LookupMsg("", field.GetTypeName())
			// t = GeneratorGoType(msg.Fields)
			t = fmt.Sprintf("*%s", msg.GoType(""))
			// glog.Error()
		} else {
			t, _ = g.GoType(nil, field.FieldDescriptorProto)
			t = strings.Replace(t, "*", "", -1)

		}
		if fieldsBuf.Len() > 1 {
			fieldsBuf.WriteString(", ")
		}
		fieldsBuf.WriteString(fmt.Sprintf("%s %s", field.GetName(), t))

	}
	// fieldsBuf.WriteString(") ")
	return fieldsBuf.String()
	//
	// /users/yorhe/go/src/github.com/golang/protobuf/protoc-gen-go/generator/generator.go
}

func GeneratorGoService(method *descriptor.Method) string {
	t := `var in *{{.RequestType.GetName}}
	in = &{{.RequestType.GetName}}{
{{range .RequestType.Fields}}		{{.GetName | CamelCase}}: {{.GetName}},
{{end}}	}
	 `
	funcMap := template.FuncMap{
		"CamelCase": stdprotogen.CamelCase,
	}

	w := bytes.NewBuffer(nil)
	_ = template.Must(template.New("").Funcs(funcMap).Parse(t)).Execute(w, method)
	return w.String()

}

func (f queryParamFilter) String() string {
	encodings := make([]string, len(f.Encoding))
	for str, enc := range f.Encoding {
		encodings[enc] = fmt.Sprintf("%q: %d", str, enc)
	}
	e := strings.Join(encodings, ", ")
	return fmt.Sprintf("&utilities.DoubleArray{Encoding: map[string]int{%s}, Base: %#v, Check: %#v}", e, f.Base, f.Check)
}

type trailerParams struct {
	Services          []*descriptor.Service
	UseRequestContext bool
}

var interfacetmpl = `{{range .}}
type {{.GetName}} interface {
{{range .Methods}}
{{template "method" .}}
{{end}}
}
{{end}}
`
var methodtmpl = `	//{{.GetName}} ...
	{{.GetName}}(*{{.RequestType.GetName}}) *{{.ResponseType.GetName}}`

type PlugSt struct {
	*descriptor.File
	PKGN string
}

// {{.GetName}}({{.RequestType.GetName}}) {{.ResponseType.GetName}}
func applyInterface(wr io.Writer, p param) error {
	templatesDir := "/users/yorhe/go/src/github.com/yourhe/grpc-gateway/protoc-gen-plugin/gengateway/templates/"
	funcMap := template.FuncMap{
		"ToUpper":      strings.ToUpper,
		"ToLower":      strings.ToLower,
		"GoType":       GeneratorGoType,
		"GoMethod":     GeneratorGoMethod,
		"GoService":    GeneratorGoService,
		"CamelCase":    stdprotogen.CamelCase,
		"GoFieldsName": GeneratorFiledsGoName,
	}
	ti := template.Must(template.New("").Funcs(funcMap).ParseGlob((templatesDir + "*.tmpl")))
	// ti := template.Must(template.ParseGlob(templatesDir + "*.tmpl")).Funcs(funcMap)
	// ti := template.Must(template.New("interfacetmpl").Parse(interfacetmpl))
	ps := &PlugSt{File: p.File}
	ps.PKGN = strings.ToUpper(ps.GoPkg.Name)
	return ti.ExecuteTemplate(wr, "auth.tmpl", p)
	// return ti.ExecuteTemplate(wr, "auth.tmpl", p)
}
func applyTemplate(p param) (string, error) {
	w := bytes.NewBuffer(nil)
	if err := headerTemplate.Execute(w, p); err != nil {
		return "", err
	}
	var targetServices []*descriptor.Service
	for _, svc := range p.Services {
		var methodWithBindingsSeen bool
		for _, meth := range svc.Methods {
			glog.V(2).Infof("Processing %s.%s", svc.GetName(), meth.GetName())
			methName := strings.Title(*meth.Name)
			meth.Name = &methName
			for _, b := range meth.Bindings {
				methodWithBindingsSeen = true
				// glog.Errorf("herer %v", b.PathTmpl)

				if err := handlerTemplate.Execute(w, binding{Binding: b}); err != nil {
					return "", err
				}
			}
		}
		if methodWithBindingsSeen {
			targetServices = append(targetServices, svc)
		}
	}
	if len(targetServices) == 0 {
		return "", errNoTargetService
	}

	tp := trailerParams{
		Services:          targetServices,
		UseRequestContext: p.UseRequestContext,
	}
	if err := trailerTemplate.Execute(w, tp); err != nil {
		return "", err
	}
	return w.String(), nil
}

var reg *descriptor.Registry

func applyTemplate2(p param) (string, error) {
	// glog.Errorf("%v", p.GoPkg.Standard())
	// glog.Errorf("%v", p)
	reg = p.Registry
	w := bytes.NewBuffer(nil)
	applyInterface(w, p)
	return w.String(), nil
	// for _, t := range templates {
	// 	if err := t.Execute(w, p); err != nil {
	// 		return "", err
	// 	}
	// }

	// return w.String(), nil
}

func init() {
	// loadTemplates()
	// glog.Error(templates)

	// fmt.Println(templates)
}
func loadTemplates() {
	// templates = make(map[string]*template.Template)
	// tpl = template.Must(template.ParseGlob("templates/**/*.tmpl"))

	if templates == nil {
		templates = make(map[string]*template.Template)
	}

	templatesDir := "/users/yorhe/go/src/github.com/yourhe/grpc-gateway/protoc-gen-plugin/gengateway/templates/"

	layouts, err := filepath.Glob(templatesDir + "/*.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	// Generate our templates map from our layouts/ and includes/ directories
	for _, layout := range layouts {
		templates[filepath.Base(layout)] =
			template.Must(template.New(filepath.Base(layout)).ParseFiles(layout))

	}
	glog.Error(templates)
}

var templates map[string]*template.Template
var tpl *template.Template
var (
	headerTemplate = template.Must(template.New("header").Parse(`
		import "isomorphic-fetch";
		import reduxApi, {transformers} from "redux-api";
		import adapterFetch from "redux-api/lib/adapters/fetch";
		export default reduxApi({
		  // simple endpoint description
		  entry: "/api/v1/entry/:id",
		  // complex endpoint description
		  regions: {
			url: "/api/v1/regions",
			// reimplement default "transformers.object"
			transformer: transformers.array,
			// base endpoint options "fetch(url, options)"
			options: {
			  headers: {
				"Accept": "application/json"
			  }
			}
			
		  }
		}).use("fetch", adapterFetch(fetch)); // it's necessary to point using REST backend
		
		{{range $svc := .Services}}
		  var {{$svc.GetName}} = {}
		  const {{$svc.GetName}}Api = reduxApi({{$svc.GetName}}).use("fetch", adapterFetch(fetch));
		{{end}}
		
		`))
	handlerTemplate = template.Must(template.New("handler").Parse(`{{template "client-redux-func" .}}`))
	_               = template.Must(handlerTemplate.New("client-redux-func").Parse(`
		{{.Method.Service.GetName}}.{{.Method.GetName}} = {
			url: "{{.PathTmplToJSTmpl}}",
			method: "{{.HTTPMethod}}",
		};`))

	// (strings.Replace(`{{.Method.Service.GetName}}_{{.Method.GetName}}_{{.Index}}`, "\n", "", -1)))
	// _               = template.Must(handlerTemplate.New("client-streaming-request-func").Parse(``))
	// _ = template.Must(handlerTemplate.New("client-rpc-request-func").Parse(``)))
	// _ = template.Must(handlerTemplate.New("bidi-streaming-request-func").Parse(``))
	trailerTemplate = template.Must(template.New("trailer").Parse(``))
	// _ = template.Must(trailerTemplate.New("hydra-warden").Parse(``))
)

// var (
// 	headerTemplate = template.Must(template.New("header").Parse(`
// // Code generated by protoc-gen-grpc-gateway. DO NOT EDIT.
// // source: {{.GetName}}

// /*
// Package {{.GoPkg.Name}} is a reverse proxy.

// It translates gRPC into RESTful JSON APIs.
// */
// package {{.GoPkg.Name}}
// import (
// 	{{range $i := .Imports}}{{if $i.Standard}}{{$i | printf "%s\n"}}{{end}}{{end}}

// 	{{range $i := .Imports}}{{if not $i.Standard}}{{$i | printf "%s\n"}}{{end}}{{end}}

// )

// var _ codes.Code
// var _ io.Reader
// var _ status.Status
// var _ = runtime.String
// var _ = utilities.NewDoubleArray
// var HydraClient hydra.SDK
// `))

// 	handlerTemplate = template.Must(template.New("handler").Parse(`

// {{if and .Method.GetClientStreaming .Method.GetServerStreaming}}
// {{template "bidi-streaming-request-func" .}}
// {{else if .Method.GetClientStreaming}}
// {{template "client-streaming-request-func" .}}
// {{else}}
// {{template "client-rpc-request-func" .}}
// {{end}}
// `))

// 	_ = template.Must(handlerTemplate.New("request-func-signature").Parse(strings.Replace(`
// {{if .Method.GetServerStreaming}}
// func request_{{.Method.Service.GetName}}_{{.Method.GetName}}_{{.Index}}(ctx context.Context, marshaler runtime.Marshaler, client {{.Method.Service.GetName}}Client, req *http.Request, pathParams map[string]string) ({{.Method.Service.GetName}}_{{.Method.GetName}}Client, runtime.ServerMetadata, error)
// {{else}}
// func request_{{.Method.Service.GetName}}_{{.Method.GetName}}_{{.Index}}(ctx context.Context, marshaler runtime.Marshaler, client {{.Method.Service.GetName}}Client, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error)
// {{end}}`, "\n", "", -1)))

// 	_ = template.Must(handlerTemplate.New("client-streaming-request-func").Parse(`
// {{template "request-func-signature" .}} {
// 	var metadata runtime.ServerMetadata
// 	stream, err := client.{{.Method.GetName}}(ctx)
// 	if err != nil {
// 		grpclog.Printf("Failed to start streaming: %v", err)
// 		return nil, metadata, err
// 	}
// 	dec := marshaler.NewDecoder(req.Body)
// 	for {
// 		var protoReq {{.Method.RequestType.GoType .Method.Service.File.GoPkg.Path}}
// 		err = dec.Decode(&protoReq)
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			grpclog.Printf("Failed to decode request: %v", err)
// 			return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
// 		}
// 		if err = stream.Send(&protoReq); err != nil {
// 			grpclog.Printf("Failed to send request: %v", err)
// 			return nil, metadata, err
// 		}
// 	}

// 	if err := stream.CloseSend(); err != nil {
// 		grpclog.Printf("Failed to terminate client stream: %v", err)
// 		return nil, metadata, err
// 	}
// 	header, err := stream.Header()
// 	if err != nil {
// 		grpclog.Printf("Failed to get header from client: %v", err)
// 		return nil, metadata, err
// 	}
// 	metadata.HeaderMD = header
// {{if .Method.GetServerStreaming}}
// 	return stream, metadata, nil
// {{else}}
// 	msg, err := stream.CloseAndRecv()
// 	metadata.TrailerMD = stream.Trailer()
// 	return msg, metadata, err
// {{end}}
// }
// `))

// 	_ = template.Must(handlerTemplate.New("client-rpc-request-func").Parse(`
// {{if .HasQueryParam}}
// var (
// 	filter_{{.Method.Service.GetName}}_{{.Method.GetName}}_{{.Index}} = {{.QueryParamFilter}}
// )
// {{end}}
// {{template "request-func-signature" .}} {
// 	var protoReq {{.Method.RequestType.GoType .Method.Service.File.GoPkg.Path}}
// 	var metadata runtime.ServerMetadata
// {{if .Body}}
// 	if err := marshaler.NewDecoder(req.Body).Decode(&{{.Body.RHS "protoReq"}}); err != nil {
// 		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
// 	}
// {{end}}
// {{if .PathParams}}
// 	var (
// 		val string
// 		ok bool
// 		err error
// 		_ = err
// 	)
// 	{{range $param := .PathParams}}
// 	val, ok = pathParams[{{$param | printf "%q"}}]
// 	if !ok {
// 		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", {{$param | printf "%q"}})
// 	}
// {{if $param.IsNestedProto3 }}
// 	err = runtime.PopulateFieldFromPath(&protoReq, {{$param | printf "%q"}}, val)
// {{else}}
// 	{{$param.RHS "protoReq"}}, err = {{$param.ConvertFuncExpr}}(val)
// {{end}}
// 	if err != nil {
// 		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", {{$param | printf "%q"}}, err)
// 	}
// 	{{end}}
// {{end}}
// {{if .HasQueryParam}}
// 	if err := runtime.PopulateQueryParameters(&protoReq, req.URL.Query(), filter_{{.Method.Service.GetName}}_{{.Method.GetName}}_{{.Index}}); err != nil {
// 		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
// 	}
// {{end}}
// {{if .Method.GetServerStreaming}}
// 	stream, err := client.{{.Method.GetName}}(ctx, &protoReq)
// 	if err != nil {
// 		return nil, metadata, err
// 	}
// 	header, err := stream.Header()
// 	if err != nil {
// 		return nil, metadata, err
// 	}
// 	metadata.HeaderMD = header
// 	return stream, metadata, nil
// {{else}}
// 	msg, err := client.{{.Method.GetName}}(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
// 	return msg, metadata, err
// {{end}}
// }`))

// 	_ = template.Must(handlerTemplate.New("bidi-streaming-request-func").Parse(`
// {{template "request-func-signature" .}} {
// 	var metadata runtime.ServerMetadata
// 	stream, err := client.{{.Method.GetName}}(ctx)
// 	if err != nil {
// 		grpclog.Printf("Failed to start streaming: %v", err)
// 		return nil, metadata, err
// 	}
// 	dec := marshaler.NewDecoder(req.Body)
// 	handleSend := func() error {
// 		var protoReq {{.Method.RequestType.GoType .Method.Service.File.GoPkg.Path}}
// 		err = dec.Decode(&protoReq)
// 		if err == io.EOF {
// 			return err
// 		}
// 		if err != nil {
// 			grpclog.Printf("Failed to decode request: %v", err)
// 			return err
// 		}
// 		if err = stream.Send(&protoReq); err != nil {
// 			grpclog.Printf("Failed to send request: %v", err)
// 			return err
// 		}
// 		return nil
// 	}
// 	if err := handleSend(); err != nil {
// 		if cerr := stream.CloseSend(); cerr != nil {
// 			grpclog.Printf("Failed to terminate client stream: %v", cerr)
// 		}
// 		if err == io.EOF {
// 			return stream, metadata, nil
// 		}
// 		return nil, metadata, err
// 	}
// 	go func() {
// 		for {
// 			if err := handleSend(); err != nil {
// 				break
// 			}
// 		}
// 		if err := stream.CloseSend(); err != nil {
// 			grpclog.Printf("Failed to terminate client stream: %v", err)
// 		}
// 	}()
// 	header, err := stream.Header()
// 	if err != nil {
// 		grpclog.Printf("Failed to get header from client: %v", err)
// 		return nil, metadata, err
// 	}
// 	metadata.HeaderMD = header
// 	return stream, metadata, nil
// }
// `))

// 	trailerTemplate = template.Must(template.New("trailer").Parse(`
// 		{{template "hydra-warden" .}}
// 		{{$UseRequestContext := .UseRequestContext}}
// {{range $svc := .Services}}
// // Register{{$svc.GetName}}HandlerFromEndpoint is same as Register{{$svc.GetName}}Handler but
// // automatically dials to "endpoint" and closes the connection when "ctx" gets done.
// func Register{{$svc.GetName}}HandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error) {
// 	conn, err := grpc.Dial(endpoint, opts...)
// 	if err != nil {
// 		return err
// 	}
// 	defer func() {
// 		if err != nil {
// 			if cerr := conn.Close(); cerr != nil {
// 				grpclog.Printf("Failed to close conn to %s: %v", endpoint, cerr)
// 			}
// 			return
// 		}
// 		go func() {
// 			<-ctx.Done()
// 			if cerr := conn.Close(); cerr != nil {
// 				grpclog.Printf("Failed to close conn to %s: %v", endpoint, cerr)
// 			}
// 		}()
// 	}()

// 	return Register{{$svc.GetName}}Handler(ctx, mux, conn)
// }

// // Register{{$svc.GetName}}Handler registers the http handlers for service {{$svc.GetName}} to "mux".
// // The handlers forward requests to the grpc endpoint over "conn".
// func Register{{$svc.GetName}}Handler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
// 	return Register{{$svc.GetName}}HandlerClient(ctx, mux, New{{$svc.GetName}}Client(conn))
// }

// // Register{{$svc.GetName}}Handler registers the http handlers for service {{$svc.GetName}} to "mux".
// // The handlers forward requests to the grpc endpoint over the given implementation of "{{$svc.GetName}}Client".
// // Note: the gRPC framework executes interceptors within the gRPC handler. If the passed in "{{$svc.GetName}}Client"
// // doesn't go through the normal gRPC flow (creating a gRPC client etc.) then it will be up to the passed in
// // "{{$svc.GetName}}Client" to call the correct interceptors.
// func Register{{$svc.GetName}}HandlerClient(ctx context.Context, mux *runtime.ServeMux, client {{$svc.GetName}}Client) error {
// 	{{range $m := $svc.Methods}}
// 	{{range $b := $m.Bindings}}

// 	{{if $m.Policy}}
// 	// {{$m.Policy}}
// 	//exampleAuthFunc(w,req,pathParams)
// 	mux.Handle(
// 		{{$b.HTTPMethod | printf "%q"}},
// 		pattern_{{$svc.GetName}}_{{$m.GetName}}_{{$b.Index}},
// 		exampleAuthFunc(
// 			&policy.PolicyRule{
// 				Action:  "{{$svc.GetName}}:{{$m.GetName}}",
// 			//	Action:    {{ $m.Policy.Action | printf "%q" }},
// 				Resources: {{$m.Policy.Resources | printf "%q" }},
// 				Effect:    "allow",
// 			},
// 			func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
// 	{{else}}
// 	mux.Handle({{$b.HTTPMethod | printf "%q"}}, pattern_{{$svc.GetName}}_{{$m.GetName}}_{{$b.Index}}, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
// 	{{end}}
// 	{{- if $UseRequestContext }}
// 		ctx, cancel := context.WithCancel(req.Context())
// 	{{- else -}}
// 		ctx, cancel := context.WithCancel(ctx)
// 	{{- end }}
// 		defer cancel()
// 		if cn, ok := w.(http.CloseNotifier); ok {
// 			go func(done <-chan struct{}, closed <-chan bool) {
// 				select {
// 				case <-done:
// 				case <-closed:
// 					cancel()
// 				}
// 			}(ctx.Done(), cn.CloseNotify())
// 		}

// 		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
// 		rctx, err := runtime.AnnotateContext(ctx, mux, req)
// 		if err != nil {
// 			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
// 			return
// 		}
// 		resp, md, err := request_{{$svc.GetName}}_{{$m.GetName}}_{{$b.Index}}(rctx, inboundMarshaler, client, req, pathParams)
// 		ctx = runtime.NewServerMetadataContext(ctx, md)
// 		if err != nil {
// 			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
// 			return
// 		}
// 		{{if $m.GetServerStreaming}}
// 		forward_{{$svc.GetName}}_{{$m.GetName}}_{{$b.Index}}(ctx, mux, outboundMarshaler, w, req, func() (proto.Message, error) { return resp.Recv() }, mux.GetForwardResponseOptions()...)
// 		{{else}}
// 		forward_{{$svc.GetName}}_{{$m.GetName}}_{{$b.Index}}(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
// 		{{end}}
// 	{{if $m.Policy}}
// 	// {{$m.Policy}}
// 	// end examplyAuthfunc
// 	}))
// 	{{else}}
// 	})
// 	{{end}}
// 	{{end}}
// 	{{end}}
// 	return nil
// }

// var (
// 	{{range $m := $svc.Methods}}
// 	{{range $b := $m.Bindings}}
// 	pattern_{{$svc.GetName}}_{{$m.GetName}}_{{$b.Index}} = runtime.MustPattern(runtime.NewPattern({{$b.PathTmpl.Version}}, {{$b.PathTmpl.OpCodes | printf "%#v"}}, {{$b.PathTmpl.Pool | printf "%#v"}}, {{$b.PathTmpl.Verb | printf "%q"}}))
// 	{{end}}
// 	{{end}}
// )

// var (

// 	{{range $m := $svc.Methods}}
// 	{{range $b := $m.Bindings}}
// 	forward_{{$svc.GetName}}_{{$m.GetName}}_{{$b.Index}} = {{if $m.GetServerStreaming}}runtime.ForwardResponseStream{{else}}runtime.ForwardResponseMessage{{end}}
// 	{{end}}
// 	{{end}}

// )
// // type RequestPolicyMap map[string]*policy.PolicyRule
// type RequestPolicyMap struct {
// 	PolicyMap map[string]*policy.PolicyRule
// 	PatternList []ServiceMothedMap
// }
// type ServiceMothedMap struct {
// 	Name string
// 	Pattern *runtime.Pattern
// }

// var (
// 	RPM = RequestPolicyMap{
// 	PolicyMap: map[string]*policy.PolicyRule{
// 	{{range $m := $svc.Methods}}
// 	{{range $b := $m.Bindings}}
// 		{{if $m.Policy}}
// 		"{{$b.HTTPMethod}} {{$b.PathTmpl.Template}}" : &policy.PolicyRule{
// 			Action:  "{{$svc.GetName}}:{{$m.GetName}}",
// 			Resources: {{$m.Policy.Resources | printf "%q" }},
// 			Effect:    "allow",
// 		},
// 		{{end}}

// 	{{end}}
// 	{{end}}
// 	},
// 	// PatternList => []ServiceMothedMap
// 	PatternList: []ServiceMothedMap{
// 	{{range $m := $svc.Methods}}
// 	{{range $b := $m.Bindings}}
// 		{
// 			Name:"{{$b.PathTmpl.Template}}",
// 			Pattern: &pattern_{{$svc.GetName}}_{{$m.GetName}}_{{$b.Index}},
// 		},
// 	{{end}}
// 	{{end}}
// 	},
// 	}

// )

// {{end}}`))

// 	_ = template.Must(trailerTemplate.New("hydra-warden").Parse(`
// var (
// headerAuthorize = "authorization"
// )
// // func exampleAuthFunc(rule *policy.PolicyRule, w http.ResponseWriter, req *http.Request, pathParams map[string]string) (context.Context, error) {
// func exampleAuthFunc(rule *policy.PolicyRule, fn func(w http.ResponseWriter, req *http.Request, pathParams map[string]string)) func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
// 	return func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
// 		fn(w,req,pathParams)
// 		return
// 		val := req.Header.Get(headerAuthorize)
// 		var expectedScheme = "bearer"
// 		if val == "" {
// 			// return "", grpc.Errorf(codes.Unauthenticated, "Request unauthenticated with "+expectedScheme)
// 			// runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
// 			runtime.DefaultOtherErrorHandler(w, req, "Request unauthenticated with v:"+val, http.StatusUnauthorized)
// 			return

// 		}
// 		splits := strings.SplitN(val, " ", 2)
// 		if len(splits) < 2 {
// 			runtime.DefaultOtherErrorHandler(w, req, "Bad authorization string", http.StatusUnauthorized)

// 			// return "", grpc.Errorf(codes.Unauthenticated, "Bad authorization string")
// 			return
// 		}
// 		if strings.ToLower(splits[0]) != strings.ToLower(expectedScheme) {
// 			runtime.DefaultOtherErrorHandler(w, req, "Request unauthenticated with ", http.StatusUnauthorized)
// 			return
// 			// return "", grpc.Errorf(codes.Unauthenticated, "Request unauthenticated with "+expectedScheme)
// 		}
// 		token := splits[1]
// 		Accesstoken, _, err := HydraClient.DoesWardenAllowTokenAccessRequest(swagger.WardenTokenAccessRequest{
// 			Action:   rule.Action,
// 			Resource: rule.Resources,
// 			Token:    token,
// 		})
// 		if err != nil {
// 			runtime.DefaultOtherErrorHandler(w, req, err.Error(), http.StatusUnauthorized)
// 			return
// 		}
// 		if Accesstoken.Allowed != true {
// 			runtime.DefaultOtherErrorHandler(w, req,"Request unauthenticated with not Allowed", http.StatusUnauthorized)
// 			return
// 		}
// 		fn(w,req,pathParams)
// 	}
// }
// 	`))
// )
