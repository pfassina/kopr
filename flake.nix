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
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gnumake
            gopls
            delve
            golangci-lint
            neovim
            sqlite
          ];

          shellHook = ''
            export CGO_ENABLED=0
            export PATH="$PWD/bin:$PATH"
            echo "Kopr dev shell ready"
          '';
        };
      }
    );
}
