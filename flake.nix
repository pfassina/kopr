{
  description = "Kopr - terminal-first knowledge management";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        baseShell = { packages ? [ ], shellHookExtra ? "" }:
          pkgs.mkShell {
            packages = packages;

            shellHook = ''
              export CGO_ENABLED=0
              export PATH="$PWD/bin:$PATH"
              ${shellHookExtra}
            '';
          };
      in
      {
        # Full-featured dev shell for humans.
        devShells.default = baseShell {
          packages = with pkgs; [
            go
            gnumake
            gopls
            delve
            golangci-lint
            neovim
            sqlite
          ];
          shellHookExtra = "echo \"Kopr dev shell ready\"";
        };

        # Minimal shell for CI (smaller closure => less to download).
        devShells.ci = baseShell {
          packages = with pkgs; [
            go
            gnumake
            golangci-lint
            sqlite
          ];
        };
      }
    );
}
