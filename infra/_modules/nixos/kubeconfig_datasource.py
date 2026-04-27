#! /usr/bin/env nix-shell
#! nix-shell -i python3 -p python313Packages.pyyaml

import json
import subprocess
import sys
import time
import yaml

SSH_OPTIONS = [
    "-o", "BatchMode=yes",
    "-o", "ConnectTimeout=10",
    "-o", "StrictHostKeyChecking=no",
    "-o", "UserKnownHostsFile=/dev/null",
    "-o", "GlobalKnownHostsFile=/dev/null",
]


def get_kubeconfig(host, user, retries=30, delay=2):
    last_error = None

    for _ in range(retries):
        try:
            result = subprocess.check_output(
                [
                    "ssh",
                    *SSH_OPTIONS,
                    f"{user}@{host}",
                    "cat /etc/rancher/k3s/k3s.yaml",
                ],
                stderr=subprocess.STDOUT,
            ).decode("utf-8")

            # Replace the server value so clients connect to the public node IP.
            config = yaml.safe_load(result)
            config["clusters"][0]["cluster"]["server"] = f"https://[{host}]:6443"

            updated_yaml = yaml.dump(config, default_flow_style=False)
            return {"kubeconfig": updated_yaml}
        except subprocess.CalledProcessError as e:
            last_error = e.output.decode("utf-8").strip()
        except Exception as e:
            last_error = str(e)

        time.sleep(delay)

    raise RuntimeError(last_error or "timed out waiting for /etc/rancher/k3s/k3s.yaml")

if __name__ == "__main__":
    args = json.load(sys.stdin)
    host = args.get("host")
    user = args.get("user", "root")
    try:
        output = get_kubeconfig(host, user)
        print(json.dumps(output))
    except Exception as e:
        print(str(e), file=sys.stderr)
        sys.exit(1)
