{ lib, ... }:

let
  chartVersion = "v1.19.6";
  namespace = "rook-ceph";
  csiRBDPluginVolume = [
    {
      name = "lib-modules";
      hostPath.path = "/run/booted-system/kernel-modules/lib/modules/";
    }
    {
      name = "host-nix";
      hostPath.path = "/nix";
    }
  ];
  csiRBDPluginVolumeMount = [
    {
      name = "host-nix";
      mountPath = "/nix";
      readOnly = true;
    }
  ];
in
{
  boot.kernelModules = [
    "ceph"
    "rbd"
  ];

  # Rook documents this for NixOS/containerd to avoid slow Ceph commands and
  # mons falling out of quorum. k3s owns containerd in this setup.
  systemd.services.k3s.serviceConfig.LimitNOFILE = lib.mkForce null;

  services.k3s.manifests.rook-ceph = {
    content = [
      # k3s AddOns prune previously-managed objects when they disappear from
      # the manifest. Keep the namespace in this object set to avoid deleting
      # all Rook resources during manifest minimization.
      {
        apiVersion = "v1";
        kind = "Namespace";
        metadata.name = namespace;
      }
      {
        apiVersion = "source.toolkit.fluxcd.io/v1";
        kind = "HelmRepository";
        metadata = {
          name = "rook-ceph";
          namespace = "flux-system";
        };
        spec = {
          interval = "1h";
          url = "https://charts.rook.io/release";
        };
      }
      {
        apiVersion = "helm.toolkit.fluxcd.io/v2";
        kind = "HelmRelease";
        metadata = {
          name = "rook-ceph";
          namespace = "flux-system";
        };
        spec = {
          interval = "30m";
          chart.spec = {
            chart = "rook-ceph";
            version = chartVersion;
            sourceRef = {
              kind = "HelmRepository";
              name = "rook-ceph";
            };
          };
          releaseName = "rook-ceph";
          targetNamespace = namespace;
          install = {
            createNamespace = true;
            remediation.retries = -1;
          };
          values = {
            csi = {
              enableCephfsDriver = false;
              inherit csiRBDPluginVolume csiRBDPluginVolumeMount;
            };
          };
        };
      }
      {
        apiVersion = "helm.toolkit.fluxcd.io/v2";
        kind = "HelmRelease";
        metadata = {
          name = "rook-ceph-cluster";
          namespace = "flux-system";
        };
        spec = {
          interval = "30m";
          dependsOn = [ { name = "rook-ceph"; } ];
          chart.spec = {
            chart = "rook-ceph-cluster";
            version = chartVersion;
            sourceRef = {
              kind = "HelmRepository";
              name = "rook-ceph";
            };
          };
          releaseName = "rook-ceph-cluster";
          targetNamespace = namespace;
          install.remediation.retries = -1;
          values = {
            cephImage.tag = "v20.2.1";
            cephClusterSpec = {
              mon.count = 1;
              mgr.count = 1;
              network.ipFamily = "IPv6";
              dashboard.enabled = false;
              cephConfig.global = {
                mon_warn_on_pool_no_redundancy = "false";
                ms_bind_ipv4 = "false";
                ms_bind_ipv6 = "true";
                osd_pool_default_size = "1";
                osd_pool_default_min_size = "1";
              };
              placement = {
                osd.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms = [
                  {
                    matchExpressions = [
                      {
                        key = "cloudlab.khuedoan.com/rook-storage";
                        operator = "NotIn";
                        values = [ "disabled" ];
                      }
                    ];
                  }
                ];
              };
              disruptionManagement.managePodBudgets = false;
            };
            cephBlockPools = [
              {
                name = "ceph-blockpool";
                spec = {
                  replicated = {
                    size = 1;
                    requireSafeReplicaSize = false;
                  };
                };
                storageClass = {
                  enabled = true;
                  name = "standard";
                  isDefault = true;
                  allowVolumeExpansion = true;
                  parameters = {
                    imageFormat = "2";
                    imageFeatures = "layering";
                    "csi.storage.k8s.io/provisioner-secret-name" = "rook-csi-rbd-provisioner";
                    "csi.storage.k8s.io/provisioner-secret-namespace" = namespace;
                    "csi.storage.k8s.io/controller-expand-secret-name" = "rook-csi-rbd-provisioner";
                    "csi.storage.k8s.io/controller-expand-secret-namespace" = namespace;
                    "csi.storage.k8s.io/controller-publish-secret-name" = "rook-csi-rbd-provisioner";
                    "csi.storage.k8s.io/controller-publish-secret-namespace" = namespace;
                    "csi.storage.k8s.io/node-stage-secret-name" = "rook-csi-rbd-node";
                    "csi.storage.k8s.io/node-stage-secret-namespace" = namespace;
                    "csi.storage.k8s.io/fstype" = "ext4";
                  };
                };
              }
            ];
            cephFileSystems = [ ];
            cephObjectStores = [ ];
          };
        };
      }
    ];
  };
}
