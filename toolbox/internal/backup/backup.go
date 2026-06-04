package backup

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strings"

	volsyncv1alpha1 "github.com/backube/volsync/api/v1alpha1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/ptr"
	k8syaml "sigs.k8s.io/yaml"
)

const (
	schedule           = "*/30 * * * *"
	pruneInterval      = 14
	enableFileDeletion = true
	vaultRefPrefix     = "vault:secret/data/backup"
)

type Config struct {
	Backup struct {
		Volumes map[string]VolumeSettings `yaml:"volumes"`
	} `yaml:"backups"`
}

type VolumeSettings struct {
	MoverSecurityContext *corev1.PodSecurityContext `yaml:"mover_security_context,omitempty"`
}

type Volume struct {
	Namespace            string
	PVC                  string
	MoverSecurityContext *corev1.PodSecurityContext
}

type Object interface {
	metav1.Object
	runtime.Object
}

func (v Volume) Key() string { return v.Namespace + "/" + v.PVC }

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	return &config, nil
}

func ParseAndValidate(config *Config) ([]Volume, error) {
	if config == nil || len(config.Backup.Volumes) == 0 {
		return nil, fmt.Errorf("backups.volumes: at least one volume is required")
	}
	volumes := make([]Volume, 0, len(config.Backup.Volumes))
	for key, settings := range config.Backup.Volumes {
		namespace, pvc, err := splitVolumeKey("backups.volumes", key)
		if err != nil {
			return nil, err
		}
		volumes = append(volumes, Volume{
			Namespace:            namespace,
			PVC:                  pvc,
			MoverSecurityContext: settings.MoverSecurityContext,
		})
	}
	slices.SortFunc(volumes, func(a, b Volume) int { return cmp.Compare(a.Key(), b.Key()) })
	return volumes, nil
}

func FilterVolumes(volumes []Volume, selectors []string) ([]Volume, error) {
	if len(selectors) == 0 {
		return volumes, nil
	}

	byKey := map[string]Volume{}
	for _, volume := range volumes {
		byKey[volume.Key()] = volume
	}

	selected := map[string]struct{}{}
	for _, selector := range selectors {
		key := strings.TrimSpace(selector)
		if _, _, err := splitVolumeKey("backup volume selector", key); err != nil {
			return nil, err
		}
		volume, ok := byKey[key]
		if !ok {
			return nil, fmt.Errorf("backup volume selector %q is not configured", key)
		}
		selected[volume.Key()] = struct{}{}
	}

	filtered := make([]Volume, 0, len(selected))
	for _, volume := range volumes {
		if _, ok := selected[volume.Key()]; ok {
			filtered = append(filtered, volume)
		}
	}
	return filtered, nil
}

func BuildSetupObjects(volumes []Volume) []Object {
	objects := make([]Object, 0, len(volumes)*2)
	for _, volume := range volumes {
		objects = append(objects, repositorySecret(volume), replicationSource(volume))
	}
	return objects
}

func BuildRestoreObjects(volumes []Volume, restoreTrigger string) []Object {
	objects := make([]Object, 0, len(volumes)*2)
	for _, volume := range volumes {
		objects = append(objects, repositorySecret(volume), replicationDestination(volume, restoreTrigger))
	}
	return objects
}

func RenderYAML(objects []Object) ([]byte, error) {
	var builder strings.Builder
	for i, object := range objects {
		if i > 0 {
			builder.WriteString("---\n")
		}
		data, err := k8syaml.Marshal(object)
		if err != nil {
			return nil, fmt.Errorf("marshal %s/%s: %w", object.GetNamespace(), object.GetName(), err)
		}
		builder.Write(data)
	}
	return []byte(builder.String()), nil
}

func DestinationName(volume Volume) string { return volume.PVC + "-restore" }

func splitVolumeKey(context, key string) (string, string, error) {
	namespace, pvc, ok := strings.Cut(key, "/")
	if !ok || len(validation.IsDNS1123Label(namespace)) > 0 || len(validation.IsDNS1123Label(pvc)) > 0 {
		return "", "", fmt.Errorf("%s: expected valid namespace/pvc, got %q", context, key)
	}
	return namespace, pvc, nil
}

