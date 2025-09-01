{ pkgs, ... }:

# TODO figure out if I can do this on all servers without conflict
{
  services = {
    k3s = {
      manifests = {
        flux = {
          source = pkgs.fetchurl {
            # nix-prefetch-url
            url = "https://github.com/fluxcd/flux2/releases/download/v2.6.4/install.yaml";
            sha256 = "1f1smpa5jwmb6x13w2zb8wdp8a4b2i386h6yp6s2frv64fl47l3w";
          };
        };
      };
    };
  };
}
