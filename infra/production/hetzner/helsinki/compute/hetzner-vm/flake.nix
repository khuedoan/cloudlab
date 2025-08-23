{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    disko = {
      url = "github:nix-community/disko";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, disko }: {
    nixosConfigurations = {
      master = nixpkgs.lib.nixosSystem {
        system = "aarch64-linux"; # TODO support multiple systems
        modules = [
          disko.nixosModules.disko
          ./configuration.nix
        ];
      };
    };
  };
}
