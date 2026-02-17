{
  description = "C4 ID - Go implementation of SMPTE ST 2114:2017 universal identifiers";

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
        
        # Build the c4 CLI tool
        c4 = pkgs.buildGoModule {
          pname = "c4";
          version = "0.8.1";
          
          src = builtins.path { path = ./.; name = "source"; };
          
          vendorHash = "sha256-M3yJ3sMD78qV8AE9cBXDK6HocZ3+ajf6A9ae9hSi8Oc=";
          
          subPackages = [ "cmd/c4" ];
          
          ldflags = [ "-s" "-w" ];
          
          meta = with pkgs.lib; {
            description = "Go implementation of SMPTE ST 2114:2017 for universally unique and consistent identifiers";
            homepage = "https://github.com/bgyss/c4";
            license = licenses.asl20;
            maintainers = [ ];
            mainProgram = "c4";
          };
        };
        
      in
      {
        # Default package
        packages.default = c4;
        packages.c4 = c4;
        
        # Development shell
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go development
            go_1_24
            gotools
            gopls
            go-tools
            
            # Testing and coverage tools
            gotestsum
            
            # Code formatting and linting
            golangci-lint
            gofumpt
            gosec
            
            # Build tools
            gnumake
            
            # Git for version control
            git
            
            # Benchmarking (included in go-tools above)
          ];
          
          shellHook = ''
            echo "🚀 C4 ID Development Environment"
            echo "Go version: $(go version)"
            echo ""
            echo "Available commands:"
            echo "  go build ./cmd/c4          - Build the CLI tool"
            echo "  go test ./...              - Run all tests"
            echo "  go test -cover ./...       - Run tests with coverage"
            echo "  golangci-lint run          - Run linter"
            echo "  gosec ./...                - Run security scanner"
            echo ""
            echo "CLI usage after building:"
            echo "  ./c4 filename.txt          - Generate C4 ID from file"
            echo "  echo 'data' | ./c4         - Generate C4 ID from stdin"
            echo ""
            
            # Set Go environment
            export GOROOT="$(go env GOROOT)"
            export GOPATH="$(go env GOPATH)"
          '';
        };
        
        # Apps for nix run
        apps.default = flake-utils.lib.mkApp {
          drv = c4;
          name = "c4";
        };
        
        # Checks for nix flake check
        checks = {
          # Build check
          build = c4;
          
          # Test check
          test = pkgs.runCommand "c4-tests" {
            buildInputs = [ pkgs.go_1_24 ];
            src = builtins.path { path = ./.; name = "source"; };
          } ''
            cd $src
            go test ./...
            touch $out
          '';
          
          # Linting check
          lint = pkgs.runCommand "c4-lint" {
            buildInputs = [ pkgs.go_1_24 pkgs.golangci-lint ];
            src = builtins.path { path = ./.; name = "source"; };
          } ''
            cd $src
            golangci-lint run
            touch $out
          '';
          
          # Security check
          security = pkgs.runCommand "c4-security" {
            buildInputs = [ pkgs.go_1_24 pkgs.gosec ];
            src = builtins.path { path = ./.; name = "source"; };
          } ''
            cd $src
            gosec ./...
            touch $out
          '';
        };
      });
}
