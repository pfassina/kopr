package editor

import (
	"fmt"
	"sync"
	"time"

	"github.com/neovim/go-client/nvim"
)

// NvimMode represents Neovim's current mode.
type NvimMode string

const (
	ModeNormal  NvimMode = "n"
	ModeInsert  NvimMode = "i"
	ModeVisual  NvimMode = "v"
	ModeVisLine NvimMode = "V"
	ModeVisBlk  NvimMode = "\x16"
	ModeCommand NvimMode = "c"
	ModeReplace NvimMode = "R"
	ModeTermnl  NvimMode = "t"
)

// RPC manages the Neovim RPC connection.
type RPC struct {
	client *nvim.Nvim
	mu     sync.RWMutex
	mode   NvimMode
	onMode func(NvimMode) // callback when mode changes
}

// ConnectRPC dials the Neovim socket and sets up event subscriptions.
// It retries briefly since Neovim may not have the socket ready immediately.
func ConnectRPC(socketPath string, onMode func(NvimMode)) (*RPC, error) {
	var client *nvim.Nvim
	var err error

	for i := 0; i < 50; i++ {
		client, err = nvim.Dial(socketPath)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return nil, fmt.Errorf("connect to nvim socket: %w", err)
	}

	rpc := &RPC{
		client: client,
		mode:   ModeNormal,
		onMode: onMode,
	}

	if err := rpc.setupModeChanged(); err != nil {
		client.Close()
		return nil, fmt.Errorf("setup mode events: %w", err)
	}

	return rpc, nil
}

func (r *RPC) setupModeChanged() error {
	r.client.RegisterHandler("mode_changed", func(args ...interface{}) {
		if len(args) < 2 {
			return
		}
		newMode, ok := args[1].(string)
		if !ok {
			return
		}

		r.mu.Lock()
		r.mode = NvimMode(newMode)
		r.mu.Unlock()

		if r.onMode != nil {
			r.onMode(NvimMode(newMode))
		}
	})

	if err := r.client.Subscribe("mode_changed"); err != nil {
		return err
	}

	cid := r.client.ChannelID()
	_, err := r.client.Exec(fmt.Sprintf(`
		augroup VimVaultMode
			autocmd!
			autocmd ModeChanged * call rpcnotify(%d, 'mode_changed', v:event.old_mode, v:event.new_mode)
		augroup END
	`, cid), false)
	return err
}

// Mode returns the current Neovim mode.
func (r *RPC) Mode() NvimMode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mode
}

// OpenFile opens a file in Neovim.
func (r *RPC) OpenFile(path string) error {
	return r.client.Command("edit " + path)
}

// CurrentFile returns the current buffer's file path.
func (r *RPC) CurrentFile() (string, error) {
	buf, err := r.client.CurrentBuffer()
	if err != nil {
		return "", err
	}
	return r.client.BufferName(buf)
}

// BufferContent returns all lines of the current buffer.
func (r *RPC) BufferContent() ([][]byte, error) {
	buf, err := r.client.CurrentBuffer()
	if err != nil {
		return nil, err
	}
	return r.client.BufferLines(buf, 0, -1, false)
}

// ExecCommand runs an Ex command in Neovim.
func (r *RPC) ExecCommand(cmd string) error {
	return r.client.Command(cmd)
}

// ExecLua runs Lua code in Neovim.
func (r *RPC) ExecLua(code string, result interface{}, args ...interface{}) error {
	return r.client.ExecLua(code, result, args...)
}

// FormatBuffer formats the current buffer using Neovim's built-in formatter.
func (r *RPC) FormatBuffer() error {
	return r.client.Command("normal! gg=G``")
}

// InsertText inserts text at the current cursor position.
func (r *RPC) InsertText(text string) error {
	_, err := r.client.Input("i" + text)
	return err
}

// Close closes the RPC connection.
func (r *RPC) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}
