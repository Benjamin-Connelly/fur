{
  description = "fur - dual-mode markdown navigator with TUI and web interfaces";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = "0.4.0-dev";
      in
      {
        packages = {
          fur = pkgs.buildGoModule {
            pname = "fur";
            inherit version;
            src = ./.;

            vendorHash = null;

            ldflags = [
              "-s" "-w"
              "-X main.version=${version}"
              "-X main.date=1970-01-01T00:00:00Z"
            ];

            subPackages = [ "cmd/fur" ];

            postInstall = ''
              $out/bin/fur completion bash > fur.bash
              $out/bin/fur completion zsh > _fur
              $out/bin/fur completion fish > fur.fish
              installShellCompletion fur.bash _fur fur.fish

              installManPage man/man1/*.1
            '';

            nativeBuildInputs = [ pkgs.installShellCompletion ];

            meta = with pkgs.lib; {
              description = "Dual-mode markdown navigator with TUI and web interfaces";
              homepage = "https://github.com/Benjamin-Connelly/fur";
              license = licenses.mit;
              mainProgram = "fur";
            };
          };

          default = self.packages.${system}.fur;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            golangci-lint
          ];
        };
      }
    );
}
