安装 Docker  
`yum install docker`  
开启 Docker  
`sudo systemctl start docker`  
构建镜像  
`sudo docker build --restart=always -t online-editor-backend .`  
查看镜像  
`sudo docker images`  
运行  
`sudo docker run -p 9527:9527 -d online-editor-backend`  
查看运行实列  
`sudo docker ps -a`  
进入实列  
`sudo docker exec -it "ID" /bin/bash`  
安装JDK和Python3   
`apt install python3 openjdk-11-jdk`
