package engine

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// msgFuncs are the functions whose first string argument (after an optional status) is a
// message key.
var msgFuncs = map[string]bool{"msg": true, "NewMsg": true, "invalid": true, "newError": true, "NewError": true}

// TestMessageKeysInCatalog checks that every message key the Go code emits has German and English
// text in web/src/i18n/messages.*.ts (or the UI catalog for shared vocabulary).
func TestMessageKeysInCatalog(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..")
	keys := map[string]string{} // key -> position

	for _, dir := range []string{"internal/engine", "internal/gameservice", "internal/api", "cmd/wasm"} {
		collectKeys(t, filepath.Join(root, dir), keys)
	}

	if len(keys) < 50 {
		t.Fatalf("found only %d message keys; is the scanner broken?", len(keys))
	}

	for _, lang := range []string{"de", "en"} {
		cat := catalogKeys(t, filepath.Join(root, "web/src/i18n/messages."+lang+".ts"))
		for k, v := range catalogKeys(t, filepath.Join(root, "web/src/i18n/ui."+lang+".ts")) {
			cat[k] = v
		}

		for k, pos := range keys {
			if !cat[k] {
				t.Errorf("%s: message key %q missing in messages.%s.ts", pos, k, lang)
			}
		}
	}
}

func collectKeys(t *testing.T, dir string, keys map[string]string) {
	t.Helper()

	fset := token.NewFileSet()

	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if !msgFuncs[funcName(n.Fun)] {
					return true
				}

				for _, a := range n.Args {
					if s, ok := stringLit(a); ok {
						addKey(fset, a, s, keys)
						break
					}
				}
			case *ast.KeyValueExpr:
				if id, ok := n.Key.(*ast.Ident); ok && (id.Name == "Key" || id.Name == "NameKey") {
					if s, ok := stringLit(n.Value); ok {
						addKey(fset, n.Value, s, keys)
					}
				}
			}

			return true
		})
	}
}

func addKey(fset *token.FileSet, n ast.Node, key string, keys map[string]string) {
	// Prefixes like "boatClass." are completed at runtime (ids from the data).
	if strings.HasSuffix(key, ".") || strings.HasPrefix(key, "data.") || key == "" {
		return
	}

	keys[key] = fset.Position(n.Pos()).String()
}

func funcName(e ast.Expr) string {
	switch f := e.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}

	return ""
}

func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}

	s, err := strconv.Unquote(lit.Value)

	return s, err == nil
}

var catalogKeyRe = regexp.MustCompile(`(?m)^\s*"([^"]+)":`)

func catalogKeys(t *testing.T, path string) map[string]bool {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	out := map[string]bool{}
	for _, m := range catalogKeyRe.FindAllStringSubmatch(string(raw), -1) {
		out[m[1]] = true
	}

	return out
}
