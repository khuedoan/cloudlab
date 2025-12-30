{
  inputs = {
    nixpkgs.url = "nixpkgs/nixos-25.11";
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
          ./profiles/k3s-server.nix
          ./profiles/k3s-addons.nix
          {
            networking.hostName = "kube-1";
            systemd.network.networks."30-wan" = {
              matchConfig.Name = "ens18";
              networkConfig.DHCP = "ipv4";
              address = [
                hosts.kube-1.ipv6_address
              ];
              routes = [
                { Gateway = "fe80::1"; }
              ];
            };
            services.k3s = {
              clusterInit = true;
              extraFlags = nixpkgs.lib.mkAfter [
                "--node-external-ip=${hosts.kube-1.ipv6_address}"
              ];
            };
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
          ./profiles/k3s-server.nix
          {
            networking.hostName = "kube-2";
            systemd.network.networks."30-wan" = {
              matchConfig.Name = "ens18";
              networkConfig.DHCP = "ipv4";
              address = [
                hosts.kube-2.ipv6_address
              ];
              routes = [
                { Gateway = "fe80::1"; }
              ];
            };
            services.k3s = {
              serverAddr = "https://[${hosts.kube-1.ipv6_address}]:6443";
              extraFlags = nixpkgs.lib.mkAfter [
                "--node-external-ip=${hosts.kube-2.ipv6_address}"
              ];
            };
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
          ./profiles/k3s-server.nix
          {
            networking.hostName = "kube-3";
            systemd.network.networks."30-wan" = {
              matchConfig.Name = "ens18";
              networkConfig.DHCP = "ipv4";
              address = [
                hosts.kube-3.ipv6_address
              ];
              routes = [
                { Gateway = "fe80::1"; }
              ];
            };
            services.k3s = {
              serverAddr = "https://[${hosts.kube-1.ipv6_address}]:6443";
              extraFlags = nixpkgs.lib.mkAfter [
                "--node-external-ip=${hosts.kube-3.ipv6_address}"
              ];
            };
          }
        ];
      };
      kube-4 = nixpkgs.lib.nixosSystem {
        system = "aarch64-linux";
        modules = [
          disko.nixosModules.disko
          sops-nix.nixosModules.sops
          ./configuration.nix
          ./disks.nix
          ./profiles/k3s-agent.nix
          {
            networking.hostName = "kube-4";
            systemd.network.networks."30-wan" = {
              matchConfig.Name = "enp1s0";
              networkConfig.DHCP = "ipv4";
              address = [
                hosts.kube-4.ipv6_address
              ];
              routes = [
                { Gateway = "fe80::1"; }
              ];
            };
            services.k3s = {
              serverAddr = "https://[${hosts.kube-1.ipv6_address}]:6443";
              extraFlags = nixpkgs.lib.mkAfter [
                "--node-external-ip=${hosts.kube-4.ipv6_address}"
              ];
            };
          }
        ];
      };
    };
  };
}
