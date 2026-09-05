package main

import (
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestCommandDispatch(t *testing.T) {
	tests := []struct {
		name string
		args []string
		mode string
		file string
		dark bool
		fail bool
	}{
		{name: "background", args: []string{"view", "notes.md"}, mode: "background", file: "notes.md"},
		{name: "empty view", args: []string{"view"}, mode: "background"},
		{name: "dark", args: []string{"view", "--dark", "a b.md"}, mode: "background", file: "a b.md", dark: true},
		{name: "foreground", args: []string{"view", "a.md", "--foregruond", "--dark"}, mode: "desktop", file: "a.md", dark: true},
		{name: "false foreground", args: []string{"view", "--foregruond=false"}, mode: "background"},
		{name: "root", args: []string{}, mode: "desktop"},
		{name: "separator", args: []string{"view", "--", "--dark"}, mode: "background", file: "--dark"},
		{name: "help", args: []string{"view", "--help"}},
		{name: "unknown flag", args: []string{"view", "--unknown"}, fail: true},
		{name: "unrequested spelling", args: []string{"view", "--foreground"}, fail: true},
		{name: "extra file", args: []string{"view", "a", "b"}, fail: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			callback := func(mode string) func(string, bool) error {
				return func(file string, dark bool) error {
					calls = append(calls, mode)
					if file != tt.file || dark != tt.dark {
						t.Errorf("got file=%q dark=%v", file, dark)
					}
					return nil
				}
			}
			cmd := newRootCommand(callback("desktop"), callback("background"))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tt.args)
			if err := cmd.Execute(); (err != nil) != tt.fail {
				t.Fatalf("Execute error = %v, want failure %v", err, tt.fail)
			}
			var want []string
			if tt.mode != "" {
				want = []string{tt.mode}
			}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("calls = %v, want %v", calls, want)
			}
		})
	}
}

func TestCommandPropagatesLaunchErrors(t *testing.T) {
	want := errors.New("launch failed")
	fail := func(string, bool) error { return want }
	for _, args := range [][]string{{"view"}, {"view", "--foregruond"}, {}} {
		cmd := newRootCommand(fail, fail)
		cmd.SetArgs(args)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		if err := cmd.Execute(); !errors.Is(err, want) {
			t.Fatalf("args=%v: got %v", args, err)
		}
	}
}
