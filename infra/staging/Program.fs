module Program

open Pulumi.FSharp;
open Pulumi.ProxmoxVE.VM;
open Pulumi.ProxmoxVE.VM.Inputs;

let infra () =
    let node =
        VirtualMachine("hellopulumi", VirtualMachineArgs(
            NodeName = "proxmox",
            Cpu = VirtualMachineCpuArgs(
                Cores = input 2,
                Type = "host"
            ),
            Memory = VirtualMachineMemoryArgs(
                Dedicated = input (1024 * 4),
                Floating = input (1024 * 4)
            ),
            Cdrom = VirtualMachineCdromArgs(
                Enabled = true,
                FileId = "local:iso/nixos-installer.iso"
            ),
            Bios = "ovmf"
        ))
    dict [ "vm", node :> obj ]

[<EntryPoint>]
let main _ =
    Deployment.run infra
