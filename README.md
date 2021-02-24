yum install docker
sudo systemctl start docker


sudo docker build -t online-editor-backend .
sudo docker images
sudo docker run -p 9527:9527 -d online-editor-backend
sudo docker ps -a
sudo docker exec -it 327a /bin/bash
