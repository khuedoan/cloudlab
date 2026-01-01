{ pkgs, ... }:

# TODO figure out if I can do this on all servers without conflict
{
  services = {
    k3s = {
      # TODO we may run into consistency problem with multiple master nodes:
      # From https://docs.k3s.io/installation/packaged-components:
      # If you have multiple server nodes, and place additional AddOn manifests on
      # more than one server, it is your responsibility to ensure that files stay
      # in sync across those nodes. K3s does not sync AddOn content between
      # nodes, and cannot guarantee correct behavior if different servers attempt
      # to deploy conflicting manifests.
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
        gitops = {
          content = [
            {
              apiVersion = "source.toolkit.fluxcd.io/v1";
              kind = "GitRepository";
              metadata = {
                name = "gitops";
                namespace = "flux-system";
              };
              spec = {
                interval = "1m";
                # TODO use internal URL
                # url = "http://forgejo-http.forgejo.svc.cluster.local:3000/khuedoan/cloudlab";
                url = "https://code.khuedoan.com/khuedoan/cloudlab";
                ref = {
                  branch = "master";
                };
              };
            }
            {
              apiVersion = "kustomize.toolkit.fluxcd.io/v1";
              kind = "Kustomization";
              metadata = {
                name = "platform";
                namespace = "flux-system";
              };
              spec = {
                interval = "1m";
                path = "platform/staging";
                prune = true;
                sourceRef = {
                  kind = "GitRepository";
                  name = "gitops";
                };
              };
            }
          ];
        };
      };
    };
  };
}
