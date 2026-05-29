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
  outputs =
    {
      nixpkgs,
      disko,
      sops-nix,
      ...
    }:
    {
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
          hetzner-metal-1 = nixpkgs.lib.nixosSystem {
            system = "x86_64-linux";
            specialArgs = {
              hostConfig = hosts.hetzner-metal-1;
            };
            modules = [
              disko.nixosModules.disko
              sops-nix.nixosModules.sops
              ./configuration.nix
              ./disks.nix
              ./profiles/k3s-server.nix
              ./profiles/k3s-addons.nix
              ./profiles/rook-ceph.nix
              {
                networking = {
                  hostName = "hetzner-metal-1";
                  hosts = {
                    # ffs it's 2026 and GitHub still doesn't have IPv6
                    # Workaround by using the IPv6 proxy thanks to https://danwin1210.de/github-ipv6-proxy.php
                    # TODO make everything air gapped so I don't have to do this anymore
                    "2a01:4f8:c010:d56::6" = [
                      "ghcr.io"
                    ];
                  };
                };
                hardware = {
                  enableRedistributableFirmware = true;
                  cpu.amd.updateMicrocode = true;
                };
                systemd.network.networks."30-wan" = {
                  matchConfig.Name = hosts.hetzner-metal-1.network_interface;
                  networkConfig = {
                    DNS = [
                      # https://developers.cloudflare.com/1.1.1.1/ip-addresses/#block-malware
                      "2606:4700:4700::1112"
                      "2606:4700:4700::1002"
                    ];
                  };
                  address = [
                    "${hosts.hetzner-metal-1.ipv6_address}/64"
                  ];
                  routes = [
                    {
                      Gateway = "fe80::1";
                      GatewayOnLink = true;
                    }
                  ];
                };
                services.k3s = {
                  clusterInit = true;
                  extraFlags = nixpkgs.lib.mkAfter [
                    "--node-external-ip=${hosts.hetzner-metal-1.ipv6_address}"
                  ];
                };
              }
            ];
          };
        };
    };
}
