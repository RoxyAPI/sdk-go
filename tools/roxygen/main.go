// Command roxygen is the RoxyAPI Go SDK code generator. The OpenAPI spec at
// roxyapi.com is the single source of truth; everything below regenerates from it.
//
//	roxygen generate    Fetch the spec, normalize it for oapi-codegen, generate the
//	                    typed client (roxyapi.gen.go), generate the domain-grouped
//	                    facade (roxy.gen.go), then sync the spec-derived docs.
//	roxygen sync-docs   Regenerate only the generated regions of README.md, AGENTS.md
//	                    and docs/llms-full.txt (between BEGIN/END markers): DOMAINS,
//	                    LANGS and METHODS from the spec, NOPARAMS from the facade.
//	                    Run by CI and the pre-push hook to fail on drift.
//
// Run from the repository root: `go run ./tools/roxygen <generate|sync-docs>`.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/util"
)

const (
	specURL    = "https://roxyapi.com/api/v2/openapi.json"
	specPath   = "specs/openapi.json"
	clientPath = "roxyapi.gen.go"
	facadePath = "roxy.gen.go"
)

var httpVerbs = map[string]int{"get": 0, "post": 1, "put": 2, "patch": 3, "delete": 4}

func main() {
	if len(os.Args) < 2 {
		fail("usage: roxygen <generate|sync-docs>")
	}
	switch os.Args[1] {
	case "generate":
		generate()
	case "sync-docs":
		syncDocs()
	default:
		fail("usage: roxygen <generate|sync-docs>")
	}
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}

func check(err error, msg string) {
	if err != nil {
		fail("%s: %v", msg, err)
	}
}

// ─── generate ────────────────────────────────────────────────────────────────

func generate() {
	raw := loadSpec()

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var spec map[string]any
	check(dec.Decode(&spec), "parse spec")

	patchPathParameters(spec)
	normalizeErrors(spec)

	check(os.MkdirAll("specs", 0o755), "mkdir specs")
	pretty, err := json.MarshalIndent(spec, "", "  ")
	check(err, "marshal spec")
	check(os.WriteFile(specPath, append(pretty, '\n'), 0o644), "write spec")
	fmt.Printf("Spec saved to %s\n", specPath)

	swagger, err := util.LoadSwagger(specPath)
	check(err, "load normalized spec")

	code, err := codegen.Generate(swagger, codegen.Configuration{
		PackageName: "roxyapi",
		Generate:    codegen.GenerateOptions{Models: true, Client: true},
		OutputOptions: codegen.OutputOptions{
			NameNormalizer:              "ToCamelCaseWithInitialisms",
			ClientResponseBytesFunction: true,
		},
	})
	check(err, "generate client")
	check(os.WriteFile(clientPath, []byte(code), 0o644), "write client")
	fmt.Printf("Typed client written to %s\n", clientPath)

	buildFacade(spec)
	syncDocs()
}

// loadSpec returns the raw spec bytes, from disk when ROXYAPI_SPEC_FILE is set and from the API
// otherwise. Reading from a file keeps generation offline and byte-reproducible, which is what the
// codegen drift check in CI relies on.
func loadSpec() []byte {
	if path := os.Getenv("ROXYAPI_SPEC_FILE"); path != "" {
		fmt.Printf("Reading OpenAPI spec from %s (offline, ROXYAPI_SPEC_FILE)\n", path)
		raw, err := os.ReadFile(path)
		check(err, "read spec file")
		return raw
	}
	fmt.Printf("Fetching OpenAPI spec from %s\n", specURL)
	return fetchSpec()
}

// fetchSpec retries with exponential backoff: a transient upstream error (e.g. a
// CDN 520) must not fail the daily release run.
func fetchSpec() []byte {
	const attempts = 5
	for attempt := 1; ; attempt++ {
		body, err := fetchSpecOnce()
		if err == nil {
			return body
		}
		if attempt == attempts {
			fail("fetch spec: %v (after %d attempts)", err, attempts)
		}
		delay := time.Duration(1<<attempt) * time.Second
		fmt.Printf("Fetch attempt %d/%d failed (%v), retrying in %s\n", attempt, attempts, err, delay)
		time.Sleep(delay)
	}
}

