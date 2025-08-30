{ config, ... }:

{
  networking = {
    firewall = {
      # https://docs.k3s.io/installation/requirements#inbound-rules-for-k3s-nodes
      allowedTCPPorts = [
        10250 # Kubelet metrics
      ];
      allowedUDPPorts = [
        51820 # Flannel Wireguard with IPv4
        51821 # Flannel Wireguard with IPv6
      ];
    };
  };

  services = {
    k3s = {
      enable = true;
      role = "agent";
      tokenFile = config.sops.secrets.k3s_token.path;
    };
  };
}
