FROM centos:latest

#
# These packages are optional and used only for debugging
#
RUN yum -y update \
 && yum -y install pciutils \
 && yum -y install net-tools \
 && yum -y install sysfsutils \
 && yum -y install iproute \
 && yum clean all \
 && rm -rf /var/cache/yum

#
# vfio is testing binary, it tests liveness of vfio device
#
 COPY ./bin/vfio /vfio
 COPY entrypoint.sh /entrypoint.sh
 RUN chmod +x /vfio
 RUN chmod +x /entrypoint.sh
 #
 # Executing script which will change ulimit values
 # once completed it will call Docker's original entrypoint
 #
ENTRYPOINT ["/entrypoint.sh"]

 
