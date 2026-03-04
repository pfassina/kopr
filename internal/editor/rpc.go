package editor

import (
	"errors"
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
		return nil, errors.Join(fmt.Errorf("setup mode events: %w", err), client.Close())
	}

	return rpc, nil
}

func (r *RPC) setupModeChanged() error {
	if err := r.client.RegisterHandler("mode_changed", func(args ...interface{}) {
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
	}); err != nil {
		return err
	}

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
	if err := r.client.RegisterHandler("kopr:close-note", func(args ...interface{}) {
		save := false
		if len(args) > 0 {
			if b, ok := args[0].(bool); ok {
				save = b
			}
		}
		if program != nil {
			program.Send(NoteClosedMsg{Save: save})
		}
	}); err != nil {
		return err
	}

	if err := r.client.RegisterHandler("kopr:save-unnamed", func(args ...interface{}) {
		if program != nil {
			program.Send(SaveUnnamedMsg{})
		}
	}); err != nil {
		return err
	}

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

-- Intercept :w/:wq/:x on unnamed buffers via cnoreabbrev.
-- Uses single quotes in the Vimscript ternary to avoid escaping issues.
vim.cmd([[cnoreabbrev <expr> w  getcmdtype()==':' && getcmdline()=='w'  && bufname()=='' ? 'lua vim.rpcnotify(]] .. chan .. [[, "kopr:save-unnamed")' : 'w']])
vim.cmd([[cnoreabbrev <expr> wq getcmdtype()==':' && getcmdline()=='wq' && bufname()=='' ? 'lua vim.rpcnotify(]] .. chan .. [[, "kopr:close-note", true)' : 'wq']])
vim.cmd([[cnoreabbrev <expr> x  getcmdtype()==':' && getcmdline()=='x'  && bufname()=='' ? 'lua vim.rpcnotify(]] .. chan .. [[, "kopr:close-note", true)' : 'x']])
`, cid)

	return r.client.ExecLua(lua, nil)
}

// SetupSaveNotify installs an autocmd that notifies Kopr after a buffer is written.
// Used for features like auto-format-on-save.
func (r *RPC) SetupSaveNotify(program *tea.Program) error {
	if err := r.client.RegisterHandler("kopr:buf-written", func(args ...interface{}) {
		if program == nil {
			return
		}
		if len(args) < 1 {
			return
		}
		path, ok := args[0].(string)
		if !ok {
			return
		}
		program.Send(BufferWrittenMsg{Path: path})
	}); err != nil {
		return err
	}
	if err := r.client.Subscribe("kopr:buf-written"); err != nil {
		return err
	}

	cid := r.client.ChannelID()
	lua := fmt.Sprintf(`
vim.api.nvim_create_augroup('KoprBufWrite', {clear=true})
vim.api.nvim_create_autocmd('BufWritePost', {
  group = 'KoprBufWrite',
  callback = function(args)
    -- args.file is the absolute path, empty for unnamed buffers.
    if args == nil or args.file == nil or args.file == '' then
      return
    end
    vim.rpcnotify(%d, 'kopr:buf-written', args.file)
  end,
})
`, cid)
	return r.client.ExecLua(lua, nil)
}

// SetupYankClipboard installs a TextYankPost autocmd that sends yanked text
// back to the Go side via RPC, so it can be forwarded to the system clipboard.
func (r *RPC) SetupYankClipboard(program *tea.Program) error {
	if err := r.client.RegisterHandler("kopr:yank", func(args ...interface{}) {
		if program == nil || len(args) < 1 {
			return
		}
		text, ok := args[0].(string)
		if !ok {
			return
		}
		program.Send(YankMsg{Text: text})
	}); err != nil {
		return err
	}

	if err := r.client.Subscribe("kopr:yank"); err != nil {
		return err
	}

	cid := r.client.ChannelID()
	lua := fmt.Sprintf(`
vim.api.nvim_create_autocmd('TextYankPost', {
  callback = function()
    local text = table.concat(vim.v.event.regcontents, '\n')
    vim.rpcnotify(%d, 'kopr:yank', text)
  end,
})
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

// SetCursorPosition sets the current window cursor position.
// Line is 1-based, col is 0-based.
func (r *RPC) SetCursorPosition(line, col int) error {
	return r.client.ExecLua("vim.api.nvim_win_set_cursor(0, {...})", nil, line, col)
}

// SetBufferLines replaces the entire contents of the current buffer.
func (r *RPC) SetBufferLines(lines []string) error {
	return r.client.ExecLua(`
local lines = ...
local buf = vim.api.nvim_get_current_buf()
vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)
`, nil, lines)
}

// SetupLinkNavigation maps gf/gb in normal mode to send RPC notifications
// for following wiki links and navigating back.
func (r *RPC) SetupLinkNavigation(program *tea.Program) error {
	if err := r.client.RegisterHandler("kopr:follow-link", func(args ...interface{}) {
		if program != nil {
			program.Send(FollowLinkMsg{})
		}
	}); err != nil {
		return err
	}

	if err := r.client.RegisterHandler("kopr:go-back", func(args ...interface{}) {
		if program != nil {
			program.Send(GoBackMsg{})
		}
	}); err != nil {
		return err
	}

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
	// Errors are expected during shutdown: Neovim may close the connection mid-command.
	r.client.ExecLua("vim.api.nvim_clear_autocmds({event='QuitPre'})", nil) //nolint:errcheck // shutdown
	r.client.Command("qa!")                                                 //nolint:errcheck // shutdown
}

// ApplyColorscheme sets the active colorscheme in Neovim.
func (r *RPC) ApplyColorscheme(name string) error {
	return r.client.ExecLua("vim.cmd('colorscheme ' .. ...)", nil, name)
}

// ExtractColors queries Neovim highlight groups and returns a map of
// group name → [fg, bg] hex color strings. Empty string means the group
// did not define that attribute.
func (r *RPC) ExtractColors() (map[string][2]string, error) {
	groups := []string{
		"Normal", "Function", "Keyword", "Comment",
		"NonText", "LineNr", "WinSeparator",
		"StatusLine", "DiagnosticError",
		"String", "Visual", "WarningMsg",
		"@markup.heading.1.markdown", "@markup.heading.2.markdown",
		"markdownH1", "markdownH2",
	}

	result := make(map[string][2]string, len(groups))

	for _, g := range groups {
		var raw map[string]interface{}
		err := r.client.ExecLua(
			"return vim.api.nvim_get_hl(0, {name=..., link=false})",
			&raw, g,
		)
		if err != nil {
			continue // group may not exist in this colorscheme
		}
		var pair [2]string
		if fg, ok := raw["fg"]; ok {
			pair[0] = intToHex(fg)
		}
		if bg, ok := raw["bg"]; ok {
			pair[1] = intToHex(bg)
		}
		if pair[0] != "" || pair[1] != "" {
			result[g] = pair
		}
	}

	return result, nil
}

// intToHex converts an integer-typed color value to a #rrggbb hex string.
func intToHex(v interface{}) string {
	switch n := v.(type) {
	case int64:
		return fmt.Sprintf("#%06x", n)
	case uint64:
		return fmt.Sprintf("#%06x", n)
	case float64:
		return fmt.Sprintf("#%06x", int64(n))
	default:
		return ""
	}
}

// ClearHighlightBgs clears explicit backgrounds on common highlight groups
// so Neovim uses the terminal default, preserving terminal transparency.
func (r *RPC) ClearHighlightBgs() {
	for _, g := range []string{"Normal", "NonText", "EndOfBuffer", "FoldColumn", "SignColumn", "NormalNC"} {
		r.ExecCommand("hi " + g + " guibg=NONE") //nolint:errcheck // cosmetic; group may not exist
	}
}

// SetupMathRendering installs a custom math renderer that finds latex_block
// nodes in the built-in markdown_inline treesitter tree, converts them via the
// kopr-latex shell script, and overlays the result using Neovim extmarks.
// This avoids requiring the external latex treesitter parser.
// When enabled is false, any existing math rendering is torn down.
func (r *RPC) SetupMathRendering(enabled bool) error {
	return r.client.ExecLua(mathRenderingLua, nil, enabled)
}

const mathRenderingLua = `
local enabled = ...

-- Disable render-markdown.nvim's built-in latex handler (it requires a
-- treesitter latex parser we don't ship). We handle math ourselves below.
local rm_ok, rm_state = pcall(require, 'render-markdown.state')
if rm_ok and rm_state and rm_state.config then
  if not rm_state.config.latex then rm_state.config.latex = {} end
  rm_state.config.latex.enabled = false
end

local group = 'KoprMath'
vim.api.nvim_create_augroup(group, {clear = true})

if not enabled then return end
if vim.fn.executable('kopr-latex') ~= 1 then return end

local ns = vim.api.nvim_create_namespace('kopr-math')
local conv_cache = {}

local function convert(input)
  if conv_cache[input] ~= nil then return conv_cache[input] end
  local result = vim.system({'kopr-latex'}, {stdin = input, text = true}):wait()
  if result.code == 0 and result.stdout then
    local out = vim.trim(result.stdout)
    if out ~= '' then conv_cache[input] = out; return out end
  end
  conv_cache[input] = false
  return false
end

-- render_math renders all math in buf, skipping nodes that touch skip_row
-- so the cursor line shows raw LaTeX instead of both raw + virtual text.
local function render_math(buf, skip_row)
  vim.api.nvim_buf_clear_namespace(buf, ns, 0, -1)

  local ok, parser = pcall(vim.treesitter.get_parser, buf, 'markdown')
  if not ok then return end
  parser:parse(true)

  local nodes = {}
  parser:for_each_tree(function(tree, lang_tree)
    if lang_tree:lang() ~= 'markdown_inline' then return end
    local function walk(node)
      if node:type() == 'latex_block' then
        nodes[#nodes + 1] = node
        return
      end
      for i = 0, node:child_count() - 1 do walk(node:child(i)) end
    end
    walk(tree:root())
  end)

  for _, node in ipairs(nodes) do
    local sr, sc, er, ec = node:range()

    -- Skip nodes on the cursor line so raw LaTeX is visible for editing
    if skip_row and skip_row >= sr and skip_row <= er then
      goto continue
    end

    local text = vim.treesitter.get_node_text(node, buf)
    local inner = text:match('^%$%$%s*(.-)%s*%$%$$') or text:match('^%$(.-)%$$')
    if not inner or inner == '' then goto continue end
    inner = vim.trim(inner)

    local converted = convert(inner)
    if not converted then goto continue end

    if sr == er then
      -- Inline math: conceal original, show converted inline
      vim.api.nvim_buf_set_extmark(buf, ns, sr, sc, {
        end_row = er, end_col = ec,
        conceal = '',
        virt_text = {{converted, 'RenderMarkdownMath'}},
        virt_text_pos = 'inline',
      })
    else
      -- Block math: conceal all lines, show converted as virtual lines
      for r = sr, er do
        local line = vim.api.nvim_buf_get_lines(buf, r, r + 1, false)[1] or ''
        vim.api.nvim_buf_set_extmark(buf, ns, r, 0, {
          end_row = r, end_col = #line, conceal = '',
        })
      end
      local vlines = {}
      for ln in converted:gmatch('[^\n]+') do
        vlines[#vlines + 1] = {{ln, 'RenderMarkdownMath'}}
      end
      if #vlines > 0 then
        vim.api.nvim_buf_set_extmark(buf, ns, sr, 0, {
          virt_lines = vlines, virt_lines_above = true,
        })
      end
    end
    ::continue::
  end
end

local last_cursor_row = -1

local function scheduled_render(buf, skip_row)
  vim.schedule(function()
    if vim.api.nvim_buf_is_valid(buf) then
      render_math(buf, skip_row)
    end
  end)
end

vim.api.nvim_create_autocmd(
  {'BufEnter', 'TextChanged', 'InsertLeave'},
  {
    group = group,
    pattern = '*.md',
    callback = function(args)
      if not vim.api.nvim_buf_is_valid(args.buf) then return end
      if vim.bo[args.buf].filetype ~= 'markdown' then return end
      last_cursor_row = vim.api.nvim_win_get_cursor(0)[1] - 1
      scheduled_render(args.buf, last_cursor_row)
    end,
  }
)

-- Re-render when cursor moves to a different line (reveal raw on cursor line)
vim.api.nvim_create_autocmd('CursorMoved', {
  group = group,
  pattern = '*.md',
  callback = function(args)
    if not vim.api.nvim_buf_is_valid(args.buf) then return end
    local row = vim.api.nvim_win_get_cursor(0)[1] - 1
    if row == last_cursor_row then return end
    last_cursor_row = row
    render_math(args.buf, row)
  end,
})

-- Render the current buffer immediately if it's markdown
local buf = vim.api.nvim_get_current_buf()
if vim.bo[buf].filetype == 'markdown' then
  last_cursor_row = vim.api.nvim_win_get_cursor(0)[1] - 1
  scheduled_render(buf, last_cursor_row)
end
`

// Close closes the RPC connection.
func (r *RPC) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}
