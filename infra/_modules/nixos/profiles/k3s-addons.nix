{ pkgs, ... }:

# TODO figure out if I can do this on all servers without conflict
{
  services = {
    k3s = {
      manifests = {
        flux = {
          source = pkgs.runCommand "flux-install-manifest" {
            nativeBuildInputs = [ pkgs.fluxcd ];
          } ''
            mkdir -p $out
            flux install \
              --components=source-controller,kustomize-controller,helm-controller \
              --export > $out/flux.yaml
          '';
        };
      };
    };
  };
}
