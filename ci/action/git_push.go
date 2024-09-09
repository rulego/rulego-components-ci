/*
 * Copyright 2024 The RuleGo Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package action

import (
	"github.com/go-git/go-git/v5"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/maps"
	"github.com/rulego/rulego/utils/str"
)

func init() {
	_ = rulego.Registry.Register(&GitPushNode{})
}

// GitPushNodeConfiguration 节点配置
type GitPushNodeConfiguration struct {
	// Git 仓库 URL
	Repository string
	// 推送到的本地目录
	Directory string
	//RefSpecs 用于定义本地分支与远程分支之间的映射关系，例如：refs/heads/your-branch:refs/heads/your-branch，多个映射关系与逗号隔开
	RefSpecs string
	// 认证类型，可以是 "ssh", "password", 或 "token"
	AuthType string
	// 用户名
	AuthUser string
	// 密码或 token
	AuthPassword string
	// SSH 秘钥文件路径
	AuthPemFile string
	// 代理地址
	ProxyUrl string
	// 代理用户名
	ProxyUsername string
	// 代理密码
	ProxyPassword string
}

// GitPushNode 实现 Git 推送
type GitPushNode struct {
	baseGitNode
	// 节点配置
	Config GitPushNodeConfiguration
	hasVar bool
}

// Type 组件类型
func (x *GitPushNode) Type() string {
	return "ci/gitPush"
}

func (x *GitPushNode) New() types.Node {
	return &GitPushNode{Config: GitPushNodeConfiguration{}}
}

// Init 初始化
func (x *GitPushNode) Init(ruleConfig types.Config, configuration types.Configuration) error {
	err := maps.Map2Struct(configuration, &x.Config)
	err = maps.Map2Struct(configuration, &x.baseGitNode.Config)
	if str.CheckHasVar(x.Config.Repository) || str.CheckHasVar(x.Config.Directory) || str.CheckHasVar(x.Config.RefSpecs) {
		x.hasVar = true
	}
	return err
}

// OnMsg 处理消息
func (x *GitPushNode) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	var evn map[string]interface{}
	if x.hasVar {
		evn = base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	}
	refSpecs := x.getRefSpecs(msg, evn)
	workDir := x.getWorkDir(msg, evn)
	msg.Metadata.PutValue(KeyWorkDir, workDir)
	repository := x.getRepository(msg, evn)
	// 打开仓库
	r, err := git.PlainOpen(workDir)
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	// 根据 AuthType 字段的值选择认证方式
	if auth, err := x.getAuthMethod(); err != nil {
		ctx.TellFailure(msg, err)
		return
	} else {
		pushOptions := &git.PushOptions{
			RemoteURL: repository,
			RefSpecs:  refSpecs,
			Auth:      auth,
		}
		// 推送到远程仓库
		if err = r.Push(pushOptions); err != nil {
			ctx.TellFailure(msg, err)
		} else {
			ctx.TellSuccess(msg)
		}
	}
}

// Destroy 销毁
func (x *GitPushNode) Destroy() {
}
