sudo yum check-update
sudo yum install httpd-tools
sudo firewall-cmd --zone=public --permanent --add-port=3000/tcp
sudo firewall-cmd --reload

https://stackoverflow.com/questions/24906097/apache-benchmark-error-too-many-open-files
https://www.robustperception.io/dealing-with-too-many-open-files
