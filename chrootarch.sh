#!/bin/bash -ex
set -ex

KVARIANT=-ec2

DEV=$1
[ -z "$DEV" ] && echo "usage: install <DEVICE>" && exit 1


function import_gpg_key_bbcx() {
	echo "
-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v2

mQENBFctGs4BCACos7zwa5L34JdQJddKR7n8flDkdnKHwtUv8zRiYvFOShcOt2j5
+coF+L++lkoVRgcvrucPG8SoAZp9oJ1Ng7jOBKUuB1gjvZThFypPJ7pc/erpRILM
Ulg+c3QdPGWe5Wa+srbM2vMbr0WkkC9+Lwi0HajJ//pQ+UF9ZPLp8iwGT1jaUkOr
bBbV0WKEslJvLD4oZgzRFH5sQfXCAYvsaQUVoqPg7zD/kjO0MIJ+OJS2GJ8pAH64
F4+gJfNg67Ll1h37qnGqrRVMGyxn+1/6uuR4rslYbEKJX7nncxX/jHRPqvWADRGb
vyDb/6jAEDpjdhmzrqqK1pZqLQkNWL9AO4ZVABEBAAG0LEplcmVteSBEZWluaW5n
ZXIgPGplcmVteWRlaW5pbmdlckBnbWFpbC5jb20+iQE3BBMBCAAhBQJXLRrOAhsD
BQsJCAcCBhUICQoLAgQWAgMBAh4BAheAAAoJEDLLS1XXinC3LhMH/3l6qeEUvglU
hdljaW+APInzPDeagni42Os9iMNBlO4eCi3sGTJ0fBartv0VoVWJqFlYQeiesHvm
fWNN61TTg9SNrymkpbz7af/IvAvZmuL4+cgJohVkwOhS5nXgCstbfEhLBwe+65mI
TfsjJcUlCulrtz61rMwoVLdkCOPX4v23qZS5ew6+J8ovrkXW6SGHz5Ro/mKr8gLY
Uw3HCHj2nQ4EnaT8oIuvV2DsMXpunIejgQTGqU1mh6ZFvI0DiHCi7yD/Wq8A9/Y6
GbSeocqGomPH12mj0EinyyfW9CH+TfbqYvN9lRi+O9kKrBkwJu5PP2jCRQsKUJTb
uldivTRevM+5AQ0EVy0azgEIAOYUy2sdtjWlPqCRrrnWxUU2MlcS8RKtwj7oStoo
zw5JrRmuPHsx9r3RxWtLCrnvCaWhxoR8SrsHNNoyjoSkUaiuDF/HVogmvPPzHA3h
+GpYZ3XhzcxTidH06PAfChGBx+JSfWRVNHRAjehqJXr2KDVT31EHFrbuI0LyITWh
H3xqnu4oKwnRzyG4d573Ax2Wc7UPiRpK5bcziVaFqhZRz62uK+tlzP0QX4sjH79l
gC1qw+8SIPXbwSFlwOMZsDXCYAJJD5XyCeKnnKTXKXWk9ReBb4LPOJH37PTwsm/O
IQom4WNwMZXm9p+BRgCMFlQXMCb4dIi5hEbe225mSRuFlOkAEQEAAYkBHwQYAQgA
CQUCVy0azgIbDAAKCRAyy0tV14pwt42zB/9pEs+hfbrYCtiVkHj2w5Alc+WckqRZ
GKTX7qkpLYTw1ekuJa488r/3h19g6SRPFzXwpQ7/irzj59lxvsYd5NbUHo9ReHsL
FSp5cfHLKI4nwrUKtsKrBoGdMIpsVamUNnqNICKJm4TmrZQBt0/PwSafbByLCGMX
UOZugTACNRQHGxKzYkIFCLzKM5doQ+R3pCYEWKVCZxH1+hCPB4Os5xINlAEdQ3Dx
kM0mdGkyN6Nkc0cDgRk2YB6JznpRarIiZxgFqafH28+fJd+EtpNCM2wZ4f7fvBTI
U2Grq1btbwUI76Fi7mL457+orD0fpryMLzCT09kq8kICr/N9HirDr7yb
=lDfw
-----END PGP PUBLIC KEY BLOCK-----
" | gpg --command-fd 0 --status-fd 2 --batch --homedir $1/etc/pacman.d/gnupg --import -
	printf "y\ny\n" | gpg --command-fd 0 --status-fd 2 --batch --homedir $1/etc/pacman.d/gnupg --no-permission-warning --lsign-key D78A70B7
	printf "4\n" | gpg --command-fd 0 --status-fd 2 --batch --homedir $1/etc/pacman.d/gnupg --no-permission-warning --edit-key D78A70B7 trust quit
	killall gpg-agent || true
}

