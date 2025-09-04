{ pkgs, ... }:

# TODO figure out if I can do this on all servers without conflict
{
  services = {
    k3s = {
      manifests = {
        flux = {
          source = pkgs.runCommand "flux-install-manifest" {} ''
            mkdir -p $out
            ${pkgs.fluxcd}/bin/flux install \
              --components=source-controller,kustomize-controller,helm-controller \
              --export > $out/flux.yaml
          '';
        };
        registry = {
          source = pkgs.runCommand "registry-install-manifest" {} ''
            mkdir -p $out

            ${pkgs.kubectl}/bin/kubectl create namespace registry \
              --output=yaml \
              --dry-run=client \
              > $out/registry-namespace.yaml

            ${pkgs.kubernetes-helm}/bin/helm template \
              --skip-tests \
              --namespace=registry \
              registry \
              ${pkgs.fetchurl {
                url = "https://github.com/project-zot/helm-charts/releases/download/zot-0.1.80/zot-0.1.80.tgz";
                sha256 = "093frrijhycb4dsdwy7h99j5klrphr4xa0chnibmkqiy5znlaf4w";
              }} \
              --version=0.1.67 \
              --values=${pkgs.writeText "values.yaml" (builtins.toJSON {
                nameOverride = "registry";
                image = {
                  repository = "docker.io/khuedoan/zot-airgap";
                };
                mountConfig = true;
                configFiles = {
                  "config.json" = (builtins.toJSON {
                    storage = {
                      rootDirectory = "/var/lib/registry";
                      subpaths = {
                        "/vendor" = {
                          rootDirectory = "/var/lib/registry-airgap";
                        };
                      };
                    };
                    http = {
                      address = "0.0.0.0";
                      port = "5000";
                    };
                    log.level = "debug";
                  });
                };
                service = {
                  type = "ClusterIP";
                };
              })} \
              > $out/registry.yaml
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
                # TODO param?
                # url = "http://forgejo-http.forgejo.svc.cluster.local:3000/khuedoan/cloudlab";
                url = "https://code.khuedoan.com/khuedoan/cloudlab";
                ref = {
                  branch = "fast-cluster"; # TODO back to master ofc
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
                sourceRef = {
                  kind = "GitRepository";
                  name = "gitops";
                };
                path = "./platform/production";
                prune = true;
                interval = "30m";
              };
            }
          ];
        };
      };
    };
  };
}
