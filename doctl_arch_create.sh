#!/bin/bash
if doctl compute droplet list --format "Name" | grep "manager1"; then
    echo "Error: Droplet with name manager1 already exists."
    exit 1
fi


# define fingerprints
ssh_key_fingerprints="ea:e6:b9:88:5a:8f:d6:4b:c7:03:06:c2:fd:aa:ca:a8,3d:83:40:84:6a:ee:3b:e4:da:e6:b8:c9:1e:4f:24:e5,58:15:a0:48:24:47:dc:79:af:80:1b:f9:e3:0e:ff:67,cd:82:77:0a:e5:c3:b4:b2:06:91:30:b6:b3:60:f0:76,ee:36:4b:17:af:40:f4:0c:00:8e:bf:a9:46:ab:df:e4"
echo "ssh_key_fingerprints: $ssh_key_fingerprints"

env_file_path="itu-minitwit/backend/.env"

# check if file exists
if [ ! -f "$env_file_path" ]; then
    echo "Error: File $env_file_path does not exist."
    exit 1
fi

# create droplets
echo "creating droplets"
doctl compute droplet create --image ubuntu-22-10-x64 --size s-2vcpu-4gb-amd --region fra1 --enable-monitoring manager1 --ssh-keys "$ssh_key_fingerprints"

check_doctl_command() {
    doctl compute droplet get manager1 --format PublicIPv4 --no-header >/dev/null 2>&1
}

# Loop to get the Manager1 IP until it succeeds and has length greater than 2
while ! check_doctl_command || [[ ${#manager1_ip} -le 2 ]]; do
    echo "waiting 5 more seconds to see if droplet is ready... "
    sleep 5  # Wait for 5 seconds before retrying
    manager1_ip=$(doctl compute droplet get manager1 --format PublicIPv4 --no-header)
done

# print IP addresses
echo "Manager1 IP address: $manager1_ip"

#Create DNS records on digital ocean
doctl compute domain records create cicdont.live --record-type A --record-name @ --record-data $manager1_ip --record-ttl 1800
doctl compute domain records create cicdont.live --record-type A --record-name elasticsearch --record-data $manager1_ip --record-ttl 1800
doctl compute domain records create cicdont.live --record-type A --record-name logs --record-data $manager1_ip --record-ttl 1800
doctl compute domain records create cicdont.live --record-type A --record-name grafana --record-data $manager1_ip --record-ttl 1800
doctl compute domain records create cicdont.live --record-type A --record-name api --record-data $manager1_ip --record-ttl 1800
doctl compute domain records create cicdont.live --record-type A --record-name simulator --record-data $manager1_ip --record-ttl 1800
doctl compute domain records create cicdont.live --record-type A --record-name prometheus --record-data $manager1_ip --record-ttl 1800
doctl compute domain records create cicdont.live --record-type A --record-name sla --record-data $manager1_ip --record-ttl 1800


# add droplets to known hosts
sleep 1
ssh-keyscan -H $manager1_ip >> ~/.ssh/known_hosts
sleep 1

# Check that all dropslets are added to known hosts otherwise run again
# add droplets to known hosts
for ip in $manager1_ip; do
    while ! ssh-keygen -F $ip | grep -q $ip; do
        echo "Adding $ip to known hosts failed...retrying"
        sleep 5
        ssh-keyscan -H $ip >> ~/.ssh/known_hosts
    done
done

# install docker engine on droplets
doctl compute ssh manager1 --ssh-command "curl -fsSL https://get.docker.com -o get-docker.sh"

# get tokens from .env file
echo "trying to get tokens from .env file"
source "$env_file_path"
echo "manager_token: $manager_token"

echo "installing doctl on droplet"
doctl compute ssh manager1 --ssh-command "sudo snap install doctl"
echo "manager1 doctl auth init with token"
doctl compute ssh manager1 --ssh-command "sudo mkdir /root/.config"
doctl compute ssh manager1 --ssh-command "sudo snap connect doctl:dot-docker"
doctl compute ssh manager1 --ssh-command "sudo doctl auth init -t $manager_token"
echo "manager1 doctl registry login"
doctl compute ssh manager1 --ssh-command "sudo doctl registry login"

# get docker compose file on manager 1
echo "trying to get docker compose file on manager1"
doctl compute ssh manager1 --ssh-command "curl https://raw.githubusercontent.com/DevOps-CI-CDont/DevOps-CI-CDont/main/docker-compose-IaC.yml --output ./docker-compose.yml"
doctl compute ssh manager1 --ssh-command "curl https://raw.githubusercontent.com/DevOps-CI-CDont/DevOps-CI-CDont/main/itu-minitwit/nginx.conf --output ./nginx.conf"
doctl compute ssh manager1 --ssh-command "curl https://raw.githubusercontent.com/DevOps-CI-CDont/DevOps-CI-CDont/main/itu-minitwit/filebeat.yml --output ./filebeat.yml"
doctl compute ssh manager1 --ssh-command "curl https://raw.githubusercontent.com/DevOps-CI-CDont/DevOps-CI-CDont/main/.htpasswd --output ./.htpasswd"
doctl compute ssh manager1 --ssh-command "curl https://raw.githubusercontent.com/DevOps-CI-CDont/DevOps-CI-CDont/main/itu-minitwit/prometheus.yml --output ./prometheus.yml"
doctl compute ssh manager1 --ssh-command "mkdir grafana"
doctl compute ssh manager1 --ssh-command "mkdir grafana/dashboards"
doctl compute ssh manager1 --ssh-command "curl https://raw.githubusercontent.com/DevOps-CI-CDont/DevOps-CI-CDont/main/itu-minitwit/grafana/dashboards/main-dashboard.json --output ./grafana/dashboards/main-dashboard.json"
doctl compute ssh manager1 --ssh-command "curl https://raw.githubusercontent.com/DevOps-CI-CDont/DevOps-CI-CDont/main/itu-minitwit/grafana/dashboard.yaml --output ./grafana/dashboard.yaml"
doctl compute ssh manager1 --ssh-command "curl https://raw.githubusercontent.com/DevOps-CI-CDont/DevOps-CI-CDont/main/itu-minitwit/grafana/datasources.yaml --output ./grafana/datasources.yaml"


echo "installing docker on droplet"
doctl compute ssh manager1 --ssh-command "sh get-docker.sh"
# Function to check if the 'docker' command is recognized on the droplet
check_docker_command() {
    doctl compute ssh manager1 --ssh-command "docker version >/dev/null 2>&1"
}
# Check if the 'docker' command is recognized
while ! check_docker_command; do
    echo "Docker command not recognized. Rerunning the script..."
    doctl compute ssh manager1 --ssh-command "sh get-docker.sh"
done
scp $env_file_path root@$manager1_ip:./.env
doctl compute ssh manager1 --ssh-command "docker compose up -d"

doctl compute ssh manager1 --ssh-command "sudo iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 3000"

