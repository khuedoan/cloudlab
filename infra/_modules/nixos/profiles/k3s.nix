{ config, ... }:

{
  networking = {
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

  services = {
    k3s = {
      enable = true;
      role = "server";
      tokenFile = config.sops.secrets.k3s_token.path;
      extraFlags = toString [
        "--disable-helm-controller"
        "--disable-network-policy"
        "--disable=traefik"
      ];
    };
  };
}