func fetchSpecOnce() ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, specURL, nil)
	check(err, "build request")
	req.Header.Set("Cache-Control", "no-cache")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ─── spec normalization ──────────────────────────────────────────────────────

// patchPathParameters forces required:true on every path parameter. OpenAPI requires
// it and the generator rejects an optional path parameter; the served spec still emits
// required:false on one route (/kabbalah/names/{number}, measured 2026-09-19). This
// step is dead the day a live run stops printing the Forced line; delete it then.
func patchPathParameters(spec map[string]any) {
	paths, _ := spec["paths"].(map[string]any)
	fixed := 0
	for _, item := range paths {
		pathItem, ok := item.(map[string]any)
		if !ok {
			continue
		}
		for key, val := range pathItem {
			if key == "parameters" {
				fixed += forceRequiredPath(val)
				continue
			}
			if _, isVerb := httpVerbs[key]; !isVerb {
				continue
			}
			if op, ok := val.(map[string]any); ok {
				fixed += forceRequiredPath(op["parameters"])
			}
		}
	}
	if fixed > 0 {
		fmt.Printf("Forced required:true on %d path parameters.\n", fixed)
	}
}

func forceRequiredPath(params any) int {
	list, ok := params.([]any)
	if !ok {
		return 0
	}
	fixed := 0
	for _, p := range list {
		param, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if param["in"] == "path" && param["required"] != true {
			param["required"] = true
			fixed++
		}
	}
	return fixed
}

// normalizeErrors adds one shared ErrorResponse schema and repoints every 4xx and
// 5xx response at it, so the generator emits a single typed error body (mapped to
// *RoxyError in errors.go) instead of a distinct inline struct per operation. The
// served spec inlines the same `{ error, code }` shape on every error status.
func normalizeErrors(spec map[string]any) {
	components, _ := spec["components"].(map[string]any)
	if components == nil {
		components = map[string]any{}
		spec["components"] = components
	}
	schemas, _ := components["schemas"].(map[string]any)
	if schemas == nil {
		schemas = map[string]any{}
		components["schemas"] = schemas
	}

	schemas["ErrorResponseIssue"] = map[string]any{
		"type":        "object",
		"description": "A single field-level validation failure from a 400 response.",
		"properties": map[string]any{
			"path":     stringProp("Dot-separated field path, or (root) for a top-level error."),
			"message":  stringProp("Human readable description of this failure."),
			"code":     stringProp("Validation issue code, for example invalid_type or too_small."),
			"expected": stringProp("Expected type, when the issue is a type mismatch."),
		},
	}
	schemas["ErrorResponse"] = map[string]any{
		"type":        "object",
		"description": "Error body returned by every RoxyAPI endpoint on a 4xx or 5xx response.",
		"required":    []any{"error", "code"},
		"properties": map[string]any{
			"error": stringProp("Human readable error message. Wording may change; switch on code instead."),
			"code":  stringProp("Stable machine readable error code, for example validation_error or rate_limit_exceeded."),
			"issues": map[string]any{
				"type":        "array",
				"description": "Present on 400 responses: every field that failed validation.",
				"items":       map[string]any{"$ref": "#/components/schemas/ErrorResponseIssue"},
			},
			"allow": map[string]any{
				"type":        "array",
				"description": "Present on 405 responses: the HTTP methods this path accepts.",
				"items":       map[string]any{"type": "string"},
			},
			"docs": stringProp("Link to the documentation for this domain, when available."),
		},
	}

	errRef := map[string]any{"$ref": "#/components/schemas/ErrorResponse"}
	paths, _ := spec["paths"].(map[string]any)
	normalized := 0
	for path, item := range paths {
		pathItem, ok := item.(map[string]any)
		if !ok {
			continue
		}
		for verb, val := range pathItem {
			if _, isVerb := httpVerbs[verb]; !isVerb {
				continue
			}
			op, ok := val.(map[string]any)
			if !ok {
				continue
			}
			responses, ok := op["responses"].(map[string]any)
			if !ok {
				continue
			}
			for code, r := range responses {
				status, err := strconv.Atoi(code)
				if err != nil || status < 400 {
					continue
				}
				resp, ok := r.(map[string]any)
				if !ok {
					continue
				}
				if _, isRef := resp["$ref"]; isRef {
					fail("normalize errors: %s %s %s is a $ref response, which this generator has never seen; teach normalizeErrors the shape before regenerating", strings.ToUpper(verb), path, code)
				}
				resp["content"] = map[string]any{
					"application/json": map[string]any{"schema": cloneMap(errRef)},
				}
				normalized++
			}
		}
	}
	fmt.Printf("Normalized %d error responses to ErrorResponse.\n", normalized)
}

func stringProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// ─── facade generation ───────────────────────────────────────────────────────

// buildFacade emits roxy.gen.go: a domain-grouped facade over the generated
// ClientWithResponses. It reads the real generated method signatures from the AST
// (so the facade is signature-faithful by construction; `go build` is the backstop)
// and groups them by URL first segment, matching the operations of the spec.
func buildFacade(spec map[string]any) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, clientPath, nil, parser.ParseComments)
	check(err, "parse generated client")

	domains := domainsInOrder(spec)
	segByOpKey := segmentByOpKey(domains)
	var orderedSegments []string
	for _, d := range domains {
		orderedSegments = append(orderedSegments, d.segment)
	}

	imports := importMap(file)
	used := map[string]bool{}
	methodsBySeg := map[string][]*ast.FuncDecl{}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || receiverType(fn.Recv) != "ClientWithResponses" {
			continue
		}
		name := fn.Name.Name
		if !strings.HasSuffix(name, "WithResponse") || strings.HasSuffix(name, "WithBodyWithResponse") {
			continue
		}
		opName := strings.TrimSuffix(name, "WithResponse")
		seg, ok := segByOpKey[normalizeKey(opName)]
		if !ok {
			fail("facade: generated method %s matches no operation of the spec, so it would be missing from the facade", name)
		}
		methodsBySeg[seg] = append(methodsBySeg[seg], fn)
		collectSelectorPkgs(fn.Type, used)
	}
	if got, want := countMethods(methodsBySeg), len(segByOpKey); got != want {
		fail("facade: %d generated methods for %d spec operations; a route is missing from the client", got, want)
	}

	var b strings.Builder
	b.WriteString("// Code generated by roxygen; DO NOT EDIT.\n//\n")
	b.WriteString("// Domain-grouped facade over the generated ClientWithResponses. Each method wraps\n")
	b.WriteString("// the matching <Operation>WithResponse call and maps a 4xx/5xx response to a\n")
	b.WriteString("// *RoxyError. Regenerated from the OpenAPI spec on every release.\n\n")
	b.WriteString("package roxyapi\n\n")
	writeImports(&b, imports, used)

	// Aggregate struct + per-domain service fields, in canonical tag order.
	b.WriteString("// Roxy is the domain-grouped entry point returned by NewRoxy.\n")
	b.WriteString("type Roxy struct {\n")
	b.WriteString("\tclient *ClientWithResponses\n")
	for _, seg := range orderedSegments {
		if len(methodsBySeg[seg]) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("\t%s *%sService\n", pascal(seg), pascal(seg)))
	}
	b.WriteString("}\n\n")

	b.WriteString("func newRoxy(c *ClientWithResponses) *Roxy {\n")
	b.WriteString("\tr := &Roxy{client: c}\n")
	for _, seg := range orderedSegments {
		if len(methodsBySeg[seg]) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("\tr.%s = &%sService{client: c}\n", pascal(seg), pascal(seg)))
	}
	b.WriteString("\treturn r\n}\n\n")

	for _, seg := range orderedSegments {
		methods := methodsBySeg[seg]
		if len(methods) == 0 {
			continue
		}
		svc := pascal(seg) + "Service"
		sort.Slice(methods, func(i, j int) bool { return methods[i].Name.Name < methods[j].Name.Name })
		b.WriteString(fmt.Sprintf("// %s groups the %s endpoints.\n", svc, seg))
		b.WriteString(fmt.Sprintf("type %s struct{ client *ClientWithResponses }\n\n", svc))
		for _, fn := range methods {
			writeWrapper(&b, fset, svc, fn)
		}
	}

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		// Write the unformatted source so the error is debuggable.
		_ = os.WriteFile(facadePath, []byte(b.String()), 0o644)
		fail("format facade: %v", err)
	}
	check(os.WriteFile(facadePath, formatted, 0o644), "write facade")
	fmt.Printf("Facade written to %s (%d methods across %d domains).\n", facadePath, countMethods(methodsBySeg), len(methodsBySeg))
}

