echo 拉取依赖
go mod vendor

cp -n ./etc/app.ini.example ./etc/app.ini

echo 开始构建版本
go build -o micro-mall-order-consumer main.go