# Hetzner Metal

Manual notes for Hetzner dedicated servers created through the server auction.

There's no official Hetzner Robot Terraform provider,
though there are some community-maintained ones.
I only have a few servers rented through auctions,
and they require manual steps to select and optimize costs anyway,
so I treat them the same way as my other metal servers:
hardware purchase is manual, but OS installation and management are automated.

## Server Auction

- Listing: <https://www.hetzner.com/sb?currency=USD>

### Before ordering

- Use an SSH key instead of a password
- Create and save the SSH keypair in password manage e.g.,`~/.ssh/hetzner_metal`

### After ordering

Management console: [Robot](https://robot.hetzner.com/server)

- View the confirmation email for the server IP address
- Rename the server in Robot console
- Connect to the server:

```sh
ssh -i ~/.ssh/hetzner_metal root@$IP
```

## Inventory

- `hetzner-metal-1`:
    - CPU: AMD Ryzen 5 3600
    - RAM: 64 GB ECC
    - Drives: 2 x 2 TB HDD
    - Networking: 1 Gbps Intel NIC, IPv6-only
    - Location: `HEL1-DC3` (Helsinki, Finland)
