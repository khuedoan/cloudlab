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
            flux install \
              --components=source-controller,kustomize-controller,helm-controller \
              --export > $out
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
            helm template --skip-tests registry ${
                pkgs.fetchurl {
                  url = "https://github.com/project-zot/helm-charts/releases/download/zot-0.1.67/zot-0.1.67.tgz";
                  sha256 = "118js6m16fvzxxjznydjp6kip67548s6l47zvp0fjjsz9fzz438r";
                }
              } \
              --namespace registry \
              --values ${./values/registry.yaml} > $out
          '';
        };
        gateway-api = {
          source = pkgs.fetchurl {
            url = "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.1/standard-install.yaml";
            sha256 = "1lnsairn0169xg4ylngaiaib3lsv9rkgqsf9ja63l0myyrvipfbk";
          };
        };
        gitops = {
          content = [
            {
              apiVersion = "source.toolkit.fluxcd.io/v1";
              kind = "OCIRepository";
              metadata = {
                name = "platform";
                namespace = "flux-system";
              };
              spec = {
                interval = "30s";
                url = "oci://registry.registry.svc.cluster.local:5000/platform";
                insecure = true;
                ref = {
                  tag = "latest";
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
                path = ".";
                prune = true;
                sourceRef = {
                  kind = "OCIRepository";
                  name = "platform";
                };
              };
            }
          ];
        };
      };
    };
  };
}
