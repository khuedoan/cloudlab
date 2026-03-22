{ modulesPath, pkgs, ... }:

{
  imports = [
    (modulesPath + "/profiles/all-hardware.nix")
  ];

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
    tempAddresses = "disabled";
  };

  systemd = {
    network = {
      enable = true;
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
    openssh = {
      enable = true;
    };
    qemuGuest = {
      enable = true;
    };
  };

  users.users = {
    admin = {
      isNormalUser = true;
      extraGroups = [
        "wheel"
      ];
      openssh.authorizedKeys.keys = [
        "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN5ue4np7cF34f6dwqH1262fPjkowHQ8irfjVC156PCG"
      ];
    };
    root = {
      openssh.authorizedKeys.keys = [
        "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN5ue4np7cF34f6dwqH1262fPjkowHQ8irfjVC156PCG"
      ];
    };
  };

  sops = {
    age = {
      keyFile = "/var/lib/secrets/age";
    };
    defaultSopsFile = ./secrets.yaml;
    secrets = {
      k3s_token = {};
    };
  };

  environment.etc."rancher/k3s/registries.yaml".text =
    pkgs.lib.generators.toYAML { } {
      # SEe ./profiles/values/registry.yaml for registry service IP
      mirrors."registry.registry.svc.cluster.local".endpoint = [
        "http://[fd6a:7c7b:3e12:100::f]:5000"
      ];
    };

  system.stateVersion = "25.05";
}
