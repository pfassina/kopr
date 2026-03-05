package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMouseMsgToBytes(t *testing.T) {
	tests := []struct {
		name       string
		msg        tea.MouseMsg
		col        int
		row        int
		lastButton tea.MouseButton
		want       string
	}{
		{
			name:       "left press at origin",
			msg:        tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},
			col:        0,
			row:        0,
			lastButton: tea.MouseButtonNone,
			want:       "\x1b[<0;1;1M",
		},
		{
			name:       "left release at 5,10",
			msg:        tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonNone},
			col:        5,
			row:        10,
			lastButton: tea.MouseButtonLeft,
			want:       "\x1b[<0;6;11m",
		},
		{
			name:       "right press at 3,7",
			msg:        tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonRight},
			col:        3,
			row:        7,
			lastButton: tea.MouseButtonNone,
			want:       "\x1b[<2;4;8M",
		},
		{
			name:       "left drag at 10,5",
			msg:        tea.MouseMsg{Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft},
			col:        10,
			row:        5,
			lastButton: tea.MouseButtonLeft,
			want:       "\x1b[<32;11;6M",
		},
		{
			name:       "scroll up",
			msg:        tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp},
			col:        0,
			row:        0,
			lastButton: tea.MouseButtonNone,
			want:       "\x1b[<64;1;1M",
		},
		{
			name:       "scroll down",
			msg:        tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown},
			col:        2,
			row:        3,
			lastButton: tea.MouseButtonNone,
			want:       "\x1b[<65;3;4M",
		},
		{
			name: "ctrl+left press",
			msg: tea.MouseMsg{
				Action: tea.MouseActionPress,
				Button: tea.MouseButtonLeft,
				Ctrl:   true,
			},
			col:        0,
			row:        0,
			lastButton: tea.MouseButtonNone,
			want:       "\x1b[<16;1;1M",
		},
		{
			name:       "middle press",
			msg:        tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonMiddle},
			col:        1,
			row:        1,
			lastButton: tea.MouseButtonNone,
			want:       "\x1b[<1;2;2M",
		},
		{
			name:       "right release uses lastButton",
			msg:        tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonNone},
			col:        4,
			row:        6,
			lastButton: tea.MouseButtonRight,
			want:       "\x1b[<2;5;7m",
		},
		{
			name:       "unsupported button returns nil",
			msg:        tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonBackward},
			col:        0,
			row:        0,
			lastButton: tea.MouseButtonNone,
			want:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mouseMsgToBytes(tt.msg, tt.col, tt.row, tt.lastButton)
			if tt.want == "" {
				if got != nil {
					t.Errorf("got %q, want nil", string(got))
				}
				return
			}
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", string(got), tt.want)
			}
		})
	}
}
