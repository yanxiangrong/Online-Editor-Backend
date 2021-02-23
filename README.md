yum install docker
sudo systemctl start docker


sudo docker build -t online-editor-backend .
sudo docker images
sudo docker run -d online-editor-backend
sudo docker ps -a
