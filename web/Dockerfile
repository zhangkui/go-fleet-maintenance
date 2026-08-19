# syntax=docker/dockerfile:1
# 构建阶段：编译 Vue 3 SPA（node:22 与 nginx:stable 均为多架构镜像，
# 通过 docker buildx build --platform linux/amd64,linux/arm64 即可产出双架构镜像）
FROM node:22 AS build
WORKDIR /app

# 先拷贝依赖清单以利用 Docker 缓存
COPY package.json package-lock.json* ./
RUN npm ci || npm install

# 拷贝源码并构建
COPY . .
RUN npm run build

# 运行阶段：Nginx 托管静态资源
FROM nginx:stable AS serve
LABEL org.opencontainers.image.title="go-fleet-maintenance-web"
LABEL org.opencontainers.image.description="Vue 3 SPA for fleet maintenance"

# Nginx 配置
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 构建产物
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
