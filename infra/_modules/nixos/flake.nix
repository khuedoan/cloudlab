{
  inputs = {
    nixpkgs.url = "nixpkgs/nixos-25.05";
    disko = {
      url = "github:nix-community/disko";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };
  outputs = { nixpkgs, disko, ... }: {
    nixosConfigurations =
      let
        hosts = builtins.fromJSON (builtins.readFile ./hosts.json);
      in
      {
      nixos = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          disko.nixosModules.disko
          ./configuration.nix
          ./disks.nix
        ];
      };
      kube-1 = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          disko.nixosModules.disko
          ./configuration.nix
          ./disks.nix
          ./profiles/k3s.nix
          {
            networking.hostName = "kube-1";
            services.k3s.clusterInit = true;
          }
        ];
      };
      kube-2 = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          disko.nixosModules.disko
          ./configuration.nix
          ./disks.nix
          ./profiles/k3s.nix
          {
            networking.hostName = "kube-2";
            services.k3s.serverAddr = hosts.kube-1.ipv6_address;
          }
        ];
      };
      kube-3 = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          disko.nixosModules.disko
          ./configuration.nix
          ./disks.nix
          ./profiles/k3s.nix
          {
            networking.hostName = "kube-3";
            services.k3s.serverAddr = hosts.kube-1.ipv6_address;
          }
        ];
      };
    };
  };
}
