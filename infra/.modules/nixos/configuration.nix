{ config, pkgs, ... }:

{
  disko.devices = {
    disk = {
      main = {
        type = "disk";
        content = {
          type = "gpt";
          partitions = {
            boot = {
              size = "1M";
              type = "EF02"; # for grub MBR
            };
            ESP = {
              size = "1G";
              type = "EF00";
              content = {
                type = "filesystem";
                format = "vfat";
                mountpoint = "/boot";
                mountOptions = [ "umask=0077" ];
              };
            };
            root = {
              size = "100%";
              content = {
                type = "filesystem";
                format = "ext4";
                mountpoint = "/";
              };
            };
          };
        };
      };
    };
  };

  boot = {
    loader = {
      systemd-boot = {
        enable = true;
      };
      efi = {
        canTouchEfiVariables = true;
      };
    };
  };

  networking = {
    networkmanager = {
      enable = true;
    };
    firewall = {
      # https://docs.k3s.io/installation/requirements#inbound-rules-for-k3s-server-nodes
      allowedTCPPorts = [
        6443
        10250
      ];
      allowedTCPPortRanges = [
        { from = 2379; to = 2380; }
      ];
    };
  };

  time.timeZone = "Asia/Ho_Chi_Minh";
  i18n.defaultLocale = "en_US.UTF-8";

  nix = {
    settings = {
      experimental-features = [
        "nix-command"
        "flakes"
      ];
    };
    optimise.automatic = true;
    gc = {
      automatic = true;
      dates = "weekly";
      options = "--delete-older-than 30d";
    };
  };

  services = {
    openssh.enable = true;
    k3s = {
      enable = true;
      role = "server";
      extraFlags = toString [
        "--disable-helm-controller"
        "--disable-network-policy"
        "--disable=traefik"
        "--secrets-encryption=true"

        # TODO not sure why it doesn't play nice with the registry config below
        # "--snapshotter=stargz"

        # TODO if the node doesn't have IPv6, this will fail, so we can't enable by default yet
        # "--cluster-cidr=10.42.0.0/16,2001:cafe:42::/56"
        # "--service-cidr=10.43.0.0/16,2001:cafe:43::/112"
      ];
    };
    yggdrasil = {
      enable = true;
      persistentKeys = true;
      settings = {
        Peers = [
          # https://publicpeers.neilalexander.dev
          "tls://sin.yuetau.net:6643" # Singapore
          "tls://mima.localghost.org:443" # Philippines
          "tls://133.18.201.69:54232" # Japan
          "tls://vpn.itrus.su:7992" # Netherlands
          "tls://ygg.jjolly.dev:3443" # United States
        ];
      };
    };
  };

  # TODO switch to services.k3s.registries https://github.com/NixOS/nixpkgs/pull/292023
  # Static ClusterIP so we can pull from the internal registry without going through an ingress for better performance
  # Alternatively this can be done by resolving DNS on the node via CoreDNS in some way, maybe via /etc/resolv.conf?
  # nix-shell -p dig --command "dig @10.43.0.10 zot.zot.svc.cluster.local"
  environment.etc."rancher/k3s/registries.yaml".text = ''
    mirrors:
      zot.zot.svc.cluster.local:
        endpoint:
          - http://10.43.0.50:5000
  '';

  users.users.admin = {
    isNormalUser = true;
    description = "Admin";
    extraGroups = [
      "networkmanager"
      "wheel"
    ];
    packages = with pkgs; [
      neovim
      git
      gnumake
      tmux
    ];
  };

  system.stateVersion = "23.11";
}