function import_gpg_key() {
	echo "
-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v2

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
U6en+48h8rjn8za2FyoO
=VKOt
-----END PGP PUBLIC KEY BLOCK-----
" | gpg --command-fd 0 --status-fd 2 --batch --homedir $1/etc/pacman.d/gnupg --import -
	printf "y\ny\n" | gpg --command-fd 0 --status-fd 2 --batch --homedir $1/etc/pacman.d/gnupg --no-permission-warning --lsign-key A7B30DB9
	printf "4\n" | gpg --command-fd 0 --status-fd 2 --batch --homedir $1/etc/pacman.d/gnupg --no-permission-warning --edit-key A7B30DB9 trust quit
	killall gpg-agent || true
}

# arch-chroot breaks if it doesn't see this link, because it believes /dev
# isn't mounted.
[ -L /dev/fd ] || ln -s /proc/self/fd /dev/fd

# We need mirrorlist not to be a symlink on our host for the pacstrap
rm -f /etc/pacman.d/mirrorlist
cp /etc/pacman.d/ec2-mirrors-default /etc/pacman.d/mirrorlist

pacman -Sy --noconfirm parted arch-install-scripts

# Disk partitioning and formatting.
parted $DEV --script -- mklabel msdos
parted $DEV --script -- mkpart primary 0% 100%
mkfs.ext4 -E lazy_itable_init=0 ${DEV}1
mount ${DEV}1 /mnt

# Base install.
pacstrap /mnt base
pacstrap /mnt grub

if [ -L /mnt/etc/pacman.d/gnupg ]; then
	rm -f /mnt/etc/pacman.d/gnupg
	cp -R /run/pacman.d/gnupg /mnt/etc/pacman.d/
	chmod 0700 /mnt/etc/pacman.d/gnupg
fi

# Store partitioning details to fstab
genfstab -U -p /mnt >> /mnt/etc/fstab

# Set up glibc locales
sed -ri 's/^#en_US/en_US/' /mnt/etc/locale.gen
arch-chroot /mnt /bin/bash -c "locale-gen"

# Set timezone to UTC (only one that makes sense for EC2, really)
ln -sf ../usr/share/zoneinfo/UTC /mnt/etc/localtime

# Several modules for initcpio image. xen-*, button, and ipmi-* ones are for EC2.
MODULES="button ipmi-msghandler ipmi-poweroff"

# Support for Xen guest
MODULES+=" xen-blkfront xen-netfront xen-pcifront xen-privcmd"

# Support for SR-IOV on EC2
MODULES+=" ixgbevf"

# Apply our module changes.
sed -ri "s/^MODULES=.*/MODULES=\"$MODULES\"/g" /mnt/etc/mkinitcpio.conf

# Our root volume can vary in size, we should include the growfs hook.
sed -ri 's/^(HOOKS=".*filesystems)(.*)/\1 growfs\2/g' /mnt/etc/mkinitcpio.conf

# Prepare to chroot
cp /etc/resolv.conf /mnt/etc/

# Drop all the repo definitions, including the commented ones. We're filling
# this ourselves.
sed -i '/# uncommented to enable the repo./q' /mnt/etc/pacman.conf
cat >> /mnt/etc/pacman.conf << "EOF"
#
EOF

# EC2-specific package repository
cat >> /mnt/etc/pacman.conf <<"EOF"

[ec2]
SigLevel = PackageRequired
Server = https://www.uplinklabs.net/repo/ec2/$arch

[bbs]
SigLevel = PackageRequired
Server = https://blackbird-software.s3.amazonaws.com/

EOF

# Stock Arch Linux repositories
cat >> /mnt/etc/pacman.conf <<"EOF"

[core]
SigLevel = PackageRequired
Include = /etc/pacman.d/mirrorlist

[extra]
SigLevel = PackageRequired
Include = /etc/pacman.d/mirrorlist

[community]
SigLevel = PackageRequired
Include = /etc/pacman.d/mirrorlist
EOF

