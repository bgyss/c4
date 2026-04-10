{
  description = "C4 development environment and multi-platform builds";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachSystem [
      "aarch64-darwin"
      "aarch64-linux"
      "i686-linux"
      "x86_64-darwin"
      "x86_64-linux"
    ] (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        go = if pkgs ? go_1_24 then pkgs.go_1_24 else pkgs.go;

        c4 = pkgs.buildGoModule {
          pname = "c4";
          version = "1.0.12";
          src = builtins.path { path = ./.; name = "source"; };
          vendorHash = null;
          subPackages = [ "cmd/c4" ];
          ldflags = [ "-s" "-w" ];
          inherit go;

          meta = with pkgs.lib; {
            description = "Universal content identification and c4m tooling";
            homepage = "https://github.com/Avalanche-io/c4";
            license = licenses.asl20;
            maintainers = [ ];
            mainProgram = "c4";
          };
        };
      in
      {
        packages.default = c4;
        packages.c4 = c4;

        apps.default = flake-utils.lib.mkApp {
          drv = c4;
          name = "c4";
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gofumpt
            gotestsum
            golangci-lint
            gnumake
            git
          ];

          shellHook = ''
            echo "C4 development shell"
            echo "Go version: $(go version)"
            echo "Useful commands: go test ./..., go vet ./..., nix build"
          '';
        };

        checks = {
          build = c4;

          test = pkgs.runCommand "c4-tests" {
            nativeBuildInputs = [ go ];
            src = builtins.path { path = ./.; name = "source"; };
          } ''
            export HOME="$TMPDIR"
            cd $src
            go test ./...
            touch $out
          '';

          vet = pkgs.runCommand "c4-vet" {
            nativeBuildInputs = [ go ];
            src = builtins.path { path = ./.; name = "source"; };
          } ''
            export HOME="$TMPDIR"
            cd $src
            go vet ./...
            touch $out
          '';
        };
      });
}
