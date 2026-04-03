package main

import "testing"

func TestShouldAttachConsoleForCLI(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "no_args_gui_default_launch",
			args: []string{},
			want: false,
		},
		{
			name: "pure_gui_flag",
			args: []string{"--gui"},
			want: false,
		},
		{
			name: "pure_gui_flag_bool",
			args: []string{"--gui=true"},
			want: false,
		},
		{
			name: "help_flag",
			args: []string{"--help"},
			want: true,
		},
		{
			name: "version_command",
			args: []string{"version"},
			want: true,
		},
		{
			name: "pull_command",
			args: []string{"pull", "accounts"},
			want: true,
		},
		{
			name: "gui_plus_other_flags_is_cli_mode",
			args: []string{"--gui", "--help"},
			want: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := shouldAttachConsoleForCLI(tc.args)
			if got != tc.want {
				t.Fatalf("expected %t, got %t for args=%v", tc.want, got, tc.args)
			}
		})
	}
}
