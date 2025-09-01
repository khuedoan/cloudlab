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
        registry-namespace = {
          content = {
            apiVersion = "v1";
            kind = "Namespace";
            metadata = {
              name = "registry";
            };
          };
        };
        registry = {
          source = pkgs.runCommand "registry-install-manifest" {
            nativeBuildInputs = [ pkgs.kubernetes-helm ];
          } ''
            mkdir -p $out
            helm template --skip-tests registry ${
                pkgs.fetchurl {
                  url = "https://github.com/project-zot/helm-charts/releases/download/zot-0.1.67/zot-0.1.67.tgz";
                  sha256 = "118js6m16fvzxxjznydjp6kip67548s6l47zvp0fjjsz9fzz438r";
                }
              } \
              --namespace registry \
              --values ${./values/registry.yaml} > $out/registry.yaml
          '';
        };
      };
    };
  };
}