func countMethods(methodsBySeg map[string][]*ast.FuncDecl) int {
	total := 0
	for _, m := range methodsBySeg {
		total += len(m)
	}
	return total
}

func writeWrapper(b *strings.Builder, fset *token.FileSet, svc string, fn *ast.FuncDecl) {
	opName := strings.TrimSuffix(fn.Name.Name, "WithResponse")
	params := fn.Type.Params
	results := fn.Type.Results

	var paramParts, args []string
	for _, f := range params.List {
		ts := exprString(fset, f.Type)
		_, variadic := f.Type.(*ast.Ellipsis)
		if len(f.Names) == 0 {
			paramParts = append(paramParts, ts)
			continue
		}
		var names []string
		for _, n := range f.Names {
			names = append(names, n.Name)
			if variadic {
				args = append(args, n.Name+"...")
			} else {
				args = append(args, n.Name)
			}
		}
		paramParts = append(paramParts, strings.Join(names, ", ")+" "+ts)
	}

	var resultParts []string
	for _, f := range results.List {
		resultParts = append(resultParts, exprString(fset, f.Type))
	}

	b.WriteString(fmt.Sprintf("func (s *%s) %s(%s) (%s) {\n", svc, opName, strings.Join(paramParts, ", "), strings.Join(resultParts, ", ")))
	b.WriteString(fmt.Sprintf("\tresp, err := s.client.%s(%s)\n", fn.Name.Name, strings.Join(args, ", ")))
	b.WriteString("\tif err != nil {\n\t\treturn resp, err\n\t}\n")
	b.WriteString("\treturn resp, asRoxyError(resp)\n}\n\n")
}

// receiverType returns the type name of a pointer receiver (`*Foo` -> "Foo"), or "".
func receiverType(recv *ast.FieldList) string {
	if len(recv.List) != 1 {
		return ""
	}
	star, ok := recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return ""
	}
	id, ok := star.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return id.Name
}

func exprString(fset *token.FileSet, e ast.Expr) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, fset, e)
	return b.String()
}

// collectSelectorPkgs records the package qualifier of every selector expression in
// a function signature (e.g. context.Context -> "context") so writeImports can emit
// exactly the imports the facade signatures reference.
func collectSelectorPkgs(t *ast.FuncType, used map[string]bool) {
	visit := func(fl *ast.FieldList) {
		if fl == nil {
			return
		}
		for _, f := range fl.List {
			ast.Inspect(f.Type, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok {
					if id, ok := sel.X.(*ast.Ident); ok {
						used[id.Name] = true
					}
				}
				return true
			})
		}
	}
	visit(t.Params)
	visit(t.Results)
}

func importMap(file *ast.File) map[string]string {
	out := map[string]string{}
	for _, spec := range file.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		name := ""
		if spec.Name != nil {
			name = spec.Name.Name
		} else {
			parts := strings.Split(path, "/")
			name = parts[len(parts)-1]
		}
		out[name] = path
	}
	return out
}

func writeImports(b *strings.Builder, imports map[string]string, used map[string]bool) {
	var lines []string
	for name := range used {
		path, ok := imports[name]
		if !ok {
			continue
		}
		last := path[strings.LastIndex(path, "/")+1:]
		if last == name {
			lines = append(lines, fmt.Sprintf("\t%q", path))
		} else {
			lines = append(lines, fmt.Sprintf("\t%s %q", name, path))
		}
	}
	if len(lines) == 0 {
		return
	}
	sort.Strings(lines)
	b.WriteString("import (\n")
	b.WriteString(strings.Join(lines, "\n"))
	b.WriteString("\n)\n\n")
}

// ─── sync-docs ───────────────────────────────────────────────────────────────

