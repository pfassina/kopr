package editor

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
		augroup KoprMode
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
	return r.client.ExecLua("vim.cmd('edit ' .. vim.fn.fnameescape(...))", nil, path)
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

// SetupQuitSaveIntercept remaps quit/save commands in neovim to send
// RPC notifications instead of actually quitting. This keeps the app alive.
func (r *RPC) SetupQuitSaveIntercept(program *tea.Program) error {
	r.client.RegisterHandler("kopr:close-note", func(args ...interface{}) {
		save := false
		if len(args) > 0 {
			if b, ok := args[0].(bool); ok {
				save = b
			}
		}
		if program != nil {
			program.Send(NoteClosedMsg{Save: save})
		}
	})

	r.client.RegisterHandler("kopr:save-unnamed", func(args ...interface{}) {
		if program != nil {
			program.Send(SaveUnnamedMsg{})
		}
	})

	if err := r.client.Subscribe("kopr:close-note"); err != nil {
		return err
	}
	if err := r.client.Subscribe("kopr:save-unnamed"); err != nil {
		return err
	}

	cid := r.client.ChannelID()
	lua := fmt.Sprintf(`
local chan = %d

-- Intercept all quit commands via QuitPre autocmd.
-- Throwing an error from QuitPre aborts the :q/:wq/:qa etc.
-- For :wq on named files, the write has already happened before QuitPre fires.
vim.api.nvim_create_autocmd('QuitPre', {
  callback = function()
    vim.rpcnotify(chan, 'kopr:close-note', false)
    error('Kopr')
  end,
})

-- ZZ = save and close note, ZQ = discard and close note
vim.keymap.set('n', 'ZZ', function()
  vim.rpcnotify(chan, 'kopr:close-note', true)
end, {noremap=true})
vim.keymap.set('n', 'ZQ', function()
  vim.rpcnotify(chan, 'kopr:close-note', false)
end, {noremap=true})

-- Intercept :w on unnamed buffers via cnoreabbrev.
-- Uses single quotes in the Vimscript ternary to avoid escaping issues.
vim.cmd([[cnoreabbrev <expr> w getcmdtype()==':' && getcmdline()=='w' && bufname()=='' ? 'lua vim.rpcnotify(]] .. chan .. [[, "kopr:save-unnamed")' : 'w']])
`, cid)

	return r.client.ExecLua(lua, nil)
}

// CursorPosition returns the current cursor position as (line, col).
// Line is 1-based, col is 0-based (matching Neovim convention).
func (r *RPC) CursorPosition() (int, int, error) {
	var pos [2]int
	err := r.client.ExecLua("return vim.api.nvim_win_get_cursor(0)", &pos)
	if err != nil {
		return 0, 0, err
	}
	return pos[0], pos[1], nil
}

// SetupLinkNavigation maps gf/gb in normal mode to send RPC notifications
// for following wiki links and navigating back.
func (r *RPC) SetupLinkNavigation(program *tea.Program) error {
	r.client.RegisterHandler("kopr:follow-link", func(args ...interface{}) {
		if program != nil {
			program.Send(FollowLinkMsg{})
		}
	})

	r.client.RegisterHandler("kopr:go-back", func(args ...interface{}) {
		if program != nil {
			program.Send(GoBackMsg{})
		}
	})

	if err := r.client.Subscribe("kopr:follow-link"); err != nil {
		return err
	}
	if err := r.client.Subscribe("kopr:go-back"); err != nil {
		return err
	}

	cid := r.client.ChannelID()
	lua := fmt.Sprintf(`
vim.keymap.set('n', 'gf', function()
  vim.rpcnotify(%d, 'kopr:follow-link')
end, {noremap=true, desc='Follow wiki link'})
vim.keymap.set('n', 'gb', function()
  vim.rpcnotify(%d, 'kopr:go-back')
end, {noremap=true, desc='Go back to previous note'})
`, cid, cid)

	return r.client.ExecLua(lua, nil)
}

// SetBufferName sets the name of the current buffer.
func (r *RPC) SetBufferName(name string) error {
	buf, err := r.client.CurrentBuffer()
	if err != nil {
		return err
	}
	return r.client.SetBufferName(buf, name)
}

// WriteBuffer writes the current buffer to disk.
func (r *RPC) WriteBuffer() error {
	return r.client.Command("w!")
}

// NewBuffer creates a new empty editable buffer.
func (r *RPC) NewBuffer() error {
	return r.client.Command("enew!")
}

// LoadSplashBuffer creates a scratch buffer for the splash screen.
func (r *RPC) LoadSplashBuffer() error {
	return r.client.Command("enew! | setlocal buftype=nofile bufhidden=wipe nomodifiable noswapfile")
}

// Quit tells Neovim to exit by clearing the quit intercept and running qa!.
func (r *RPC) Quit() {
	if r.client == nil {
		return
	}
	// Remove the QuitPre autocmd that normally aborts :q/:wq.
	r.client.ExecLua("vim.api.nvim_clear_autocmds({event='QuitPre'})", nil)
	// Errors are expected here since nvim may close the connection mid-command.
	r.client.Command("qa!")
}

// Close closes the RPC connection.
func (r *RPC) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}
