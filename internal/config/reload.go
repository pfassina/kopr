package config

// RestartRequired compares two configs and returns the TOML key names of
// changed settings that cannot be applied without restarting Kopr. These are
// wired at startup (vault, watcher, nvim profile/PTY) and cannot be swapped
// live.
func RestartRequired(cur, next Config) []string {
	var changed []string
	if cur.VaultPath != next.VaultPath {
		changed = append(changed, "vault_path")
	}
	if cur.NvimMode != next.NvimMode {
		changed = append(changed, "nvim_mode")
	}
	if cur.RenderMath != next.RenderMath {
		changed = append(changed, "render_math")
	}
	if cur.TreesitterParsers != next.TreesitterParsers {
		changed = append(changed, "treesitter_parsers")
	}
	return changed
}
