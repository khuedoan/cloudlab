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
      images = [
        config.services.k3s.package.airgap-images
        # TODO other images too, e.g.:
        # (pkgs.dockerTools.pullImage {
        #   imageName = "docker.io/bitnami/keycloak";
        #   imageDigest = "sha256:714dfadc66a8e3adea6609bda350345bd3711657b7ef3cf2e8015b526bac2d6b";
        #   hash = "sha256-IM2BLZ0EdKIZcRWOtuFY9TogZJXCpKtPZnMnPsGlq0Y=";
        #   finalImageTag = "21.1.2-debian-11-r0";
        # })
      ];
    };
  };
}
