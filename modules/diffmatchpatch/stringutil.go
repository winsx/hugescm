// Copyright (c) 2012-2016 The go-diff authors. All rights reserved.
// https://github.com/sergi/go-diff
// See the included LICENSE file for license details.
//
// go-diff is a Go implementation of Google's Diff, Match, and Patch library
// Original library is Copyright (c) 2006 Google Inc.
// http://code.google.com/p/google-diff-match-patch/

package diffmatchpatch

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

const UNICODE_INVALID_RANGE_START = 0xD800
const UNICODE_INVALID_RANGE_END = 0xDFFF
const UNICODE_INVALID_RANGE_DELTA = UNICODE_INVALID_RANGE_END - UNICODE_INVALID_RANGE_START + 1
const UNICODE_RANGE_MAX = 0x10FFFF

// unescaper unescapes selected chars for compatibility with JavaScript's encodeURI.
// In speed critical applications this could be dropped since the receiving application will certainly decode these fine. Note that this function is case-sensitive.  Thus "%3F" would not be unescaped.  But this is ok because it is only called with the output of HttpUtility.UrlEncode which returns lowercase hex. Example: "%3f" -> "?", "%24" -> "$", etc.
var unescaper = strings.NewReplacer(
	"%21", "!", "%7E", "~", "%27", "'",
	"%28", "(", "%29", ")", "%3B", ";",
	"%2F", "/", "%3F", "?", "%3A", ":",
	"%40", "@", "%26", "&", "%3D", "=",
	"%2B", "+", "%24", "$", "%2C", ",", "%23", "#", "%2A", "*")

// indexOf returns the first index of pattern in str, starting at str[i].
func indexOf(str string, pattern string, i int) int {
	if i > len(str)-1 {
		return -1
	}
	if i <= 0 {
		return strings.Index(str, pattern)
	}
	ind := strings.Index(str[i:], pattern)
	if ind == -1 {
		return -1
	}
	return ind + i
}

// lastIndexOf returns the last index of pattern in str, starting at str[i].
func lastIndexOf(str string, pattern string, i int) int {
	if i < 0 {
		return -1
	}
	if i >= len(str) {
		return strings.LastIndex(str, pattern)
	}
	_, size := utf8.DecodeRuneInString(str[i:])
	return strings.LastIndex(str[:i+size], pattern)
}

// runesIndexOf returns the index of pattern in target, starting at target[i].
func runesIndexOf(target, pattern []rune, i int) int {
	if i > len(target)-1 {
		return -1
	}
	if i <= 0 {
		return runesIndex(target, pattern)
	}
	ind := runesIndex(target[i:], pattern)
	if ind == -1 {
		return -1
	}
	return ind + i
}

// runesIndex is the equivalent of strings.Index for rune slices.
func runesIndex(r1, r2 []rune) int {
	last := len(r1) - len(r2)
	for i := 0; i <= last; i++ {
		if slices.Equal(r1[i:i+len(r2)], r2) {
			return i
		}
	}
	return -1
}

// These constants define the number of bits representable
// in 1,2,3,4 byte utf8 sequences, respectively.
const ONE_BYTE_BITS = 7
const TWO_BYTE_BITS = 11
const THREE_BYTE_BITS = 16
const FOUR_BYTE_BITS = 21

// Helper for getting a sequence of bits from an integer.
func getBits(i uint32, cnt byte, from byte) byte {
	return byte((i >> from) & ((1 << cnt) - 1))
}

// Converts an integer in the range 0~1112060 into a rune.
// Based on the ranges table in https://en.wikipedia.org/wiki/UTF-8
func intToRune(i uint32) rune {
	if i < (1 << ONE_BYTE_BITS) {
		return rune(i)
	}

	if i < (1 << TWO_BYTE_BITS) {
		r, size := utf8.DecodeRune([]byte{0b11000000 | getBits(i, 5, 6), 0b10000000 | getBits(i, 6, 0)})
		if size != 2 || r == utf8.RuneError {
			panic(fmt.Sprintf("Error encoding an int %d with size 2, got rune %v and size %d", size, r, i))
		}
		return r
	}

	// Last -3 here needed because for some reason 3rd to last codepoint 65533 in this range
	// was returning utf8.RuneError during encoding.
	if i < ((1 << THREE_BYTE_BITS) - UNICODE_INVALID_RANGE_DELTA - 3) {
		if i >= UNICODE_INVALID_RANGE_START {
			i += UNICODE_INVALID_RANGE_DELTA
		}

		r, size := utf8.DecodeRune([]byte{0b11100000 | getBits(i, 4, 12), 0b10000000 | getBits(i, 6, 6), 0b10000000 | getBits(i, 6, 0)})
		if size != 3 || r == utf8.RuneError {
			panic(fmt.Sprintf("Error encoding an int %d with size 3, got rune %v and size %d", size, r, i))
		}
		return r
	}

	if i < (1<<FOUR_BYTE_BITS - UNICODE_INVALID_RANGE_DELTA - 3) {
		i += UNICODE_INVALID_RANGE_DELTA + 3
		r, size := utf8.DecodeRune([]byte{0b11110000 | getBits(i, 3, 18), 0b10000000 | getBits(i, 6, 12), 0b10000000 | getBits(i, 6, 6), 0b10000000 | getBits(i, 6, 0)})
		if size != 4 || r == utf8.RuneError {
			panic(fmt.Sprintf("Error encoding an int %d with size 4, got rune %v and size %d", size, r, i))
		}
		return r
	}
	panic(fmt.Sprintf("The integer %d is too large for runeToInt()", i))
}

// Converts a rune generated by intToRune back to an integer
func runeToInt(r rune) uint32 {
	i := uint32(r)
	if i < (1 << ONE_BYTE_BITS) {
		return i
	}

	bytes := []byte{0, 0, 0, 0}

	size := utf8.EncodeRune(bytes, r)

	if size == 2 {
		return uint32(bytes[0]&0b11111)<<6 | uint32(bytes[1]&0b111111)
	}

	if size == 3 {
		result := uint32(bytes[0]&0b1111)<<12 | uint32(bytes[1]&0b111111)<<6 | uint32(bytes[2]&0b111111)
		if result >= UNICODE_INVALID_RANGE_END {
			return result - UNICODE_INVALID_RANGE_DELTA
		}

		return result
	}

	if size == 4 {
		result := uint32(bytes[0]&0b111)<<18 | uint32(bytes[1]&0b111111)<<12 | uint32(bytes[2]&0b111111)<<6 | uint32(bytes[3]&0b111111)
		return result - UNICODE_INVALID_RANGE_DELTA - 3
	}

	panic(fmt.Sprintf("Unexpected state decoding rune=%v size=%d", r, size))
}