func syncDocs() {
	raw, err := os.ReadFile(specPath)
	if err != nil {
		fail("sync-docs: %s not found, run `generate` first: %v", specPath, err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var spec map[string]any
	check(dec.Decode(&spec), "sync-docs parse spec")

	domains := domainsInOrder(spec)
	table := renderDomainsTable(domains)
	langs := renderLangs(domains)
	noParams := renderNoParams(facadeMethods())
	changed := false
	changed = swap("README.md", "DOMAINS", table) || changed
	changed = swap("README.md", "NOPARAMS", noParams) || changed
	changed = swap("AGENTS.md", "DOMAINS", table) || changed
	changed = swap("AGENTS.md", "LANGS", langs) || changed
	changed = swap("AGENTS.md", "NOPARAMS", noParams) || changed
	changed = swap("docs/llms-full.txt", "LANGS", langs) || changed
	changed = swap("docs/llms-full.txt", "METHODS", renderMethods(domains)) || changed
	changed = swap("docs/llms-full.txt", "NOPARAMS", noParams) || changed

	total := 0
	for _, d := range domains {
		total += len(d.ops)
	}
	state := "unchanged"
	if changed {
		state = "updated"
	}
	fmt.Printf("sync-docs: %d domains, %d endpoints. Docs %s.\n", len(domains), total, state)
}

// facadeMethod is one method of roxy.gen.go as it will be called: the accessor
// (`Astrology`), the method name and the argument names except the request editors.
type facadeMethod struct {
	accessor, name string
	args           []string
}

// facadeMethods reads the generated facade back, so every doc line that names a
// method or its arity comes from the code that ships and not from a second rule.
func facadeMethods() []facadeMethod {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, facadePath, nil, 0)
	if err != nil {
		fail("sync-docs: %s not readable, run `generate` first: %v", facadePath, err)
	}
	var out []facadeMethod
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || !strings.HasSuffix(receiverType(fn.Recv), "Service") {
			continue
		}
		m := facadeMethod{accessor: strings.TrimSuffix(receiverType(fn.Recv), "Service"), name: fn.Name.Name}
		for _, f := range fn.Type.Params.List {
			if _, variadic := f.Type.(*ast.Ellipsis); variadic {
				continue
			}
			for _, n := range f.Names {
				m.args = append(m.args, n.Name)
			}
		}
		out = append(out, m)
	}
	if len(out) == 0 {
		fail("sync-docs: %s declares no service methods", facadePath)
	}
	return out
}

// renderNoParams lists every facade method that takes no `params` argument, in
// facade order (canonical domain order, then method name). Passing nil to one of
// these compiles and panics at runtime, so the docs carry the exact list.
func renderNoParams(methods []facadeMethod) string {
	var b strings.Builder
	b.WriteString("<!-- BEGIN:NOPARAMS -->\n")
	for _, m := range methods {
		if slices.Contains(m.args, "params") {
			continue
		}
		b.WriteString(fmt.Sprintf("- `roxy.%s.%s(%s)`\n", m.accessor, m.name, strings.Join(m.args, ", ")))
	}
	b.WriteString("<!-- END:NOPARAMS -->")
	return b.String()
}

