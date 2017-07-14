#!/bin/bash
NginxUrl=http://nginx.org/packages/centos/7/noarch/RPMS/nginx-release-centos-7-0.el7.ngx.noarch.rpm
dbname=${db_name}
dbuser=${db_user}
dbpassword=${db_pwd}
dbrootpassword=${db_root_pwd}
export HOME=/root
export HOSTNAME=`hostname`
systemctl stop firewalld.service
systemctl disable firewalld.service
sed -i 's/^SELINUX=/# SELINUX=/' /etc/selinux/config
sed -i '/# SELINUX=/a SELINUX=disabled' /etc/selinux/config
setenforce 0
yum install yum-priorities -y
yum -y install aria2
aria2c $NginxUrl
rpm -ivh nginx-*.rpm
yum -y install nginx
systemctl start nginx.service
systemctl enable nginx.service
yum -y install php-fpm
systemctl start php-fpm.service
systemctl enable php-fpm.service
sed -i '/FastCGI/,/htaccess/s/    #/    /' /etc/nginx/conf.d/default.conf
sed -i '/FastCGI/s/^    /    #/' /etc/nginx/conf.d/default.conf
sed -i '/htaccess/s/^    /    #/' /etc/nginx/conf.d/default.conf
sed -i '/SCRIPT_FILENAME/s/\/scripts/\/usr\/share\/nginx\/html\//' /etc/nginx/conf.d/default.conf
yum -y install mariadb mariadb-server
systemctl start mariadb.service
systemctl enable mariadb.service
yum -y install php php-mysql php-gd libjpeg* php-ldap php-odbc php-pear php-xml php-xmlrpc php-mbstring php-bcmath php-mhash php-mcrypt
MDSRING=`find / -name mbstring.so`
echo extension=$MDSRING >> /etc/php.ini
systemctl restart mariadb.service
mysqladmin -u root password "$dbrootpassword"
$(mysql $dbname -u root --password="$dbrootpassword" >/dev/null 2>&1 </dev/null); (( $? != 0 ))
echo CREATE DATABASE $dbname \; > /tmp/setup.mysql
echo GRANT ALL ON $dbname.* TO "$dbuser"@"localhost" IDENTIFIED BY "'$dbpassword'" \; >> /tmp/setup.mysql
mysql -u root --password="$dbrootpassword" < /tmp/setup.mysql
$(mysql $dbname -u root --password="$dbrootpassword" >/dev/null 2>&1 </dev/null); (( $? != 0 ))
cd /root
systemctl restart php-fpm.service
systemctl restart nginx.service
echo \<?php >  /usr/share/nginx/html/test.php
echo \$conn=mysql_connect\("'127.0.0.1'", "'$dbuser'", "'$dbpassword'"\)\; >>  /usr/share/nginx/html/test.php
echo if \(\$conn\){ >>  /usr/share/nginx/html/test.php
echo   echo \"LNMP platform connect to mysql is successful\!\"\; >>  /usr/share/nginx/html/test.php
echo   }else{  >>  /usr/share/nginx/html/test.php
echo echo \"LNMP platform connect to mysql is failed\!\"\;  >>  /usr/share/nginx/html/test.php
echo }  >>  /usr/share/nginx/html/test.php
echo  phpinfo\(\)\;  >>  /usr/share/nginx/html/test.php
echo \?\>  >>  /usr/share/nginx/html/test.php