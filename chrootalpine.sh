#!/bin/bash -ex

dev=/dev/xvdx
dev_dir=/bootstrap
final_stage=/mnt
mirror=http://dl-4.alpinelinux.org/alpine
branch=v3.3

pacman -Sy --noconfirm parted arch-install-scripts syslinux

(
	mkdir -p $dev_dir
	cd $dev_dir

	if ! test -f apk-tools-static-2.6.5-r1.apk; then
		curl -L $mirror/latest-stable/main/x86_64/apk-tools-static-2.6.5-r1.apk -o apk-tools-static-2.6.5-r1.apk
		tar -xzvf apk-tools-static-2.6.5-r1.apk
	fi

  $dev_dir/sbin/apk.static -X ${mirror}/latest-stable/main -U --allow-untrusted --root ${dev_dir} --initdb add alpine-base

	mkdir -p etc/apk
	mkdir -p proc sys dev tmp
	touch etc/resolv.conf

	echo "${mirror}/${branch}/main" > etc/apk/repositories
	echo "${mirror}/${branch}/community" >> etc/apk/repositories
	# echo "${mirror}/${branch}/testing" >> etc/apk/repositories

	arch-chroot $dev_dir /sbin/apk.static update
	arch-chroot $dev_dir /sbin/apk.static add bash bkeymaps curl

	cat <<EOF> $dev_dir/alpine-answers
KEYMAPOPTS="us us"
HOSTNAMEOPTS="-n alpine-ec2"
INTERFACESOPTS="auto lo
iface lo inet loopback

auto eth0
iface eth0 inet dhcp
	hostname alpine-ec2
"
DNSOPTS="-d example.com 8.8.8.8"
TIMEZONEOPTS="-z UTC"
PROXYOPTS="none"
APKREPOSOPTS="-r"
SSHDOPTS="-c openssh"
NTPOPTS="-c busybox"
DISKOPTS="-m sys $dev"
NOCOMMIT="no"
ERASE_DISKS="-m sys $dev"
EOF

	cat <<EOF> /bootstrap/fixpath-runsetup.sh
#!/bin/sh
export PATH=${PATH}:/bin:/sbin
/sbin/setup-alpine -f /alpine-answers
EOF
	chmod +x /bootstrap/fixpath-runsetup.sh

	# fix setup-disk for non uname detect
	sed -i 's%kver=$(uname -r)%kver=grsec%' /bootstrap/sbin/setup-disk
	# manually disable prompting TODO: figure out how to do this with ERASE_DISKS answer
	sed -i 's%local erasedisks="$@"%return 0%' /bootstrap/sbin/setup-disk
  # Then run arch-chroot /setup /sbin/setup-alpine
	arch-chroot $dev_dir /fixpath-runsetup.sh

	#
	## finalize setup
	#
	umount -l /bootstrap/proc
	umount -l /bootstrap/dev
	umount -l /bootstrap/sys
	mount /dev/xvdx3 $final_stage

	cat <<EAR> $final_stage/etc/local.d/00_populate_ssh_authorized_keys.start
#!/bin/bash -e

if [ ! -d /root/.ssh ] ; then
        mkdir -p /root/.ssh
        chmod 700 /root/.ssh
fi

function curl_retry {
        OUTPUT=
        attempts=0
        while [ -z "\$OUTPUT" ]; do
                [ \$attempts -ge 30 ] && return 1
                OUTPUT="\$(curl -s \$1)"
                	[ ! -z "\$OUTPUT" ] && break
                sleep 10
                let attempts+=1
        done
        echo "\$OUTPUT"
        return 0
}

KEY=\$(curl_retry http://169.254.169.254/latest/meta-data/public-keys/0/openssh-key)
if [ \$? -eq 0 ]; then
        if ! grep -q "^\${KEY}\$" /root/.ssh/authorized_keys 2>/dev/null; then
                echo "\${KEY}" >> /root/.ssh/authorized_keys
                chmod 0644 /root/.ssh/authorized_keys
        fi
fi
EAR
	chmod +x $final_stage/etc/local.d/00_populate_ssh_authorized_keys.start
	arch-chroot $final_stage /sbin/rc-update add local default

	# setup sshd config
	cat << EOF > $final_stage/etc/ssh/sshd_config
Port 22
PubkeyAuthentication yes
AuthorizedKeysFile	.ssh/authorized_keys
PasswordAuthentication no
PermitRootLogin without-password
ChallengeResponseAuthentication no
Subsystem	sftp	/usr/lib/ssh/sftp-server
EOF
	# TODO: re-generate host keys

	umount $final_stage

)
