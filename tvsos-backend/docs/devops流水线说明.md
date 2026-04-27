# DevOps流程说明

## 技术选型

  - **版本控制**：Git + GitHub  
    
    管理项目源代码，支持分支开发和PR（Pull Request）协作流程。
    
  - **持续集成/持续部署（CI/CD）**：Jenkins  
    
    自动化构建、测试、打包及部署，确保代码变更快速、安全地上线。
    
  - **容器化**：Docker + Docker Compose  
    
    将应用和依赖打包成容器镜像，保证环境一致性和可移植性。
    
  - **数据库**：MySQL  
    
    提供持久化数据存储，支持Docker网络隔离和动态IP访问控制。
    
  - **文档与接口管理**：Swagger（gin-swagger）  
    
    提供可交互的API文档，方便前后端协作和接口调试。

## 工作流程

  1. **本地开发**  
     - 在 `develop` 分支进行功能开发。  
     - 使用 `.env.dev` 配置本地开发环境变量。
     
  2. **代码提交与PR**  
     
     - 将本地开发完成的功能提交到远程 `feature` 分支。  
     - 提交 Pull Request (PR) 到 `develop` 分支，由团队成员进行代码审查。
     
  3. **CI/CD 流程**  
     
     - Jenkins 监控 `main` 分支，一旦有更新`webhook`就会触发流水线（我设置的逻辑是监控`main`分支上的`push`和`PR(merge)`，但`main`被设置为了禁止直接`push`，所以通过`PR`触发即可）。
     - 流水线步骤：
       1. 拉取最新代码。
       2. 使用 Docker 构建应用镜像。
       3. 推送镜像到 Docker Hub。
       4. 使用 Docker Compose 部署应用到目标服务器。
       5. 注入数据库及运行配置环境变量。
     
  4. **环境管理**  
     - **开发环境**：`.env.dev` 文件，供本地调试使用，由代码逻辑读取到环境变量中。
     - **生产环境**：`.env.prod` 文件，由 Jenkins 创建，并被 Docker Compose 读取，放入默认环境变量中。
     
     > 当项目根目录下".env.dev not found"，则会采取默认环境变量，因此需要把生产环境的配置放入默认环境变量中，保证开发环境与生产环境的代码一致性。
     
  5. **数据库访问控制**  
     - MySQL 服务直接运行在服务器上，容器内服务通过 docker0 虚拟网络访问宿主机。统一使用 kirito 用户对数据库进行CRUD操作，允许访问的 ip 段为`172.%.%.%`（ docker 网络）。
     
  6. **运行模式**  
     - Gin 框架在 **release** 模式下运行，减少日志输出，提高性能（配置在了 docker-compose.yml 中）。
     - 服务监听 `0.0.0.0:8088`，确保容器内外都能访问。
     
  7. **Swagger文档**  
     - 提供接口可视化文档，浏览器访问：
       ```
       http://<服务器IP>:8088/swagger/index.html
       ```

## 使用方式

  1. **开发人员**  
     - 拉取仓库代码，编辑 `.env.dev` 进行本地调试。  
     - 提交代码到 `feature` 分支并创建 PR。

  2. **运维/CI/CD**  

     - CI:
       - Jenkins 自动拉取 `main` 分支最新代码。
       - Jenkins 自动构建镜像并推送至 Docker Hub。
     - CD：
       - Jenkins 自动准备 docker 网络，停止并删除旧容器。
       - Jenkins 自动使用仓库中的 docker-compose.yml 构建容器，环境变量相关配置写在Jenkinsfile，无需 .env.prod 文件。

  3. **手动部署**  
     - 在目标服务器上准备 `.env.prod` 文件与 `docker-compose.yml`文件（ docker-compose.yml 直接从仓库获取）。

     - `.env.prod`模版如下：

       ```bash
       # 数据库配置
       DB_USER=xxx
       DB_PASSWORD=xxx
       DB_HOST=xxx
       DB_PORT=xxx
       DB_NAME=xxx
       ```

     - 使用 Docker Compose：
       ```bash
       docker compose pull
       docker compose up -d
       ```
       
     - 访问服务和 Swagger 文档确认部署成功。

## 注意事项

- 每当有新版本需要上线时，请修改`docker-compose.yml`与`Jenkinsfile`中的镜像标签，确保构建和拉取正确的镜像版本。
- 服务器访问国外资源需要配置代理，目前采用的方案是双重代理，即你本地PC代理 + ssh隧道流量转发，服务器的代理端口是7897。
