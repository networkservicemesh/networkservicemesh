#/bin/bash
dpkg -l | grep qemu-user-static
retVal=$?
if [ $retVal -ne 0 ]; then
    sudo apt-get install -y qemu-user-static
fi
cp /usr/bin/qemu-aarch64-static $1/test/applications/build/qemu-aarch64-static 
cp /usr/bin/qemu-aarch64-static $1/dataplane/vppagent/conf/vpp/qemu-aarch64-static