// renderLangs states the `lang` query parameter as the spec declares it: the codes
// and default (one shape, asserted identical on every operation that has it) and
// which accessors carry it at all. Display names and per-domain translation depth
// are not in the spec and stay in prose outside the markers.
func renderLangs(domains []domainInfo) string {
	var codes []string
	def := ""
	var supported, englishOnly []string
	for _, d := range domains {
		hasLang := false
		for _, op := range d.ops {
			p := op.langParam()
			if p == nil {
				continue
			}
			hasLang = true
			schema, _ := p["schema"].(map[string]any)
			enum, _ := schema["enum"].([]any)
			var these []string
			for _, e := range enum {
				these = append(these, fmt.Sprint(e))
			}
			thisDef := fmt.Sprint(schema["default"])
			if codes == nil {
				codes, def = these, thisDef
			} else if strings.Join(these, ",") != strings.Join(codes, ",") || thisDef != def {
				fail("spec: %s %s declares lang as %v (default %s), other operations declare %v (default %s); one note cannot describe both", strings.ToUpper(op.verb), op.path, these, thisDef, codes, def)
			}
		}
		if hasLang {
			supported = append(supported, "`roxy."+pascal(d.segment)+"`")
		} else {
			englishOnly = append(englishOnly, "`roxy."+pascal(d.segment)+"`")
		}
	}
	if len(codes) == 0 {
		fail("spec: no operation declares a lang query parameter")
	}
	quoted := make([]string, len(codes))
	for i, c := range codes {
		quoted[i] = "`" + c + "`"
	}
	return fmt.Sprintf("<!-- BEGIN:LANGS -->\n**Multi-language responses.** Interpretations are available in %d languages: %s. Set `Lang` on the params struct of any supported endpoint with `roxyapi.Ptr(...)`; it defaults to `%s`. Supported: %s. English-only: %s.\n<!-- END:LANGS -->",
		len(codes), strings.Join(quoted, ", "), def, strings.Join(supported, ", "), strings.Join(englishOnly, ", "))
}

type operation struct {
	path, verb, opID, summary string
	params                    any // the raw `parameters` array of the operation
}

// langParam returns the `lang` query parameter of an operation, or nil.
func (op operation) langParam() map[string]any {
	list, _ := op.params.([]any)
	for _, p := range list {
		param, ok := p.(map[string]any)
		if ok && param["in"] == "query" && param["name"] == "lang" {
			return param
		}
	}
	return nil
}

type domainInfo struct {
	tag, segment, summary string
	ops                   []operation
}

// domainsInOrder walks the spec ONCE and is the single source of truth for both
// the facade grouping and every doc renderer: operations bucketed by URL first
// segment (sorted by path then verb for determinism, since JSON object order is
// not preserved), grouped into domains in canonical tag order. The accessor for a
// domain is the PascalCased first segment.
func domainsInOrder(spec map[string]any) []domainInfo {
	bySeg := map[string][]operation{}
	tagToSegment := map[string]string{}
	paths, _ := spec["paths"].(map[string]any)
	total := 0
	for path, item := range paths {
		pathItem, ok := item.(map[string]any)
		if !ok {
			continue
		}
		seg := firstSegment(path)
		for verb, val := range pathItem {
			if _, isVerb := httpVerbs[verb]; !isVerb {
				continue
			}
			op, ok := val.(map[string]any)
			if !ok {
				continue
			}
			id, _ := op["operationId"].(string)
			if id == "" {
				fail("spec: %s %s has no operationId", strings.ToUpper(verb), path)
			}
			summary, _ := op["summary"].(string)
			bySeg[seg] = append(bySeg[seg], operation{path: path, verb: verb, opID: id, summary: summary, params: op["parameters"]})
			total++
			tag := firstTag(op)
			if tag == "" {
				fail("spec: %s %s has no tag, so it belongs to no domain", strings.ToUpper(verb), path)
			}
			if prev, exists := tagToSegment[tag]; exists && prev != seg {
				fail("spec: tag %q spans path segments %s and %s; a domain accessor needs exactly one", tag, prev, seg)
			}
			tagToSegment[tag] = seg
		}
	}
	for seg := range bySeg {
		sortOps(bySeg[seg])
	}
	var out []domainInfo
	seen := map[string]bool{}
	covered := 0
	for _, t := range tagList(spec) {
		name := t["name"].(string)
		seg, ok := tagToSegment[name]
		if !ok || seen[seg] {
			continue
		}
		seen[seg] = true
		covered += len(bySeg[seg])
		out = append(out, domainInfo{tag: name, segment: seg, summary: tagSummary(t), ops: bySeg[seg]})
	}
	if covered != total {
		fail("spec: %d of %d operations carry a tag that is missing from the top-level tags list, so they would vanish from the facade and the docs", total-covered, total)
	}
	return out
}

// sortOps orders operations by path then verb for deterministic output.
func sortOps(ops []operation) {
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].path != ops[j].path {
			return ops[i].path < ops[j].path
		}
		return httpVerbs[ops[i].verb] < httpVerbs[ops[j].verb]
	})
}

