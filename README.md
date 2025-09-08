# 仓库使用方法

> 本仓库为综设项目仓库，由于项目工程时长很长，所以强烈建议各位友友仔细阅读本指南，按照流程开发。

## 初始化仓库

1. 在你的电脑本地目录下，新建一个合适的目录，打开你的终端或者git bash，执行下面这段命令。

- 如果配置了ssh密钥。

  ```
  git clone git@github.com:Myhanshuang/TVSOS.git
  ```

- 如果没有配置ssh密钥。

  ```
  git clone https://github.com/Myhanshuang/TVSOS.git
  ```

2. 请不要修改根目录下的.gitignore文件。.gitignore文件会在它当前所在的目录及其所有子目录下生效，如果你在开发过程中需要忽略上传某些文件（比如编译时文件，目标文件，包含隐私信息的文件）。请在自己的前端/后端项目根目录下配置.gitignore文件（通常在项目初始化时编译器会帮你配置）。
3. 项目开发主要在tvsos-frontend和tvsos-backend这两个目录下，请在这两个目录下创建前端/后端开发项目的根文件（⚠️注意，这两个目录不是你的项目根目录，你的项目根目录应该在这个目录下创建）。

## 分支说明

> git“分支”并非真的分支，大家可以把它理解成一种标记仓库状态的“指针”，并非代表着真实存在的“分支树”。

1. 核心分支主要有三个：

   - main（长期存在）
   - develop（长期存在）
   - feature（短生命周期）

   其中main分支与develop分支已经创建好，并已经关联了远程分支，克隆仓库之后，你的分支状态就与远程仓库同步了。

   feature分支需要你自己创建，以后你开发功能大概率会在这个分支上进行开发。

2. 详细介绍：

   - main分支

     该分支是能够部署上线的，没有bug的，阶段性功能完整的代码分支，长期存在，一般来说更新周期较长，大家平时开发不会直接接触。该仓库设置main分支禁止直接推送，需要使用pr。（pr在后面有介绍，不熟悉的跳转到后面看）

   - develop分支

     该分支是开发时期的分支，保存开发的最新版本，长期存在，但你提交代码时不应该在这个分支上提交，这是因为多人开发项目时，在同一个分支上进行开发容易出现代码冲突问题，因此，develop分支也被设置为了禁止直接推送，每个人需要在自己独有的“feature”分支上进行开发，之后通过pr，合并到develop分支。

   - feature分支

     feature分支是你每次提交代码时应该所在的分支，命名方式为**feature/backend-<yourname>**或者**feature/frontend-<yourname>**。例如**feature/frontend-kirito**。yourname请填写英文，在提交时请注意描述清楚你开发的代码功能，该分支允许直接推送，你的代码将会被提交到这个分支上。另外，这个分支每次在pr之后，对应的远程分支origin/feature就会被删除。

## 提交代码流程

>   在提交之前，请先切换到develop分支，使用“git pull”命令 保证你的代码是最新状态。

1. 在你的本地创建一个分支，例如feature/backend-kirito

   ```
   git branch feature/backend-kirito
   ```

2. 切换到该分支

   ```
   git switch feature/backend-kirito
   ```

3. 进行代码开发，例如，我在后端项目目录/tvsos-backend下新建了readme.md

4. 切换项目根目录，将文件添加到暂存区。

   ```
   cd tvsos的根目录
   git add .
   ```

5. 提交代码，注意把你开发的功能描述完整。

   ```
   git commit -m "描述开发的功能"
   ```

6. 推送代码

   ```
   git push -u origin feature/backend-kirito
   ```

   注意自己操作时需要把分支改成你自己的名字，-u表示设置本地feature/backend-kirito分支与远程分支关联，之后你只需要在该分支上简单使用“git push”，github就会知道你要把当前分支的内容推送到哪里，origin是默认远程仓库名称。

7. 打开github仓库，如果你推送成功，系统在上面会提醒你设置pr，例如：“feature/backend-kirito had recent pushes 9 seconds ago”。此时点击绿色按钮**Compare & pull request**。标题就是你commit的内容，如果没有补充内容，直接create pull request即可。

8. 项目设置了至少有一人审核，你才能合并代码到main，所以等待审核即可。

9. 待他人审核通过后，审核者会把你提交的feature/backend-kirito分支内容merge合并到develop分支上，然后在远程仓库删除feature分支，前面说过这个分支是一个短生命周期分支，此时你有两种选择。不管怎样，你都要更新develop分支，因为现在你本地的develop分支并没有feature的最新代码，最便捷的更新方法就是在审核通过后切换到develop分支拉取更新。

   - 一种是本地不删除这个分支，后续继续在这个分支上开发，但需要维护这个分支保持develop的最新状态。

      切换到develop分支，拉取最新状态。

      ```
      git branch develop
      git pull
      ```

      合并这两个分支，由于develop分支比feature分支更新，所以git会简单的把feature分支移动到develop的状态，此时你应该在develop分支上。

      ```
      git merge feature/backend-kirito
      ```
   
      此时你的develop分支和feature分支都是最新状态了。
   
   - 另外一种是本地删除这个feature分支，之后更新develop分支。
   
     在删除当前分支之前，你需要先切换到别的分支。
     
     ```
     git branch develop
     git branch -d feature/backend-kirito
     ```
     
     此时你切换到了develop分支，删除了feature分支，之后再更新develop分支。
     
     -d表示安全删除，如果你的feature分支还没有合并到develop就删除，git会报错并阻止你。
     
     ```
     git pull
     ```
     
     如果你下次再想要提交代码，重新创建feature分支即可，即用即删，无需维护。
   

