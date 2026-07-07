// Command roxygen is the RoxyAPI Go SDK code generator. The OpenAPI spec at
// roxyapi.com is the single source of truth; everything below regenerates from it.
//
//	roxygen generate    Fetch the spec, normalize it for oapi-codegen, generate the
//	                    typed client (roxyapi.gen.go), generate the domain-grouped
//	                    facade (roxy.gen.go), then sync the spec-derived docs.
//	roxygen sync-docs   Regenerate only the spec-derived regions of README.md,
//	                    AGENTS.md, and docs/llms-full.txt (between BEGIN/END markers).
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
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/util"
)

const (
	specURL           = "https://roxyapi.com/api/v2/openapi.json"
	absoluteServerURL = "https://roxyapi.com/api/v2"
	specPath          = "specs/openapi.json"
	clientPath        = "roxyapi.gen.go"
	facadePath        = "roxy.gen.go"
)

// errorStatusCodes are the codes the API returns with a `{ error, code }` body.
// The served spec inlines that shape per operation; we repoint them all at one
// shared ErrorResponse schema so the generator emits a single error type.
var errorStatusCodes = []string{"400", "401", "404", "405", "429", "500"}

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
	fmt.Printf("Fetching OpenAPI spec from %s\n", specURL)
	raw := fetchSpec()

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var spec map[string]any
	check(dec.Decode(&spec), "parse spec")

	guardOpenAPI31(spec)
	spec["openapi"] = "3.0.3"
	patchServerURL(spec)
	patchPathParameters(spec)
	simplifyFreeFormAdditionalProps(spec)
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

// guardOpenAPI31 fails loudly if the spec uses an OpenAPI 3.1-only construct that
// the 3.0-targeting code generator cannot consume. Today the spec declares 3.1 but
// uses only 3.0 features, so a version-string downconvert is enough; if that ever
// changes this guard surfaces it instead of emitting silently-wrong code.
func guardOpenAPI31(node any) {
	switch n := node.(type) {
	case map[string]any:
		if t, ok := n["type"]; ok {
			if _, isArray := t.([]any); isArray {
				fail("spec uses OpenAPI 3.1 type-array (nullable union); add a real 3.1->3.0 downconverter to roxygen")
			}
		}
		for _, k := range []string{"const", "prefixItems"} {
			if _, ok := n[k]; ok {
				fail("spec uses OpenAPI 3.1-only keyword %q; add a real 3.1->3.0 downconverter to roxygen", k)
			}
		}
		for _, v := range n {
			guardOpenAPI31(v)
		}
	case []any:
		for _, v := range n {
			guardOpenAPI31(v)
		}
	}
}

// simplifyFreeFormAdditionalProps collapses a free-form `additionalProperties`
// schema (one carrying no structural keyword, e.g. `{ nullable: true }`) to the
// boolean `true`. The generator otherwise emits a `map[string]*interface{}` field
// whose own Get/Set/UnmarshalJSON helpers use `map[string]interface{}`, which does
// not compile. The boolean form means the same thing (any extra properties allowed)
// and generates a consistent `map[string]interface{}`.
func simplifyFreeFormAdditionalProps(node any) {
	switch n := node.(type) {
	case map[string]any:
		if ap, ok := n["additionalProperties"].(map[string]any); ok && isFreeForm(ap) {
			n["additionalProperties"] = true
		}
		for _, v := range n {
			simplifyFreeFormAdditionalProps(v)
		}
	case []any:
		for _, v := range n {
			simplifyFreeFormAdditionalProps(v)
		}
	}
}

func isFreeForm(schema map[string]any) bool {
	for _, k := range []string{"type", "$ref", "properties", "items", "allOf", "oneOf", "anyOf", "additionalProperties", "enum", "format"} {
		if _, ok := schema[k]; ok {
			return false
		}
	}
	return true
}

