{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
  };

  outputs =
    { self, nixpkgs }:
    let
      forAllSystems =
        function:
        nixpkgs.lib.genAttrs [
          "x86_64-linux"
          "aarch64-linux"
          "aarch64-darwin"
        ] (system: function (import nixpkgs { inherit system; }));
    in
    {
      devShells = forAllSystems (pkgs: {
        default =
          with pkgs;
          mkShell {
            packages = [
              age
              ansible
              ansible-lint
              fzf
              gnumake
              go
              k3d
              kubectl
              kubernetes-helm
              nixfmt-rfc-style
              nixos-anywhere
              openssh
              opentofu
              oras
              pre-commit
              shellcheck
              sops
              temporal-cli
              terragrunt
              wireguard-tools
              yamlfmt
              yamllint

              (python3.withPackages (
                p: with p; [
                  kubernetes
                ]
              ))

              (pkgs.buildGoModule {
                pname = "toolbox";
                version = "0.1.0";
                src = builtins.path {
                  path = ./toolbox;
                  name = "toolbox-src";
                };
                vendorHash = "sha256-x1C5eVVI4+a/Rj1MPvR2t1YF7wAyrk5ZeaghhA1k6Po=";
              })
            ];
          };
      });
    };
}
