FROM alpine:3.7

MAINTAINER Burkhard Noltensmeier <bn@teuto.net>

LABEL Description="openstack client, Ansible, Shade" Version="0.2"

# Alpine-based installation
# #########################
RUN apk --update add \
  python-dev=2.7.14-r2 \
  py-pip=9.0.1-r1 \
  py-setuptools=33.1.1-r1 \
  make=4.2.1-r0 \
  openssl-dev=1.0.2n-r0 \
  openssl=1.0.2n-r0 \
  libffi-dev=3.2.1-r4 \
  ca-certificates \
  gcc=6.4.0-r5 \
  musl-dev=1.1.18-r3 \
  linux-headers=4.4.6-r2 \
  git==2.15.0-r1 \
  openssh-client=7.5_p1-r8 \ 
  openssh-server=7.5_p1-r8 \
  iptables=1.6.1-r1 \
  && pip install shade==1.26 \
  && pip install ansible==2.4.2 \
  && pip install kubespray==0.5.2 \
  && pip install six==1.10.0  pbr==3.1.1 Babel==2.5.0 oslo.i18n==3.17.0 oslo.utils==3.28.0 oslo.log==3.36.0 keystoneauth1==3.2.0 cliff==2.8.0 osc-lib==1.7.0 python-cinderclient==3.1.0 python-neutronclient==6.1.0 python-novaclient==9.1.0 python-glanceclient==2.8.0 python-keystoneclient==3.13.0 openstacksdk==0.9.18 sshuttle==0.76 \
  && pip install --no-cache-dir pip setuptools==38.4 python-openstackclient==3.12 pyyaml==3.12 \
  && apk del gcc musl-dev linux-headers \
  && rm -rf /var/cache/apk/*
  
RUN wget https://storage.googleapis.com/kubernetes-release/release/v1.9.2/bin/linux/amd64/kubectl -O /usr/bin/kubectl \
  && chmod +x /usr/bin/kubectl
RUN wget https://storage.googleapis.com/kubernetes-helm/helm-v2.8.1-linux-amd64.tar.gz -O /root/helm_v2.8.1.tar.gz \
  && tar -zxvf /root/helm_v2.8.1.tar.gz -C /root/ \
  && mv /root/linux-amd64/helm /usr/bin/ \
  && rm -rf /root/helm*
VOLUME ["/data"]


## Clone kubespray-incubator repository and change some variables: Enable Helm, update right socket locations, prepare for openstack cloud provider.
RUN mkdir -p /opt/kubespray \
  && git clone -n https://github.com/kubernetes-incubator/kubespray.git /opt/kubespray/\
  && cd /opt/kubespray \
  && git checkout 50e5f0d28b9cf2b9c28339ba469f39671aa05946\
  && echo "callback_whitelist = profile_tasks" >> /opt/kubespray/ansible.cfg \ 
  && sed -i 's*local_release_dir: "/tmp/releases"*local_release_dir: "/tmp"*g' /opt/kubespray/roles/kubespray-defaults/defaults/main.yaml \
  && sed -i 's*#control_path = ~/.ssh/ansible-%%r@%%h:%%p*control_path = /dev/ansible-%%r@%%h:%%p*g' /opt/kubespray/ansible.cfg\
#  && sed -i 's*kubeadm_enabled: false*kubeadm_enabled: true*g' /opt/kubespray/roles/kubespray-defaults/defaults/main.yaml\
  && sed -i 's*helm_enabled: false*helm_enabled: true*g' /opt/kubespray/roles/kubespray-defaults/defaults/main.yaml\
  && sed -i 's*helm_enabled: false*helm_enabled: true*g' /opt/kubespray/inventory/local/group_vars/k8s-cluster.yml\
  && sed -i 's*helm_enabled: false*helm_enabled: true*g' /opt/kubespray/roles/kubernetes-apps/helm/defaults/main.yml\
  && sed -i 's*bootstrap_os: none*bootstrap_os: ubuntu*g' /opt/kubespray/inventory/local/group_vars/all.yml\
  && sed -i 's*#cloud_provider:*cloud_provider: openstack*g' /opt/kubespray/inventory/local/group_vars/all.yml\
  && sed -i 's*resolvconf_mode: docker_dns*resolvconf_mode: host_resolvconf*g' /opt/kubespray/inventory/local/group_vars/k8s-cluster.yml\
  ###########################################################################
  #we need to change the name of /etc/kubernetes/cloud_config to cloud-config
  ###########################################################################
#   && sed -i 's*/cloud_config*/cloud-config*g' /opt/kubespray/roles/kubernetes/node/templates/kubelet.standard.env.j2 /opt/kubespray/roles/kubernetes/master/templates/manifests/kube-apiserver.manifest.j2 /opt/kubespray/roles/kubernetes/preinstall/tasks/main.yml /opt/kubespray/roles/kubernetes/node/templates/kubelet.kubeadm.env.j2 /opt/kubespray/roles/kubernetes/master/templates/manifests/kube-controller-manager.manifest.j2\
#  && sed -i 's*dashboard_enabled: true*dashboard_enabled: false*g' /opt/kubespray/inventory/group_vars/k8s-cluster.yml\
#  && sed -i 's*dashboard_enabled: true*dashboard_enabled: false*g' /opt/kubespray/roles/kubernetes-apps/ansible/defaults/main.yml\
  && sed -i "/#  - 8.8.4.4/a upstream_dns_servers:\n\  - 8.8.8.8"  /opt/kubespray/inventory/local/group_vars/all.yml\
  && sed -i "/- name: Kubernetes Apps | Start Resources/i - name: Wait for api to catch up...\n  wait_for:\n  port: 6443\n  timeout: 6\n  register: status\n  until: status|success\n  retries: 10\n  delay: 3\n  ignore_errors: yes\n"\
  /opt/kubespray/roles/kubernetes-apps/ansible/tasks/main.yml \
#  && sed -i "/- name: Kubernetes Apps | Start Resources/i - name: Pause for a moment to let api catch up.\n  pause:\n    seconds: 15\n"\
#  /opt/kubespray/roles/kubernetes-apps/ansible/tasks/main.yml
#  && sed -i "/  with_items: \"{{ manifests.results }}\"/i \  retries: 3\n  delay: 6"\
#  /opt/kubespray/roles/kubernetes-apps/ansible/tasks/main.yml
  && sed -i "/- name: Kubernetes Apps | Start Resources/i - name: Pause for a moment to let api catch up.\n  pause:\n    seconds: 20\n"\
  /opt/kubespray/roles/kubernetes-apps/ansible/tasks/main.yml
  