# Import uplinklabs.net and blackbird.cx GPG keys into pacman so that packages from these repos can be
# installed.
import_gpg_key /mnt
import_gpg_key_bbcx /mnt

arch-chroot /mnt /bin/bash -c "pacman -Sy"

# Install EC2 GPG signing keys
arch-chroot /mnt /bin/bash -c "pacman --noconfirm -S bbs-keyring ec2-keyring && (pkill gpg-agent || true)"

# Fix gnupg now includes dirmngr
arch-chroot /mnt /bin/bash -c "yes | pacman -Syu gnupg"

# Install any package updates we have overrides for in the EC2 repo.
#arch-chroot /mnt /bin/bash -c "pacman --noconfirm -Suu"

# Base packages needed for a working installation:
#   - cronie - cron daemon, not in base install anymore
#   - irqbalance - helps most on many-core instance types with IRQ throughput
#   - lrzip - eventually will be used for packaging
#   - openssh - needed for obvious reasons (how else are you going to use the
#               instance?)
#   - rng-tools - initial pacman keychain generation takes forever on
#                 a headless server unless you have something like rng-tools to
#                 fill up the entropy pool.

arch-chroot /mnt /bin/bash -c "pacman --needed --noconfirm -S audit cronie irqbalance lrzip openssh rng-tools"

# Convenience packages:
#   - rsync - frequently used to get files between hosts (or from hosts to
#             instances)
#   - vim - plain old vi just won't do for me.
arch-chroot /mnt /bin/bash -c "pacman --needed --noconfirm -S rsync vim"

arch-chroot /mnt /bin/bash -c "pacman --noconfirm -S ec2-pacman-mirrors"

arch-chroot /mnt /bin/bash -c "pacman --noconfirm -S mkinitcpio-growrootfs"

# We do a systemd-only installation with sysv compatibility.
arch-chroot /mnt /bin/bash -c "pacman --needed --noconfirm -S systemd-sysvcompat"

# Remove the Arch kernel, install the appropriate kernel variant for our target
# image.
if [ $KVARIANT ]; then
	arch-chroot /mnt /bin/bash -c "pacman --noconfirm -Rdd linux"
	arch-chroot /mnt /bin/bash -c "pacman --noconfirm -S linux${KVARIANT} linux${KVARIANT}-headers"
	rm -rf /mnt/boot/initramfs-linux{,-fallback}.img /mnt/boot/vmlinuz-linux /mnt/lib/modules/*ARCH
fi

arch-chroot /mnt /bin/bash -c "mkinitcpio -p linux${KVARIANT}"

# CoreOS & Blackbird additional packages
arch-chroot /mnt /bin/bash -c "pacman --noconfirm -Sy rkt coreos-cloudinit-git update-ssh-keys kubernetes etcd docker net-tools wget dnsutils conntrack-tools ethtool libmicrohttpd git python-aws-cli"

# Upgrade the rolling release
arch-chroot /mnt /bin/bash -c "pacman -Syu --noconfirm"

# Install the bootloader used in HVM. For PV, we just use the pv-grub AKI.
if [ $EFI_BOOT ]; then
	GRUB_TARGET=x86_64-efi
else
	GRUB_TARGET=i386-pc
fi
arch-chroot /mnt /bin/bash -c "grub-install --target=$GRUB_TARGET --recheck ${DEV}"

# Kernel flags which provide console logging and decent verbosity. Kernel
# modesetting doesn't make sense in a headless environment, so might as well
# turn that off here.
# Note also that these settings make the EC2 GetConsoleOutput API provide
# something meaningful in our image:
# http://docs.aws.amazon.com/AWSEC2/latest/APIReference/ApiReference-query-GetConsoleOutput.html
KERNEL_FLAGS="nomodeset console=ttyS0,9600n8 earlyprintk=serial,ttyS0,9600,verbose loglevel=7"
PV_KERNEL_FLAGS="nomodeset console=hvc0 earlyprintk=xen,verbose loglevel=7"

# Set sane options for grub so that we boot in a timely fashion and correctly.
sed -ri 's/GRUB_TIMEOUT=5/GRUB_TIMEOUT=1/' /mnt/etc/default/grub
sed -ri 's/^#GRUB_TERMINAL_OUTPUT/GRUB_TERMINAL_OUTPUT/' /mnt/etc/default/grub
sed -ri "s/^GRUB_CMDLINE_LINUX_DEFAULT.*/GRUB_CMDLINE_LINUX_DEFAULT=\"$KERNEL_FLAGS\"/g" /mnt/etc/default/grub
sed -ri '/^GRUB_TIMEOUT/a GRUB_DISABLE_SUBMENU=y' /mnt/etc/default/grub

