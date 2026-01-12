{ config, ... }:

{
  networking = {
    firewall = {
      # https://docs.k3s.io/installation/requirements#inbound-rules-for-k3s-nodes
      allowedTCPPorts = [
        6443  # K3s supervisor and Kubernetes API Server
        10250 # Kubelet metrics
        # TODO probably unify with the agent config?
        80    # HTTP
        443   # HTTPS
      ];
      allowedTCPPortRanges = [
        # Required only for HA with embedded etcd
        { from = 2379; to = 2380; }
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
      role = "server";
      tokenFile = config.sops.secrets.k3s_token.path;
      extraFlags = [
        "--disable-helm-controller"
        "--disable-network-policy"
        "--disable=traefik"
        "--cluster-cidr=fd6a:7c7b:3e12:0::/56" # TODO proper ULA planning
        "--service-cidr=fd6a:7c7b:3e12:100::/112" # TODO proper ULA planning
        "--flannel-backend=wireguard-native"
        "--flannel-external-ip"
        "--flannel-ipv6-masq" # Enable IPv6 NAT, as per default pods use their pod IPv6 address for outgoing traffic
        # TODO net.ipv6.conf.all.accept_ra=2
      ];
    };
  };
}
