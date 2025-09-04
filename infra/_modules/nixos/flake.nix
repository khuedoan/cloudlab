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
          disko.nixosModules.disko
          sops-nix.nixosModules.sops
          ./profiles/installer.nix
        ];
      };

      # Production
      production-master-1 = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          disko.nixosModules.disko
          sops-nix.nixosModules.sops
          ./configuration.nix
          ./disks.nix
          ./profiles/k3s-server.nix
          ./profiles/k3s-addons.nix
          {
            networking.hostName = "production-master-1";
            systemd.network.networks."30-wan" = {
              matchConfig.Name = "ens18";
              networkConfig.DHCP = "ipv4";
              address = [
                hosts.production-master-1.ipv6_address
              ];
              routes = [
                { Gateway = "fe80::1"; }
              ];
            };
            services.k3s = {
              # TODO may need HA later
              # clusterInit = true;
              disableAgent = true;
              extraFlags = nixpkgs.lib.mkAfter [
                "--node-external-ip=${hosts.production-master-1.ipv6_address}"
              ];
            };
          }
        ];
      };
      production-aGVsbG8K = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          disko.nixosModules.disko
          sops-nix.nixosModules.sops
          ./configuration.nix
          ./disks.nix
          ./profiles/k3s-agent.nix
          {
            networking.hostName = "production-aGVsbG8K";
            systemd.network.networks."30-wan" = {
              matchConfig.Name = "enp1s0";
              networkConfig.DHCP = "ipv4";
              address = [
                hosts.production-aGVsbG8K.ipv6_address
              ];
              routes = [
                { Gateway = "fe80::1"; }
              ];
            };
            services.k3s = {
              serverAddr = "https://[${hosts.production-master-1.ipv6_address}]:6443";
              extraFlags = nixpkgs.lib.mkAfter [
                "--node-external-ip=${hosts.production-aGVsbG8K.ipv6_address}"
              ];
            };
          }
        ];
      };
      production-d29ybGQK = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          disko.nixosModules.disko
          sops-nix.nixosModules.sops
          ./configuration.nix
          ./disks.nix
          ./profiles/k3s-agent.nix
          {
            networking.hostName = "production-d29ybGQK";
            systemd.network.networks."30-wan" = {
              matchConfig.Name = "enp1s0";
              networkConfig.DHCP = "ipv4";
              address = [
                hosts.production-d29ybGQK.ipv6_address
              ];
              routes = [
                { Gateway = "fe80::1"; }
              ];
            };
            services.k3s = {
              serverAddr = "https://[${hosts.production-master-1.ipv6_address}]:6443";
              extraFlags = nixpkgs.lib.mkAfter [
                "--node-external-ip=${hosts.production-d29ybGQK.ipv6_address}"
              ];
            };
          }
        ];
      };
      production-YnJ1aGgK = nixpkgs.lib.nixosSystem {
        system = "aarch64-linux";
        modules = [
          disko.nixosModules.disko
          sops-nix.nixosModules.sops
          ./configuration.nix
          ./disks.nix
          ./profiles/k3s-agent.nix
          {
            networking.hostName = "production-YnJ1aGgK";
            systemd.network.networks."30-wan" = {
              matchConfig.Name = "enp1s0";
              networkConfig.DHCP = "ipv4";
              address = [
                hosts.production-YnJ1aGgK.ipv6_address
              ];
              routes = [
                { Gateway = "fe80::1"; }
              ];
            };
            services.k3s = {
              serverAddr = "https://[${hosts.production-master-1.ipv6_address}]:6443";
              extraFlags = nixpkgs.lib.mkAfter [
                "--node-external-ip=${hosts.production-YnJ1aGgK.ipv6_address}"
              ];
            };
          }
        ];
      };
    };
  };
}
