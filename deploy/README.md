# 编译构建 
make image-build
make iamge-push

# 启动服务
kubectl apply  -f deploy/cubefs-dashboard.yaml

# 用户名密码 (admin/Admin@1234 原密码)
Dashboard
admin/Admin_123
