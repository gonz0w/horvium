// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"embed"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// Bind the actual platform-only fixture source into the portable regression:
// importing it or executing a native child is neither necessary nor allowed.
//
//go:embed executor_recovery_freebsd_test.go supervisor_native_freebsd_test.go guard_freebsd_test.go
var nativeFixtureSources embed.FS

func TestFixtureProcessMetadataCannotGrantSignalAuthority(t *testing.T) {
	for _, name := range []string{"executor_recovery_freebsd_test.go", "supervisor_native_freebsd_test.go", "guard_freebsd_test.go"} {
		data, err := nativeFixtureSources.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		positions := token.NewFileSet()
		file, err := parser.ParseFile(positions, name, data, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if ok && (((pkg.Name == "syscall" || pkg.Name == "unix") && selector.Sel.Name == "Kill") ||
				(pkg.Name == "os" && selector.Sel.Name == "FindProcess")) {
				t.Errorf("%s: observed process IDs cannot grant signal authority", positions.Position(call.Pos()))
			}
			return true
		})
	}
}
