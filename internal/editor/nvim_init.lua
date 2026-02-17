-- Kopr managed Neovim configuration
-- This file lives at ~/.config/kopr/init.lua
-- Edit freely — Kopr won't overwrite your changes.
-- Run `kopr --reset-nvim-config` to restore defaults.

-- Disable UI chrome (Kopr draws its own status/tabs)
vim.opt.laststatus = 0
vim.opt.showtabline = 0
vim.opt.cmdheight = 1
vim.opt.ruler = false
vim.opt.showcmd = false
vim.opt.showmode = false
vim.opt.signcolumn = "no"
vim.opt.foldcolumn = "1"
vim.opt.number = false
vim.opt.relativenumber = false

-- Suppress intro screen
vim.opt.shortmess:append("I")

-- Sensible editing defaults
vim.opt.wrap = true
vim.opt.linebreak = true
vim.opt.breakindent = true
vim.opt.scrolloff = 8

-- Tabs/spaces
vim.opt.expandtab = true
vim.opt.shiftwidth = 4
vim.opt.tabstop = 4

-- No swap/backup (Kopr manages the vault)
vim.opt.swapfile = false
vim.opt.backup = false
vim.opt.writebackup = false

-- Persistent undo
vim.opt.undofile = true

-- Colors
vim.opt.termguicolors = true
pcall(vim.cmd, "colorscheme no-clown-fiesta")

-- Disable unused built-in plugins
vim.g.loaded_netrw = 1
vim.g.loaded_netrwPlugin = 1
vim.g.loaded_tutor = 1
vim.g.loaded_zipPlugin = 1
vim.g.loaded_zip = 1
vim.g.loaded_tarPlugin = 1
vim.g.loaded_tar = 1
vim.g.loaded_gzip = 1

-- Alt key mappings for insert mode (word-level navigation/editing)
vim.keymap.set("i", "<M-BS>", "<C-w>", { noremap = true })
vim.keymap.set("i", "<M-Left>", "<C-Left>", { noremap = true })
vim.keymap.set("i", "<M-Right>", "<C-Right>", { noremap = true })

-- Markdown-specific settings
vim.api.nvim_create_autocmd("FileType", {
    pattern = "markdown",
    callback = function()
        vim.opt_local.shiftwidth = 2
        vim.opt_local.tabstop = 2
        vim.opt_local.conceallevel = 2
    end,
})

-- Render-markdown plugin (live markdown preview via conceal)
pcall(function()
    require("render-markdown").setup({
        render_modes = { "n", "v", "i", "c" },
    })
end)