// segmentByOpKey maps the normalized id of each operation to its URL segment, so the
// facade matches a generated method to its domain regardless of initialism casing.
func segmentByOpKey(domains []domainInfo) map[string]string {
	m := map[string]string{}
	for _, d := range domains {
		for _, op := range d.ops {
			key := normalizeKey(op.opID)
			if _, dup := m[key]; dup {
				fail("spec: operationId %s collides with another once casing is ignored", op.opID)
			}
			m[key] = d.segment
		}
	}
	return m
}

func renderDomainsTable(domains []domainInfo) string {
	var b strings.Builder
	b.WriteString("<!-- BEGIN:DOMAINS -->\n")
	b.WriteString("| Accessor | What it covers |\n")
	b.WriteString("|----------|----------------|\n")
	for _, d := range domains {
		b.WriteString(fmt.Sprintf("| `roxy.%s` | %s |\n", pascal(d.segment), d.summary))
	}
	b.WriteString("<!-- END:DOMAINS -->")
	return b.String()
}

func renderMethods(domains []domainInfo) string {
	var b strings.Builder
	b.WriteString("<!-- BEGIN:METHODS -->\n")
	for i, d := range domains {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("## %s - `roxy.%s`\n\n", d.tag, pascal(d.segment)))
		if d.summary != "" {
			b.WriteString(d.summary + "\n\n")
		}
		for _, op := range d.ops {
			line := fmt.Sprintf("- `%s` - %s `%s`", op.opID, strings.ToUpper(op.verb), op.path)
			if op.summary != "" {
				line += " - " + op.summary
			}
			b.WriteString(line + "\n")
		}
	}
	b.WriteString("<!-- END:METHODS -->")
	return b.String()
}

// ─── shared helpers ──────────────────────────────────────────────────────────

func tagList(spec map[string]any) []map[string]any {
	raw, _ := spec["tags"].([]any)
	var out []map[string]any
	for _, t := range raw {
		if m, ok := t.(map[string]any); ok {
			if _, ok := m["name"].(string); ok {
				out = append(out, m)
			}
		}
	}
	return out
}

func firstTag(op map[string]any) string {
	tags, _ := op["tags"].([]any)
	if len(tags) == 0 {
		return ""
	}
	s, _ := tags[0].(string)
	return s
}

func tagSummary(tag map[string]any) string {
	desc, _ := tag["description"].(string)
	if desc == "" {
		desc, _ = tag["name"].(string)
	}
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return ""
	}
	if i := strings.Index(desc, ". "); i >= 0 {
		desc = desc[:i]
	}
	desc = strings.TrimRight(strings.TrimSpace(desc), ".")
	desc = strings.Join(strings.Fields(desc), " ")
	if len(desc) > 120 {
		return strings.TrimSpace(desc[:117]) + "..."
	}
	return desc
}

func firstSegment(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return parts[0]
}

// normalizeKey lowercases and strips non-alphanumerics, so an operationId and its
// generated Go method name compare equal regardless of initialism casing.
func normalizeKey(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func pascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == '.' || r == '/'
	})
	var b strings.Builder
	for _, p := range parts {
		runes := []rune(p)
		b.WriteRune(unicode.ToUpper(runes[0]))
		b.WriteString(string(runes[1:]))
	}
	return b.String()
}

// swap replaces the text between `<!-- BEGIN:<region> -->` and its END marker in a
// doc with block (which carries both markers) and reports whether the file changed.
// The file and both markers must exist: every doc here is tracked, so a missing
// one is a broken checkout, never a doc that has not been authored yet.
func swap(path, region, block string) bool {
	begin, end := "<!-- BEGIN:"+region+" -->", "<!-- END:"+region+" -->"
	src, err := os.ReadFile(path)
	check(err, "sync-docs: read "+path)
	text := string(src)
	b := strings.Index(text, begin)
	e := strings.Index(text, end)
	if b < 0 || e < 0 || e < b {
		fail("sync-docs: %s is missing %s / %s markers", path, begin, end)
	}
	next := text[:b] + block + text[e+len(end):]
	if next == text {
		return false
	}
	check(os.WriteFile(path, []byte(next), 0o644), "write "+path)
	return true
}
