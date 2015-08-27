# Joker

> Some men just want to watch the world burn.

In the vein of Netflix's [Chaos Engineering](http://techblog.netflix.com/2014/09/introducing-chaos-engineering.html) and Aphyr's [Jepsen](https://github.com/aphyr/jepsen) testing, Joker seeks to create reproducible failures in distributed systems and observe the behavior of systems under partial failure.

## Getting started
These are the instructions for getting started with Joker on Arch Linux. Directions are similar on other operating systems but packages have different names and dependencies. (note: If you want an even easier starting point create a VM with Ubuntu 15.04 and run `apt-get install -y lxc`)

- install lxc and other required tools `pacman -S lxc libvirt ebtables dnsmasq`
- check configuration (user namespaces should be 'missing' due to security concerns) `lxc-checkconfig`
- create a base vm `lxc-create -n base -t ubuntu -- --release precise` (Ubuntu 12.04 to mimic Distil deployments; requires installing [debootstrap](https://aur.archlinux.org/packages/debootstrap/) and [ubuntu-keyring](https://aur.archlinux.org/packages/ubuntu-keyring/) from AUR and linking the installed executable `ln -sf /usr/bin/debootstrap /usr/bin/qemu-debootstrap` and `ln -sf /usr/bin/gpgv /usr/bin/gpg1v` )
- set network config (add following to /var/lib/lxc/base/config)
    lxc.network.type = veth
    lxc.network.flags = up
    lxc.network.link = virbr0
    lxc.network.ipv4 = 0.0.0.0/24
    lxc.network.hwaddr = 00:1E:62:AA:AA:AA
- configure vm as desired (e.g. run chef inside container)
- create network `systemctl start libvirtd` (edit with `virsh net-edit default` as desired)
- start network `sudo virsh net-start default
