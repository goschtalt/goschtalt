// SPDX-FileCopyrightText: 2014 https://github.com/fatih
// SPDX-License-Identifier: MIT
//
// This file originated from https://github.com/fatih/structs/blob/878a968ab22548362a09bdb3322f98b00f470d46/tags_test.go

package structs

import "testing"

func TestParseTag_Name(t *testing.T) {
	tags := []struct {
		tag string
		has bool
	}{
		{"", false},
		{"name", true},
		{"name,opt", true},
		{"name , opt, opt2", false}, // has a single whitespace
		{", opt, opt2", false},
	}

	for _, tag := range tags {
		name, _ := parseTag(tag.tag)

		if (name != "name") && tag.has {
			t.Errorf("Parse tag should return name: %#v", tag)
		}
	}
}

func TestParseTag_Opts(t *testing.T) {
	tags := []struct {
		opts string
		has  bool
	}{
		{"name", false},
		{"name,opt", true},
		{"name , opt, opt2", false}, // has a single whitespace
		{",opt, opt2", true},
		{", opt3, opt4", false},
	}

	// search for "opt"
	for _, tag := range tags {
		_, opts := parseTag(tag.opts)

		if opts.Has("opt") != tag.has {
			t.Errorf("Tag opts should have opt: %#v", tag)
		}
	}
}
