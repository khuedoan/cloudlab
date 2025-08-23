{ pkgs, ... }:

{
  disko.devices = {
    disk = {
      main = {
        type = "disk";
        # TODO don't hard code device?
        device = "/dev/sda";
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
        "--snapshotter=stargz"

        # TODO better ipv6 ipam
        "--cluster-cidr=2001:cafe:42::/56"
        "--service-cidr=2001:cafe:43::/112"
      ];
    };
  };

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

  system.stateVersion = "25.05";
}
