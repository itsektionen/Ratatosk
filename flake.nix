{
  description = "Ratatosk";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { nixpkgs, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          # process-compose serves its own API on 8080 by default, which the backend uses
          PC_PORT_NUM = "8099";
          # otherwise it may pick up the Docker compose.yml
          PC_CONFIG_FILES = "process-compose.yml";

          packages = [
            pkgs.go_1_27
            pkgs.gopls
            pkgs.delve
            pkgs.golangci-lint
            pkgs.nodejs
            pkgs.process-compose
            pkgs.nixfmt
          ];
        };
      });

      formatter = forAllSystems (pkgs: pkgs.nixfmt);
    };
}
