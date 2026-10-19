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
//go:embed executor_recovery_freebsd_test.go supervisor_native_freebsd_test.go guard_freebsd_test.go stable_fixture_freebsd_test.go
var nativeFixtureSources embed.FS

func TestFixtureProcessMetadataCannotGrantSignalAuthority(t *testing.T) {
	for _, name := range []string{"executor_recovery_freebsd_test.go", "supervisor_native_freebsd_test.go", "guard_freebsd_test.go", "stable_fixture_freebsd_test.go"} {
		data, err := nativeFixtureSources.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		positions := token.NewFileSet()
		file, err := parser.ParseFile(positions, name, data, 0)
		if err != nil {
			t.Fatal(err)
		}
		var reserved []*ast.FuncLit
		ast.Inspect(file, func(node ast.Node) bool {
			c, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			typ, ok := c.Type.(*ast.Ident)
			if !ok || typ.Name != "fixtureReservation" {
				return true
			}
			for _, entry := range c.Elts {
				kv, ok := entry.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok || key.Name != "positive" {
					continue
				}
				if f, ok := kv.Value.(*ast.FuncLit); ok {
					reserved = append(reserved, f)
				}
			}
			return true
		})
		allowed := 0
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
				if name == "stable_fixture_freebsd_test.go" && pkg.Name == "unix" && selector.Sel.Name == "Kill" {
					for _, f := range reserved {
						if call.Pos() > f.Pos() && call.End() < f.End() {
							allowed++
							return true
						}
					}
				}
				t.Errorf("%s: observed process IDs cannot grant signal authority", positions.Position(call.Pos()))
			}
			return true
		})
		if name == "stable_fixture_freebsd_test.go" && (allowed != 1 || len(reserved) != 1) {
			t.Fatal("one explicitly reserved original guard signal route required")
		}
	}
}

//go:embed execute.go
var fixtureExecutorSource string

// The notification may hold a private fixture handshake. Every born handle and
// sole finalizer must already be retained before that test callback can run.
func TestFixtureBirthNotificationFollowsOwnedFinalizers(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "execute.go", fixtureExecutorSource, 0)
	if err != nil {
		t.Fatal(err)
	}
	var start, bound, observe, readersEnd, notify, returned token.Pos
	sel := func(e ast.Expr, x, y string) bool {
		s, ok := e.(*ast.SelectorExpr)
		if !ok || s.Sel.Name != y {
			return false
		}
		i, ok := s.X.(*ast.Ident)
		return ok && i.Name == x
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if sel(x.Fun, "cmd", "Start") {
				start = x.Pos()
			}
			if sel(x.Fun, "control", "started") {
				notify = x.Pos()
			}
		case *ast.AssignStmt:
			for _, lhs := range x.Lhs {
				if sel(lhs, "calls", "wait") {
					bound = x.Pos()
				}
				if sel(lhs, "calls", "observe") {
					observe = x.Pos()
				}
			}
		case *ast.RangeStmt:
			ast.Inspect(x.Body, func(n ast.Node) bool {
				a, ok := n.(*ast.AssignStmt)
				if ok {
					for _, lhs := range a.Lhs {
						if sel(lhs, "calls", "readers") {
							readersEnd = x.End()
						}
					}
				}
				return true
			})
		case *ast.ReturnStmt:
			if notify > 0 && x.Pos() > notify && len(x.Results) == 2 {
				if i, ok := x.Results[0].(*ast.Ident); ok && i.Name == "calls" {
					returned = x.Pos()
				}
			}
		}
		return true
	})
	if !(start > 0 && start < bound && bound < observe && observe < readersEnd && readersEnd < notify && notify < returned) {
		t.Fatal("born notification crossed original custody/finalizer binding", start, bound, observe, readersEnd, notify, returned)
	}
}
