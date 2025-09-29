package main

import (
	"github.com/muhlba91/pulumi-proxmoxve/sdk/v7/go/proxmoxve/vm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func infra(ctx *pulumi.Context) error {
	_, err := vm.NewVirtualMachine(ctx, "testpulumi", &vm.VirtualMachineArgs{
		NodeName: pulumi.String("proxmox"),
		Cpu: vm.VirtualMachineCpuArgs{
			Cores: pulumi.Int(4),
			Type:  pulumi.String("host"),
		},
		Memory: vm.VirtualMachineMemoryArgs{
			Dedicated: pulumi.Int(1024 * 4),
			Floating:  pulumi.Int(1024 * 4),
		},
		Cdrom: vm.VirtualMachineCdromArgs{
			Enabled:   pulumi.Bool(true),
			FileId:    pulumi.String("local:iso/nixos-installer.iso"),
			Interface: pulumi.String("ide3"),
		},
		Bios: pulumi.String("ovmf"),
		Disks: vm.VirtualMachineDiskArray{
			&vm.VirtualMachineDiskArgs{
				DatastoreId: pulumi.String("local-lvm"),
				Interface:   pulumi.String("scsi0"),
				Size:        pulumi.Int(254),
				FileFormat:  pulumi.String("raw"),
			},
		},
		BootOrders: pulumi.StringArray{
			pulumi.String("scsi0"),
			pulumi.String("ide3"),
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func main() {
	pulumi.Run(infra)
}
