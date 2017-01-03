#! /bin/bash
apt-get update
apt-get install apache2 -y
a2ensite default-ssl
a2enmod ssl
service apache2 restart
INSTANCE_NAME=`curl -s -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/hostname | awk -F "." '{print $1}'`
ZONE=`curl -s -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/zone | awk -F "/" '{print $NF}'`
echo '<!doctype html><html><body><h1>'$INSTANCE_NAME'</h1></body></html>' | tee /var/www/html/index.html
gcloud compute instances delete-access-config $INSTANCE_NAME --zone $ZONE
