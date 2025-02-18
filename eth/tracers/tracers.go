// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package tracers is a collection of JavaScript transaction tracers.
package tracers

import (
	"io/fs"
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/eth/tracers/internal/tracers"
)

// camel converts a snake cased input string into a camel cased output.
func camel(str string) string {
	pieces := strings.Split(str, "_")
	for i := 1; i < len(pieces); i++ {
		pieces[i] = string(unicode.ToUpper(rune(pieces[i][0]))) + pieces[i][1:]
	}
	return strings.Join(pieces, "")
}

var assetTracers = make(map[string]string)

// init retrieves the JavaScript transaction tracers included in go-ethereum.
func init() {
	err := fs.WalkDir(tracers.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := fs.ReadFile(tracers.FS, path)
		if err != nil {
			return err
		}
		name := camel(strings.TrimSuffix(path, ".js"))
		assetTracers[name] = string(b)
		return nil
	})
	if err != nil {
		panic(err)
	}
}

// tracer retrieves a specific JavaScript tracer by name.
func tracer(name string) (string, bool) {
	if tracer, ok := assetTracers[name]; ok {
		return tracer, true
	}
	return "", false
}
