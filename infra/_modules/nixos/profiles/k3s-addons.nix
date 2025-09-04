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
