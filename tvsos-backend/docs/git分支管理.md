# git分支管理

> 本仓库为 fleetsim 后端服务仓库，开发人员请根据以下指南进行开发、提交和合并代码。遵循此流程可确保项目顺利协作与持续集成。

## 初始化仓库

1. 在你的电脑本地目录下，新建一个合适的目录，打开你的终端或者`git bash`，执行下面这段命令。

- 如果配置了`ssh`密钥。

  ```bash
  git clone git@github.com:kiritosuki/fleetsim.git
  ```

- 如果没有配置`ssh`密钥。

  ```bash
  git clone https://github.com/kiritosuki/fleetsim.git
  ```

2. 根目录下的`.gitignore`文件会在它当前所在的目录及其所有子目录下生效，如果你在开发过程中需要忽略上传某些文件（比如编译时文件，目标文件，包含隐私信息的文件，以及操作系统相关的配置文件），请在`.gitignore`文件下面依次顺序添加并写好注释。
3. `/fleetsim`即为后端服务项目的根目录，在此进行开发即可，请不要把与后端服务project无关的文件推送到本仓库，否则你的PR将不会被通过

## 分支管理

> 本项目采用 `GitFlow-like` 流程管理分支，确保各个成员协作开发时避免冲突，代码清晰且易于维护。

1. 核心分支：

   - **main**：线上代码，只有经过审核并合并的代码才能进入此分支，禁止直接推送，所有的发布和生产环境部署将会由`develop`分支merge到本分支上。
   - **develop**：开发主分支，保存最新的开发版本，不直接用于部署。
   - **feature**：个人功能开发分支，每个开发者开发新功能时需要从 `develop`分支派生新的 `feature/yourname` 分支。

2. 详细介绍：

    - **main**分支

      该分支是能够部署上线的，没有bug的，阶段性功能完整的代码分支，长期存在，一般来说更新周期较长，大家平时开发不会直接接触，一般由kirito进行维护。该仓库设置main分支禁止直接推送，需要使用PR。

    - **develop**分支

      该分支是开发时期的分支，保存开发的最新版本，长期存在，但你提交代码时不应该在这个分支上提交，这是因为多人开发项目时，在同一个分支上进行开发容易出现代码冲突问题，因此，develop分支也被设置为了禁止直接推送，每个人需要在自己独有的“feature”分支上进行开发，之后通过PR，合并到develop分支。

    - **feature**分支

      feature分支是你每次提交代码时应该所在的分支，命名方式为**feature/<yourname>**。例如**feature/kirito**。yourname请填写英文，在提交时请注意描述清楚你开发的代码功能，该分支允许直接推送，你的代码将会被提交到这个分支上。另外，这个分支每次PR通过并merge到`develop`分支之后，对应的远程分支origin/feature应该被及时删除。

## 提交代码流程

>   在开发过程中，请始终确保你的分支保持与 `develop` 分支同步。

1. 在你的本地创建一个分支，例如feature/kirito

   ```bash
   git branch feature/kirito
   ```

2. 切换到该分支

   ```bash
   git switch feature/kirito
   ```

3. 进行代码开发

4. 将文件添加到暂存区。

   ```bash
   cd yourdir/fleetsim
   git add .
   ```

5. 提交代码，注意把你开发的功能描述完整。

   ```bash
   git commit -m "描述开发的功能"
   ```

6. 推送代码

   ```bash
   git push -u origin feature/kirito
   ```

   注意自己操作时需要把分支改成你自己的名字，-u表示设置本地`feature/kirito`分支与远程分支关联，之后你只需要在该分支上简单使用“git push”，github就会知道你要把当前分支的内容推送到哪里，origin是默认远程仓库名称。

7. 打开github仓库，如果你推送成功，系统在上面会提醒你设置pr，例如：“feature/kirito had recent pushes 9 seconds ago”。此时点击绿色按钮**Compare & pull request**。注意选择目标分支为`develop`，不要申请PR到`main`上。标题就是你commit的内容，如果没有补充内容，直接create pull request即可。

8. 项目设置了通过审核后，才能合并代码到`develop`，所以等待审核即可。

9. 待他人审核通过后，审核者会把你提交的`feature/kirito`分支内容merge到`develop`分支上，然后在远程仓库删除`feature`分支。

10. 删除本地`feature`分支。

    ```bash
    git branch -d feature/<yourname>
    ```

11. 更新`develop`分支。

     ```bash
     git checkout develop
     git pull
     ```

## 自动部署流程

1. **自动构建与推送镜像**：
   - 当 `develop` 分支上的代码通过 PR 合并到 `main` 分支后，自动触发 Webhook，通知 Jenkins 执行构建与部署流程。
   - Jenkins 执行 Dockerfile 生成镜像，并将镜像推送至 Docker Hub。

2. **服务器端部署**：
   - Jenkins 在服务器上拉取最新的 Docker 镜像，并使用 `docker-compose` 完成部署。