func repositorySecret(volume Volume) Object {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      repositorySecretName(volume),
			Namespace: volume.Namespace,
			Annotations: map[string]string{
				"vault.security.banzaicloud.io/vault-addr": "http://vault.vault.svc.cluster.local:8200",
				"vault.security.banzaicloud.io/vault-role": "default",
				"vault.security.banzaicloud.io/vault-path": "kubernetes",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"RESTIC_REPOSITORY":     []byte(resticRepository(volume)),
			"RESTIC_PASSWORD":       []byte(vaultRefPrefix + "/restic#password"),
			"AWS_ACCESS_KEY_ID":     []byte(vaultRefPrefix + "/s3#access_key_id"),
			"AWS_SECRET_ACCESS_KEY": []byte(vaultRefPrefix + "/s3#secret_access_key"),
		},
	}
}

func replicationSource(volume Volume) Object {
	restic := &volsyncv1alpha1.ReplicationSourceResticSpec{
		ReplicationSourceVolumeOptions: volsyncv1alpha1.ReplicationSourceVolumeOptions{
			CopyMethod: volsyncv1alpha1.CopyMethodSnapshot,
		},
		Repository:        repositorySecretName(volume),
		PruneIntervalDays: ptr.To[int32](pruneInterval),
		Retain: &volsyncv1alpha1.ResticRetainPolicy{
			Hourly:  ptr.To[int32](6),
			Daily:   ptr.To[int32](5),
			Weekly:  ptr.To[int32](4),
			Monthly: ptr.To[int32](2),
			Yearly:  ptr.To[int32](1),
		},
	}
	restic.MoverSecurityContext = moverSecurityContext(volume)

	return &volsyncv1alpha1.ReplicationSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: volsyncv1alpha1.GroupVersion.String(),
			Kind:       "ReplicationSource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      volume.PVC + "-backup",
			Namespace: volume.Namespace,
		},
		Spec: volsyncv1alpha1.ReplicationSourceSpec{
			SourcePVC: volume.PVC,
			Trigger: &volsyncv1alpha1.ReplicationSourceTriggerSpec{
				Schedule: ptr.To(schedule),
			},
			Restic: restic,
		},
	}
}

func replicationDestination(volume Volume, restoreTrigger string) Object {
	restic := &volsyncv1alpha1.ReplicationDestinationResticSpec{
		ReplicationDestinationVolumeOptions: volsyncv1alpha1.ReplicationDestinationVolumeOptions{
			CopyMethod:     volsyncv1alpha1.CopyMethodDirect,
			DestinationPVC: &volume.PVC,
		},
		Repository:         repositorySecretName(volume),
		EnableFileDeletion: enableFileDeletion,
	}
	restic.MoverSecurityContext = moverSecurityContext(volume)

	return &volsyncv1alpha1.ReplicationDestination{
		TypeMeta: metav1.TypeMeta{
			APIVersion: volsyncv1alpha1.GroupVersion.String(),
			Kind:       "ReplicationDestination",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      DestinationName(volume),
			Namespace: volume.Namespace,
		},
		Spec: volsyncv1alpha1.ReplicationDestinationSpec{
			Trigger: &volsyncv1alpha1.ReplicationDestinationTriggerSpec{Manual: restoreTrigger},
			Restic:  restic,
		},
	}
}

func resticRepository(volume Volume) string {
	return fmt.Sprintf(
		"s3:${%s/s3#endpoint}/${%s/s3#bucket}/%s/%s",
		vaultRefPrefix,
		vaultRefPrefix,
		volume.Namespace,
		volume.PVC,
	)
}

func repositorySecretName(volume Volume) string { return volume.PVC + "-restic-repository" }

func moverSecurityContext(volume Volume) *corev1.PodSecurityContext {
	if volume.MoverSecurityContext == nil {
		return nil
	}
	return volume.MoverSecurityContext.DeepCopy()
}
