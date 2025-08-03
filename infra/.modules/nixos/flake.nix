{
  description = "Netamos";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
    disko = {
      url = "github:nix-community/disko";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, disko }: {
    nixosConfigurations = {
      netamos = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux"; # TODO support multiple systems
        modules = [
          disko.nixosModules.disko
          ./configuration.nix
        ];
      };
      test = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux"; # TODO support multiple systems
        modules = [
          disko.nixosModules.disko
          ./configuration.nix
          ./test.nix
        ];
      };
    };
  };
}