# Trigger udev to create /dev/disk/by-uuid symlinks. Not sure why this gets
# *missed* in systemd 220 and later...
udevadm trigger

# Generate our grub configs for native grub and pv-grub.
arch-chroot /mnt /bin/bash -c "grub-mkconfig > /boot/grub/grub.cfg"

cat > /mnt/boot/grub/menu.lst << EOF
#
# This file is only used on paravirtualized instances.
#

timeout 1
default 0
color   light-blue/black light-cyan/blue

title  Arch Linux
root   (hd0)
kernel /boot/vmlinuz-linux${KVARIANT} root=/dev/xvda1 ro rootwait rootfstype=ext4 $PV_KERNEL_FLAGS
initrd /boot/initramfs-linux${KVARIANT}.img
EOF

# Create an /etc/rc.local file
touch /mnt/etc/rc.local
chmod 0755 /mnt/etc/rc.local

cat >> /mnt/etc/rc.local << "EOF"
#!/bin/bash

if [ ! -d /root/.ssh ] ; then
	mkdir -p /root/.ssh
	chmod 700 /root/.ssh
fi

EOF

# Make sure that the screen is cleared on user logout.
cat >> /mnt/etc/bash.bash_logout << "EOF"
clear
EOF

cat >> /mnt/etc/rc.local << "EOF"
function curl_retry {
	OUTPUT=
	attempts=0
	while [ -z "$OUTPUT" ]; do
		[ $attempts -ge 20 ] && return 1
		OUTPUT="$(curl -s $1)"
		[ ! -z "$OUTPUT" ] && break
		sleep 3
		let attempts+=1
	done
	echo "$OUTPUT"
	return 0
}

EOF

