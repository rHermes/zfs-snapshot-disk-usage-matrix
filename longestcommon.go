/*
Copyright 2021 Teodor Spæren

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import "strings"

// Based on https://github.com/jpillora/longestcommon

// TrimPrefix removes the longest common prefix from all provided strings
func TrimPrefix(strs []string) []string {
	p := Prefix(strs)
	nstrs := make([]string, len(strs))
	for i, s := range strs {
		nstrs[i] = strings.TrimPrefix(s, p)
	}

	return nstrs
}

// Prefix returns the longest common prefix of the provided strings
func Prefix(strs []string) string {
	// short-circuit empty list
	if len(strs) == 0 {
		return ""
	}
	xfix := strs[0]
	// short-circuit single-element list
	if len(strs) == 1 {
		return xfix
	}
	// compare first to rest
	for _, str := range strs[1:] {
		xfixl := len(xfix)
		strl := len(str)
		// short-circuit empty strings
		if xfixl == 0 || strl == 0 {
			return ""
		}
		// maximum possible length
		maxl := xfixl
		if strl < maxl {
			maxl = strl
		}
		// compare letters
		// prefix, iterate left to right
		for i := 0; i < maxl; i++ {
			if xfix[i] != str[i] {
				xfix = xfix[:i]
				break
			}
		}
		if len(xfix) > maxl {
			xfix = xfix[:maxl]
		}
	}
	return xfix
}
