{
  inputs = {
    nixpkgs.url = "nixpkgs/nixos-25.05";
    disko = {
      url = "github:nix-community/disko";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    sops-nix = {
      url = "github:Mic92/sops-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };
  outputs = { nixpkgs, disko, sops-nix, ... }: {
    nixosConfigurations =
      let
        hosts = builtins.fromJSON (builtins.readFile ./hosts.json);
      in
      {
      installer = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          ./profiles/installer.nix
        ];
      };
      kube-1 = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          disko.nixosModules.disko
          sops-nix.nixosModules.sops
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
          sops-nix.nixosModules.sops
          ./configuration.nix
          ./disks.nix
          ./profiles/k3s.nix
          {
            networking.hostName = "kube-2";
            services.k3s.serverAddr = "https://[${hosts.kube-1.ipv6_address}]:6443";
          }
        ];
      };
      kube-3 = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          disko.nixosModules.disko
          sops-nix.nixosModules.sops
          ./configuration.nix
          ./disks.nix
          ./profiles/k3s.nix
          {
            networking.hostName = "kube-3";
            services.k3s.serverAddr = "https://[${hosts.kube-1.ipv6_address}]:6443";
          }
        ];
      };
    };
  };
}
