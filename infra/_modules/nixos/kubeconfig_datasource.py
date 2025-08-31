#! /usr/bin/env nix-shell
#! nix-shell -i python3 -p python313Packages.pyyaml

import json
import subprocess
import sys
import yaml

def get_kubeconfig(host, user):
    try:
        result = subprocess.check_output(
            ["ssh", f"{user}@{host}", "cat /etc/rancher/k3s/k3s.yaml"],
            stderr=subprocess.STDOUT
        ).decode("utf-8")

        # Replace the value of the server field with the IP of the K3s server
        config = yaml.safe_load(result)
        config["clusters"][0]["cluster"]["server"] = f"https://[{host}]:6443"

        updated_yaml = yaml.dump(config, default_flow_style=False)

        return {"kubeconfig": updated_yaml}
    # TODO fail hard when error
    except subprocess.CalledProcessError as e:
        return {"error": e.output.decode("utf-8")}
    except Exception as e:
        return {"error": str(e)}

if __name__ == "__main__":
    args = json.load(sys.stdin)
    host = args.get("host")
    user = args.get("user", "root")
    output = get_kubeconfig(host, user)
    print(json.dumps(output))
