// SPDX-License-Identifier: BSD-3-Clause
// Horvium's M1 foreground commands are development interfaces, not installation.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gonz0w/horvium/internal/admin"
	"github.com/gonz0w/horvium/internal/client"
	"github.com/gonz0w/horvium/internal/observation"
	"github.com/gonz0w/horvium/internal/service"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var errInput = errors.New("invalid command or protected input")

func readToken(r io.Reader) (string, error) {
	b, e := io.ReadAll(io.LimitReader(r, 66))
	if e != nil {
		return "", errInput
	}
	s := strings.TrimSuffix(string(b), "\n")
	if len(s) != 64 {
		return "", errInput
	}
	for _, v := range s {
		if !(v >= '0' && v <= '9' || v >= 'a' && v <= 'f') {
			return "", errInput
		}
	}
	return s, nil
}
func protectedInput(f *os.File) bool {
	st, e := f.Stat()
	if e != nil {
		return false
	}
	if st.Mode()&os.ModeNamedPipe != 0 {
		return true
	}
	if !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 {
		return false
	}
	return ownedInput(st)
}
func run(ctx context.Context, args []string, input io.Reader, out, errout io.Writer) int {
	fail := func(e error) int { _, _ = fmt.Fprintln(errout, e.Error()); return 1 }
	if len(args) == 0 {
		return fail(errInput)
	}
	flags := flag.NewFlagSet("horvium", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	switch args[0] {
	case "inventory":
		endpoint := flags.String("endpoint", "http://127.0.0.1:8088", "explicit loopback management endpoint")
		kind := flags.String("resource", "overview", "overview, pools, datasets or providers")
		generation := flags.String("generation", "", "exact observation cursor")
		filter := flags.String("filter", "", "literal name filter")
		sort := flags.String("sort", "", "resource sort")
		offset := flags.Int("offset", 0, "page offset")
		limit := flags.Int("limit", 0, "page limit")
		descending := flags.Bool("descending", false, "descending sort")
		if flags.Parse(args[1:]) != nil || flags.NArg() != 0 {
			return fail(errInput)
		}
		if f, ok := input.(*os.File); ok && !protectedInput(f) {
			return fail(errInput)
		}
		if f, ok := input.(*os.File); ok {
			stop := context.AfterFunc(ctx, func() { _ = f.Close() })
			defer stop()
		}
		token, e := readToken(input)
		if e != nil {
			return fail(e)
		}
		c, e := client.New(*endpoint)
		if e != nil {
			return fail(e)
		}
		if e = c.Login(ctx, token); e != nil {
			return fail(e)
		}
		defer func() {
			logout, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = c.Logout(logout)
		}()
		value, e := c.Query(ctx, *kind, observation.Query{Generation: *generation, Filter: *filter, Sort: *sort, Offset: *offset, Limit: *limit, Descending: *descending})
		if e != nil {
			return fail(e)
		}
		if json.NewEncoder(out).Encode(value) != nil {
			return fail(errors.New("output unavailable"))
		}
		return 0
	case "serve", "helper", "token-setup", "token-rotate":
		path := flags.String("config", "", "absolute root-owned configuration file")
		if flags.Parse(args[1:]) != nil || flags.NArg() != 0 || !filepath.IsAbs(*path) {
			return fail(errInput)
		}
		c, e := admin.LoadConfig(*path)
		if e != nil {
			return fail(e)
		}
		switch args[0] {
		case "serve":
			e = service.Serve(ctx, c)
		case "helper":
			e = service.Helper(ctx, c)
		default:
			var token string
			token, e = admin.SetupToken(c, args[0] == "token-rotate")
			if e == nil {
				_, e = fmt.Fprintln(out, token)
			}
		}
		if e != nil {
			return fail(e)
		}
		return 0
	default:
		return fail(errInput)
	}
}
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	os.Exit(run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