#- name: Master | wait for the apiserver to be running
#  uri:                                 
#    url: "{{ kube_apiserver_endpoint }}/healthz"
#    validate_certs: no      
#    client_cert: "{{ kube_apiserver_client_cert }}"
#    client_key: "{{ kube_apiserver_client_key }}"
#  register: result
#  until: result.status == 200                   
#  retries: 20                                    
#  delay: 6




###### Tmp. We going to switch to ubuntu later.
RUN echo "kubespray_path: \"/opt/kubespray\"" >> /root/.kubespray.yml
RUN  echo "    IdentityFile /root/id_rsa" >> /etc/ssh/ssh_config
######



# Default is to start a shell.  A more common behavior would be to override
# the command when starting.
## Create new user ubuntu.
#RUN addgroup ubuntu \
#  && adduser -h /home/ubuntu -s /bin/sh -G ubuntu -D ubuntu

# User Ubuntu should use kubespray
#RUN chown ubuntu:ubuntu -R /opt/kubespray \
#  && cp /root/.kubespray.yml /home/ubuntu/ \
#  && echo "kubespray_path: \"/opt/kubespray\"" >> /home/ubuntu/.kubespray.yml \
#  && chown ubuntu:ubuntu /home/ubuntu/.kubespray.yml
#RUN  echo "    IdentityFile /home/ubuntu/id_rsa" >> /etc/ssh/ssh_config

## Add version
ADD VERSION /tools/VERSION

## Add function-handler
ADD scriptweiche.sh /usr/bin/scriptweiche

## Add deploy function.
ADD deploy.sh /usr/bin/deploy

## Add further functions
# Cleanup Cluster Function
ADD cleanup.sh /usr/bin/cleanup
# Copy certs of edge-node to configure remote access
ADD aftermath.yml /tools/aftermath.yml
# Configure kubectl for remote access
ADD remote_kubectl.sh /usr/bin/remote_kubectl
# Install needed python packages etc..
ADD groundwork.yml /tools/groundwork.yml
# Make sure helm is installed and usable
ADD helm_n_tiller.yml /tools/helm_n_tiller.yml
# go-binary, userdata and deploy.example
ADD teutonetes /tools/teutonetes
ADD userdata-ssh /root/userdata-ssh
ADD deploy.example /root/deploy.example
ADD join.sh /usr/bin/join
ADD generate_inventory.sh /tools/generate_inventory.sh


# Scaling functions
ADD netesscale.sh /usr/bin/netesscale

## Tell ansible which config file it should use.
ENV ANSIBLE_CONFIG /opt/kubespray/ansible.cfg

ENV PROFILE_TASKS_SORT_ORDER none
ENV PROFILE_TASKS_TASK_OUTPUT_LIMIT all

## Run user ubuntu.
#USER ubuntu
#WORKDIR /home/ubuntu
  
## Autostart
  
# Shell-start
CMD ["/bin/sh"]
ENTRYPOINT ["/usr/bin/scriptweiche"]


