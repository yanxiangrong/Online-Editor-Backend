安装 Docker  
`yum install docker`  
开启 Docker  
`sudo systemctl start docker`  
构建镜像  
`sudo docker build -t online-editor-backend .`  
查看镜像  
`sudo docker images`  
运行  
`sudo docker run -p 9527:9527 -d online-editor-backend`  
查看运行实列  
`sudo docker ps -a`  
进入实列  
`sudo docker exec -it "ID" /bin/bash`