cat >> /mnt/etc/rc.local << "EOF"
#
# The user's public SSH key is available via instance metadata API.
#
KEY=$(curl_retry http://169.254.169.254/latest/meta-data/public-keys/0/openssh-key)
if [ $? -eq 0 ]; then
	if ! grep -q "^${KEY}$" /root/.ssh/authorized_keys 2>/dev/null; then
		echo "${KEY}" >> /root/.ssh/authorized_keys
		chmod 0644 /root/.ssh/authorized_keys
	fi
fi

EOF

cat >> /mnt/etc/rc.local << "EOF"
#
# Find the optimal mirrors for the current region
#
if [ ! -e /etc/pacman.d/mirrorlist ]; then
	REGION=$(curl_retry http://169.254.169.254/2009-04-04/dynamic/instance-identity/document | grep region | cut -d'"' -f 4)
	MIRRORLIST=/etc/pacman.d/ec2-mirrors-${REGION}
	if [ -z $REGION ] || [ ! -f $MIRRORLIST ]; then
		MIRRORLIST=/etc/pacman.d/ec2-mirrors-default
	fi
	ln -s $MIRRORLIST /etc/pacman.d/mirrorlist
	rm -f /var/lib/pacman/sync/*
fi
EOF

cat >> /mnt/etc/rc.local << "EOF"
#
# If a pacman keyring isn't already set up, create one and locally sign the
# trusted keys. Note that this also includes the key used for signing the EC2
# repository, which isn't an officially trusted key upstream.
#
PACMAN_KEYRING_DIR=/etc/pacman.d/gnupg
if [ ! -f ${PACMAN_KEYRING_DIR}/secring.gpg ]; then
	GPG_PACMAN=(gpg --homedir "${PACMAN_KEYRING_DIR}" --no-permission-warning)
	KEYRING_IMPORT_DIR='/usr/share/pacman/keyrings'
	pacman-key --init

	# pacman-key --populate gives interactive prompts, which doesn't make
	# sense on a headless server. Instead, manually import keys and batch
	# sign them. This is kind of messy, but we want the host to be usable
	# as soon as the user logs on.
	for KEYRING in archlinux ec2 bbs; do
		"${GPG_PACMAN[@]}" --import "${KEYRING_IMPORT_DIR}/${KEYRING}.gpg"
		"${GPG_PACMAN[@]}" --import-ownertrust "${KEYRING_IMPORT_DIR}/${KEYRING}-trusted"
		for KEY in $(cat "${KEYRING_IMPORT_DIR}/${KEYRING}-trusted" | cut -d':' -f 1); do
			yes | "${GPG_PACMAN[@]}" --command-fd 0 --status-fd 2 --batch --lsign-key "$KEY"
		done
	done
fi

EOF

sed -ri 's/^#Servers.*$/Servers=0.pool.ntp.org 1.pool.ntp.org 2.pool.ntp.org 3.pool.ntp.org/g' /mnt/etc/systemd/timesyncd.conf

cat > /mnt/etc/vconsole.conf << "EOF"
KEYMAP=us
FONT=LatArCyrHeb-14
EOF

cat > /mnt/etc/locale.conf << "EOF"
LANG=en_US.utf8
EOF

# Makes it so bash doesn't behave stupidly if you resize your terminal windows.
cat >> /mnt/etc/bash.bashrc << "EOF"
shopt -s checkwinsize
EOF

SERVICES="cronie rngd rc-local irqbalance systemd-timesyncd"
SOCKETS="sshd"

# Add a goofy rc.local service which will allow us to fetch an SSH public key
# and generate the pacman keychain at boot.
cat > /mnt/etc/systemd/system/rc-local.service <<-EOF
[Unit]
Description=/etc/rc.local Compatibility
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/etc/rc.local
TimeoutSec=0
#StandardInput=tty
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

# DHCP MTU should be respected in EC2.
sed -ri 's/#(option interface_mtu)/\1/g' /mnt/etc/dhcpcd.conf

cat >> /mnt/etc/dhcpcd.conf <<EOF

# Disable ARP probes, allows us to take our lease sooner.
noarp
EOF

# Bring up eth0 at boot.
cat > /mnt/etc/systemd/network/ethernet.network <<- "EOF"
[Match]
Name=en*
[Network]
DHCP=both
[DHCP]
UseMTU=yes
UseDNS=yes
EOF

cat > /mnt/etc/systemd/network/virtual-eth0.network <<- "EOF"
[Match]
Name=eth*
[Network]
DHCP=both
[DHCP]
UseMTU=yes
UseDNS=yes
EOF

# systemd-networkd is incorrectly symlinked in the default install. Disable and re-enable it.
arch-chroot /mnt /bin/bash -c "systemctl disable systemd-networkd.service"
SERVICES+=" systemd-networkd systemd-resolved"

# Now enable all services relevant to our needs
for SERVICE in $SERVICES; do
	arch-chroot /mnt /bin/bash -c "systemctl enable ${SERVICE}.service"
done
for SOCKET in $SOCKETS; do
	arch-chroot /mnt /bin/bash -c "systemctl enable ${SOCKET}.socket"
done

# Change default from 'graphical' to 'multi-user'
ln -sf ../../../../usr/lib/systemd/system/multi-user.target /mnt/etc/systemd/system/default.target

# Disable password authentication. It doesn't make sense in a cloud setting.
sed -ri 's/^#PasswordAuthentication yes/PasswordAuthentication no/' /mnt/etc/ssh/sshd_config

# The default /usr/bin/pinentry is linked to /usr/bin/pinentry-gtk, which is
# highly unlikely to be used in EC2. Switch to pinentry-curses for sanity's
# sake.
ln -sf /usr/bin/pinentry-curses /mnt/usr/bin/pinentry

# Switch to S3 repo
sed -ri 's#^Server = https://www.uplinklabs.net/repo/ec2/.*#Server = https://arch-linux-ami.s3.amazonaws.com/repo/$arch#g' /mnt/etc/pacman.conf
sed -ri 's#^Server = https://www.uplinklabs.net/repo/.*#Server = https://arch-linux-ami.s3.amazonaws.com/repo/$repo/$arch#g' /mnt/etc/pacman.conf

# Clean up all the junk we've left behind. The space occupied by these files
# will be discarded via fstrim later in the build process.
rm -f /mnt/etc/pacman.d/mirrorlist
find /mnt/var/lib/pacman/sync -type f -print0 | xargs -0 rm -fv

rm -f /mnt/etc/resolv.conf
ln -s ../run/systemd/resolve/resolv.conf /mnt/etc/resolv.conf

rm -rf /mnt/etc/pacman.d/gnupg/*
find /mnt/var/cache/pacman/pkg -type f -print0 | xargs -0 rm -fv
find /mnt/var/log -type f -print0 | xargs -0 rm -fv
umount /mnt