// patchServerURL rewrites the relative production server (`/api/v2`) to an absolute
// URL so the generated client targets production out of the box.
func patchServerURL(spec map[string]any) {
	servers, _ := spec["servers"].([]any)
	if len(servers) == 0 {
		return
	}
	if s0, ok := servers[0].(map[string]any); ok {
		s0["url"] = absoluteServerURL
	}
}

// patchPathParameters forces required:true on every path parameter. OpenAPI requires
// it and the generator rejects an optional path parameter; at least one upstream route
// omits it.
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

// normalizeErrors adds one shared ErrorResponse schema and repoints every error
// response at it, so the generator emits a single typed error body (mapped to
// *RoxyError in errors.go) instead of a distinct inline struct per operation.
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
	for _, item := range paths {
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
			for _, code := range errorStatusCodes {
				resp, ok := responses[code].(map[string]any)
				if !ok {
					continue
				}
				if _, isRef := resp["$ref"]; isRef {
					continue
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
// and groups them by URL first segment, matching the spec's operations.
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
		if !ok || fn.Recv == nil || !isClientWithResponsesRecv(fn.Recv) {
			continue
		}
		name := fn.Name.Name
		if !strings.HasSuffix(name, "WithResponse") || strings.HasSuffix(name, "WithBodyWithResponse") {
			continue
		}
		opName := strings.TrimSuffix(name, "WithResponse")
		seg, ok := segByOpKey[normalizeKey(opName)]
		if !ok {
			fmt.Printf("warning: no spec segment for generated method %s; skipping\n", name)
			continue
		}
		methodsBySeg[seg] = append(methodsBySeg[seg], fn)
		collectSelectorPkgs(fn.Type, used)
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
	total := 0
	for _, m := range methodsBySeg {
		total += len(m)
	}
	fmt.Printf("Facade written to %s (%d methods across %d domains).\n", facadePath, total, len(methodsBySeg))
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

func isClientWithResponsesRecv(recv *ast.FieldList) bool {
	if len(recv.List) != 1 {
		return false
	}
	star, ok := recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	id, ok := star.X.(*ast.Ident)
	return ok && id.Name == "ClientWithResponses"
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
	changed := false
	changed = replaceRegion("README.md", table) || changed
	changed = replaceRegion("AGENTS.md", table) || changed
	changed = replaceRegionIn("docs/llms-full.txt", renderMethods(domains)) || changed

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

type operation struct {
	path, verb, opID, summary string
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
			summary, _ := op["summary"].(string)
			bySeg[seg] = append(bySeg[seg], operation{path: path, verb: verb, opID: id, summary: summary})
			if tag := firstTag(op); tag != "" {
				if _, exists := tagToSegment[tag]; !exists {
					tagToSegment[tag] = seg
				}
			}
		}
	}
	for seg := range bySeg {
		sortOps(bySeg[seg])
	}
	var out []domainInfo
	seen := map[string]bool{}
	for _, t := range tagList(spec) {
		name := t["name"].(string)
		seg, ok := tagToSegment[name]
		if !ok || seen[seg] {
			continue
		}
		seen[seg] = true
		out = append(out, domainInfo{tag: name, segment: seg, summary: tagSummary(t), ops: bySeg[seg]})
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

// segmentByOpKey maps each operation's normalized id to its URL segment, so the
// facade matches a generated method to its domain regardless of initialism casing.
func segmentByOpKey(domains []domainInfo) map[string]string {
	m := map[string]string{}
	for _, d := range domains {
		for _, op := range d.ops {
			m[normalizeKey(op.opID)] = d.segment
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

// replaceRegion swaps the text between the DOMAINS markers in a markdown file.
// A missing file is skipped (markers are added when the doc is authored).
func replaceRegion(path, block string) bool {
	return swap(path, "<!-- BEGIN:DOMAINS -->", "<!-- END:DOMAINS -->", block)
}

func replaceRegionIn(path, block string) bool {
	return swap(path, "<!-- BEGIN:METHODS -->", "<!-- END:METHODS -->", block)
}

func swap(path, begin, end, block string) bool {
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("sync-docs: %s not present yet, skipping.\n", path)
		return false
	}
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
